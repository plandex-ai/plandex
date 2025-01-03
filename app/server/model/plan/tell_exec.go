package plan

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/types"

	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func Tell(clients map[string]*openai.Client, plan *db.Plan, branch string, auth *types.ServerAuth, req *shared.TellPlanRequest) error {
	log.Printf("Tell: Called with plan ID %s on branch %s\n", plan.Id, branch)

	_, err := activatePlan(
		clients,
		plan,
		branch,
		auth,
		req.Prompt,
		false,
		req.AutoContext,
	)

	if err != nil {
		log.Printf("Error activating plan: %v\n", err)
		return err
	}

	go execTellPlan(
		clients,
		plan,
		branch,
		auth,
		req,
		0,
		"",
		!req.IsChatOnly && req.BuildMode == shared.BuildModeAuto,
		0,
	)

	log.Printf("Tell: Tell operation completed successfully for plan ID %s on branch %s\n", plan.Id, branch)
	return nil
}

func execTellPlan(
	clients map[string]*openai.Client,
	plan *db.Plan,
	branch string,
	auth *types.ServerAuth,
	req *shared.TellPlanRequest,
	iteration int,
	missingFileResponse shared.RespondMissingFileChoice,
	shouldBuildPending bool,
	numErrorRetry int,
) {
	log.Printf("execTellPlan: Called for plan ID %s on branch %s, iteration %d\n", plan.Id, branch, iteration)
	currentUserId := auth.User.Id
	currentOrgId := auth.OrgId

	active := GetActivePlan(plan.Id, branch)

	if active == nil {
		log.Printf("execTellPlan: Active plan not found for plan ID %s on branch %s\n", plan.Id, branch)
		return
	}

	if missingFileResponse == "" {
		log.Println("Executing WillExecPlanHook")
		_, apiErr := hooks.ExecHook(hooks.WillExecPlan, hooks.HookParams{
			Auth: auth,
			Plan: plan,
		})

		if apiErr != nil {
			time.Sleep(100 * time.Millisecond)
			active.StreamDoneCh <- apiErr
			return
		}
	}

	planId := plan.Id
	log.Println("execTellPlan - Setting plan status to replying")
	err := db.SetPlanStatus(planId, branch, shared.PlanStatusReplying, "")
	if err != nil {
		log.Printf("Error setting plan %s status to replying: %v\n", planId, err)
		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error setting plan status to replying",
		}

		log.Printf("execTellPlan: execTellPlan operation completed for plan ID %s on branch %s, iteration %d\n", plan.Id, branch, iteration)
		return
	}
	log.Println("execTellPlan - Plan status set to replying")

	state := &activeTellStreamState{
		clients:                clients,
		req:                    req,
		auth:                   auth,
		currentOrgId:           currentOrgId,
		currentUserId:          currentUserId,
		plan:                   plan,
		branch:                 branch,
		iteration:              iteration,
		missingFileResponse:    missingFileResponse,
		currentReplyNumRetries: numErrorRetry,
	}

	log.Println("execTellPlan - Loading tell plan")
	err = state.loadTellPlan()
	if err != nil {
		return
	}
	log.Println("execTellPlan - Tell plan loaded")

	if iteration == 0 && missingFileResponse == "" {
		UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
			ap.Contexts = state.modelContext

			for _, context := range state.modelContext {
				if context.FilePath != "" {
					ap.ContextsByPath[context.FilePath] = context
				}
			}
		})
	} else if missingFileResponse == "" {
		// reset current reply content and num tokens
		UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
			ap.CurrentReplyContent = ""
			ap.NumTokens = 0
		})
	}

	// if any skipped paths have since been added to context, remove them from skipped paths
	if len(active.SkippedPaths) > 0 {
		var toUnskipPaths []string
		for contextPath := range active.ContextsByPath {
			if active.SkippedPaths[contextPath] {
				toUnskipPaths = append(toUnskipPaths, contextPath)
			}
		}
		if len(toUnskipPaths) > 0 {
			UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
				for _, path := range toUnskipPaths {
					delete(ap.SkippedPaths, path)
				}
			})
		}
	}

	autoContextEnabled := req.AutoContext && state.hasContextMap
	smartContextEnabled := req.AutoContext
	isPlanningStage := req.IsChatOnly || (!req.IsUserContinue && (iteration == 0 || (autoContextEnabled && iteration == 1)))
	isImplementationStage := !isPlanningStage
	isContextStage := isPlanningStage && state.hasContextMap && !state.contextMapEmpty && iteration == 0

	log.Printf("isPlanningStage: %t, isImplementationStage: %t, isContextStage: %t\n", isPlanningStage, isImplementationStage, isContextStage)

	// if auto context is enabled, we only include maps and trees on the first iteration, which is the context-gathering step, and the second iteration, which is the planning step
	var (
		includeMaps = true
	)
	if req.AutoContext && iteration > 1 {
		includeMaps = false
	}

	modelContextText, modelContextTokens, err := state.formatModelContext(includeMaps, true, isImplementationStage, smartContextEnabled, req.ExecEnabled)
	if err != nil {
		err = fmt.Errorf("error formatting model modelContext: %v", err)
		log.Println(err)

		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error formatting model modelContext",
		}
		return
	}

	var sysCreate string

	var sysCreateTokens int

	if isPlanningStage {
		log.Println("isPlanningStage")
		if autoContextEnabled && isContextStage {
			sysCreate = prompts.AutoContextPreamble
			sysCreateTokens = prompts.AutoContextPreambleNumTokens
		} else if autoContextEnabled || smartContextEnabled {
			sysCreate = prompts.SysPlanningAutoContext
			sysCreateTokens = prompts.SysPlanningAutoContextTokens
		} else {
			sysCreate = prompts.SysPlanningBasic
			sysCreateTokens = prompts.SysPlanningBasicTokens
		}
	} else {
		log.Println("isImplementationStage")
		if state.currentSubtask == nil {
			err = fmt.Errorf("no current subtask")
			log.Println(err)
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "No current subtask",
			}
			return
		}
		sysCreate = prompts.GetImplementationPrompt(state.currentSubtask.Title)
	}

	if !isContextStage {
		if req.ExecEnabled {
			sysCreate += prompts.ApplyScriptPrompt
			sysCreateTokens += prompts.ApplyScriptPromptNumTokens
		} else {
			sysCreate += prompts.NoApplyScriptPrompt
			sysCreateTokens += prompts.NoApplyScriptPromptNumTokens
		}
	}

	// log.Println("sysCreate before context:\n", sysCreate)

	sysCreate += modelContextText

	if len(active.SkippedPaths) > 0 {
		skippedPrompt := prompts.SkippedPathsPrompt
		for skippedPath := range active.SkippedPaths {
			skippedPrompt += fmt.Sprintf("- %s\n", skippedPath)
		}

		numTokens, err := shared.GetNumTokens(skippedPrompt)
		if err != nil {
			log.Printf("Error getting num tokens for skipped paths: %v\n", err)
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Error getting num tokens for skipped paths",
			}
			return
		}

		sysCreateTokens += numTokens
	}

	if len(state.subtasks) > 0 {
		subtasksPrompt, subtaskTokens, err := state.formatSubtasks()

		if err != nil {
			err = fmt.Errorf("error formatting subtasks: %v", err)
			log.Println(err)
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Error formatting subtasks",
			}
			return
		}

		log.Println("subtasksPrompt:\n", subtasksPrompt)

		sysCreate += subtasksPrompt
		sysCreateTokens += subtaskTokens
	}

	// log.Println("**sysCreate:**\n", sysCreate)

	state.messages = []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: sysCreate,
		},
	}

	// Add a separate message for image contexts
	for _, context := range state.modelContext {
		if context.ContextType == shared.ContextImageType {
			if !state.settings.ModelPack.Planner.BaseModelConfig.HasImageSupport {
				err = fmt.Errorf("%s does not support images in context", state.settings.ModelPack.Planner.BaseModelConfig.ModelName)
				log.Println(err)
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusBadRequest,
					Msg:    "Model does not support images in context",
				}
				return
			}

			imageMessage := openai.ChatCompletionMessage{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: fmt.Sprintf("Image: %s", context.Name),
					},
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL:    shared.GetImageDataURI(context.Body, context.FilePath),
							Detail: context.ImageDetail,
						},
					},
				},
			}
			state.messages = append(state.messages, imageMessage)
		}
	}

	osDetailsTokens, err := shared.GetNumTokens(req.OsDetails)
	if err != nil {
		log.Printf("Error getting num tokens for os details: %v\n", err)
		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error getting num tokens for os details",
		}
	}

	var (
		numPromptTokens int
		promptTokens    int
	)

	var wrapperTokens int
	if isPlanningStage {
		wrapperTokens = prompts.PlanningPromptWrapperTokens
	} else {
		wrapperTokens = prompts.ImplementationPromptWrapperTokens
	}

	if iteration == 0 && missingFileResponse == "" {
		numPromptTokens, err = shared.GetNumTokens(req.Prompt)
		if err != nil {
			err = fmt.Errorf("error getting number of tokens in prompt: %v", err)
			log.Println(err)
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Error getting number of tokens in prompt",
			}
			return
		}

		promptTokens = wrapperTokens + numPromptTokens + osDetailsTokens
	} else if iteration > 0 && missingFileResponse == "" {
		numPromptTokens = prompts.AutoContinuePromptTokens
		promptTokens = wrapperTokens + numPromptTokens + osDetailsTokens
	}

	if req.ExecEnabled {
		promptTokens += prompts.ApplyScriptSummaryNumTokens
	}

	state.tokensBeforeConvo = sysCreateTokens + modelContextTokens + state.latestSummaryTokens + promptTokens

	// print out breakdown of token usage
	log.Printf("System message tokens: %d\n", sysCreateTokens)
	log.Printf("Context tokens: %d\n", modelContextTokens)
	log.Printf("Prompt tokens: %d\n", promptTokens)
	log.Printf("Latest summary tokens: %d\n", state.latestSummaryTokens)
	log.Printf("Total tokens before convo: %d\n", state.tokensBeforeConvo)

	if state.tokensBeforeConvo > state.settings.GetPlannerEffectiveMaxTokens() {
		// token limit already exceeded before adding conversation
		err := fmt.Errorf("token limit exceeded before adding conversation")
		log.Printf("Error: %v\n", err)
		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Token limit exceeded before adding conversation",
		}
		return
	}

	if !state.injectSummariesAsNeeded() {
		return
	}

	var applyScriptSummary string
	if req.ExecEnabled {
		applyScriptSummary = prompts.ApplyScriptPromptSummary
	}

	state.replyId = uuid.New().String()
	state.replyParser = types.NewReplyParser()

	if missingFileResponse == "" {
		var promptMessage *openai.ChatCompletionMessage
		if req.IsUserContinue {
			if len(state.messages) == 0 {
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeContinueNoMessages,
					Status: http.StatusBadRequest,
					Msg:    "No messages yet. Can't continue plan.",
				}
				return
			}

			// if the user is continuing the plan, we need to check whether the previous message was a user message or assistant message
			lastMessage := state.messages[len(state.messages)-1]

			log.Println("User is continuing plan. Last message role:", lastMessage.Role)
			// log.Println("User is continuing plan. Last message:\n\n", lastMessage.Content)

			if lastMessage.Role == openai.ChatMessageRoleUser {
				// if last message was a user message, we want to remove it from the messages array and then use that last message as the prompt so we can continue from where the user left off

				log.Println("User is continuing plan. Last message was user message. Using last user message as prompt")

				state.messages = state.messages[:len(state.messages)-1]
				promptMessage = &openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: prompts.GetWrappedPrompt(lastMessage.Content, req.OsDetails, applyScriptSummary, isPlanningStage),
				}

				state.userPrompt = lastMessage.Content
			} else {

				// if the last message was an assistant message, we'll use the user continue prompt
				log.Println("User is continuing plan. Last message was assistant message. Using user continue prompt")

				// otherwise we'll use the continue prompt
				promptMessage = &openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: prompts.GetWrappedPrompt(prompts.UserContinuePrompt, req.OsDetails, applyScriptSummary, isPlanningStage),
				}
			}

			state.messages = append(state.messages, *promptMessage)
		} else {
			var prompt string
			if iteration == 0 {
				if req.IsChatOnly {
					prompt = req.Prompt + prompts.ChatOnlyPrompt
					state.totalRequestTokens += prompts.ChatOnlyPromptTokens
				} else if req.IsUserDebug {
					prompt = req.Prompt + prompts.DebugPrompt
					state.totalRequestTokens += prompts.DebugPromptTokens
				} else if req.IsApplyDebug {
					prompt = req.Prompt + prompts.ApplyDebugPrompt
					state.totalRequestTokens += prompts.ApplyDebugPromptTokens
				} else {
					prompt = req.Prompt
				}
			} else {
				prompt = prompts.AutoContinuePrompt
			}

			state.userPrompt = prompt

			var finalPrompt string
			if isContextStage {
				finalPrompt = prompt
			} else {
				finalPrompt = prompts.GetWrappedPrompt(prompt, req.OsDetails, applyScriptSummary, isPlanningStage)
			}

			promptMessage = &openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: finalPrompt,
			}
		}

		state.promptMessage = promptMessage
		state.messages = append(state.messages, *promptMessage)
	} else {
		log.Println("Missing file response:", missingFileResponse, "setting replyParser")
		// log.Printf("Current reply content:\n%s\n", active.CurrentReplyContent)

		state.replyParser.AddChunk(active.CurrentReplyContent, true)
		res := state.replyParser.Read()
		currentFile := res.CurrentFilePath

		log.Printf("Current file: %s\n", currentFile)
		// log.Println("Current reply content:\n", active.CurrentReplyContent)

		replyContent := active.CurrentReplyContent
		numTokens := active.NumTokens

		if missingFileResponse == shared.RespondMissingFileChoiceSkip {
			replyBeforeCurrentFile := state.replyParser.GetReplyBeforeCurrentPath()
			numTokens, err = shared.GetNumTokens(replyBeforeCurrentFile)
			if err != nil {
				log.Printf("Error getting num tokens for reply before current file: %v\n", err)
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Error getting num tokens for reply before current file",
				}
				return
			}

			replyContent = replyBeforeCurrentFile
			state.replyParser = types.NewReplyParser()
			state.replyParser.AddChunk(replyContent, true)

			UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
				ap.CurrentReplyContent = replyContent
				ap.NumTokens = numTokens
				ap.SkippedPaths[currentFile] = true
			})

		} else {
			if missingFileResponse == shared.RespondMissingFileChoiceOverwrite {
				UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
					ap.AllowOverwritePaths[currentFile] = true
				})
			}
		}

		state.messages = append(state.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: active.CurrentReplyContent,
		})

		if missingFileResponse == shared.RespondMissingFileChoiceSkip {
			res := state.replyParser.FinishAndRead()
			skipPrompt := prompts.GetSkipMissingFilePrompt(res.CurrentFilePath)
			prompt := prompts.GetWrappedPrompt(skipPrompt, req.OsDetails, applyScriptSummary, isPlanningStage) + "\n\n" + skipPrompt // repetition of skip prompt to improve instruction following

			skipPromptTokens, err := shared.GetNumTokens(skipPrompt)
			if err != nil {
				log.Printf("Error getting num tokens for skip prompt: %v\n", err)
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Error getting num tokens for skip prompt",
				}
				return
			}

			state.totalRequestTokens += skipPromptTokens

			state.messages = append(state.messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			})

		} else {
			missingPrompt := prompts.GetMissingFileContinueGeneratingPrompt(res.CurrentFilePath)
			prompt := prompts.GetWrappedPrompt(missingPrompt, req.OsDetails, applyScriptSummary, isPlanningStage) + "\n\n" + missingPrompt // repetition of missing prompt to improve instruction following

			promptTokens, err = shared.GetNumTokens(prompt)
			if err != nil {
				log.Printf("Error getting num tokens for missing file continue prompt: %v\n", err)
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Error getting num tokens for missing file continue prompt",
				}
				return
			}

			state.totalRequestTokens += promptTokens

			state.messages = append(state.messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			})
		}
	}

	log.Printf("\n\nMessages: %d\n", len(state.messages))
	// for _, message := range state.messages {
	// 	log.Printf("%s: %s\n", message.Role, message.Content)
	// }

	// ts := time.Now().Format("2006-01-02-150405")
	// os.WriteFile(fmt.Sprintf("generations/messages-%s.txt", ts), []byte(spew.Sdump(state.messages)), 0644)

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		Auth: auth,
		Plan: plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  state.totalRequestTokens,
			OutputTokens: state.settings.ModelPack.Planner.ReservedOutputTokens,
			ModelName:    state.settings.ModelPack.Planner.BaseModelConfig.ModelName,
		},
	})
	if apiErr != nil {
		active.StreamDoneCh <- apiErr
		return
	}

	modelReq := openai.ChatCompletionRequest{
		Model:    state.settings.ModelPack.Planner.BaseModelConfig.ModelName,
		Messages: state.messages,
		Stream:   true,
		StreamOptions: &openai.StreamOptions{
			IncludeUsage: true,
		},
		Temperature: state.settings.ModelPack.Planner.Temperature,
		TopP:        state.settings.ModelPack.Planner.TopP,
		Stop:        []string{"<PlandexSubtaskDone/>"},
	}

	envVar := state.settings.ModelPack.Planner.BaseModelConfig.ApiKeyEnvVar
	client := clients[envVar]

	stream, err := model.CreateChatCompletionStreamWithRetries(client, active.ModelStreamCtx, modelReq)
	if err != nil {
		log.Printf("Error starting reply stream: %v\n", err)

		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error starting reply stream: " + err.Error(),
		}
		return
	}

	if shouldBuildPending {
		go func() {
			pendingBuildsByPath, err := active.PendingBuildsByPath(auth.OrgId, auth.User.Id, state.convo)

			if err != nil {
				log.Printf("Error getting pending builds by path: %v\n", err)
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Error getting pending builds by path",
				}
				return
			}

			if len(pendingBuildsByPath) == 0 {
				log.Println("Tell plan: no pending builds")
				return
			}

			log.Printf("Tell plan: found %d pending builds\n", len(pendingBuildsByPath))
			// spew.Dump(pendingBuildsByPath)

			buildState := &activeBuildStreamState{
				tellState:     state,
				clients:       clients,
				auth:          auth,
				currentOrgId:  currentOrgId,
				currentUserId: currentUserId,
				plan:          plan,
				branch:        branch,
				settings:      state.settings,
				modelContext:  state.modelContext,
			}

			for _, pendingBuilds := range pendingBuildsByPath {
				buildState.queueBuilds(pendingBuilds)
			}
		}()
	}

	UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
		ap.CurrentStreamingReplyId = state.replyId
		ap.CurrentReplyDoneCh = make(chan bool, 1)
	})

	go state.listenStream(stream)
}
