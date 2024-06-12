package plan

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"plandex-server/db"
	"plandex-server/model"
	"plandex-server/model/lib"
	"plandex-server/model/prompts"
	"plandex-server/types"

	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func Tell(clients map[string]*openai.Client, plan *db.Plan, branch string, auth *types.ServerAuth, req *shared.TellPlanRequest) error {
	log.Printf("Tell: Called with plan ID %s on branch %s\n", plan.Id, branch)

	_, err := activatePlan(clients, plan, branch, auth, req.Prompt, false)

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
		req.BuildMode == shared.BuildModeAuto,
		"",
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
	autoContinueNextTask string,
) {
	log.Printf("execTellPlan: Called for plan ID %s on branch %s, iteration %d\n", plan.Id, branch, iteration)
	currentUserId := auth.User.Id
	currentOrgId := auth.OrgId

	active := GetActivePlan(plan.Id, branch)

	if active == nil {
		log.Printf("execTellPlan: Active plan not found for plan ID %s on branch %s\n", plan.Id, branch)
		return
	}

	if os.Getenv("IS_CLOUD") != "" &&
		missingFileResponse == "" {
		log.Println("execTellPlan: IS_CLOUD environment variable is set")
		if auth.User.IsTrial {
			if plan.TotalReplies >= types.TrialMaxReplies {
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeTrialMessagesExceeded,
					Status: http.StatusForbidden,
					Msg:    "Anonymous trial message limit exceeded",
					TrialMessagesExceededError: &shared.TrialMessagesExceededError{
						MaxReplies: types.TrialMaxReplies,
					},
				}
				return
			}
		}
	}

	planId := plan.Id
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

	state := &activeTellStreamState{
		clients:             clients,
		req:                 req,
		auth:                auth,
		currentOrgId:        currentOrgId,
		currentUserId:       currentUserId,
		plan:                plan,
		branch:              branch,
		iteration:           iteration,
		missingFileResponse: missingFileResponse,
	}

	err = state.loadTellPlan()
	if err != nil {
		return
	}

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

	modelContextText, modelContextTokens, err := lib.FormatModelContext(state.modelContext)
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

	systemMessageText := prompts.SysCreate + modelContextText
	systemMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemMessageText,
	}

	if len(active.SkippedPaths) > 0 {
		systemMessageText += prompts.SkippedPathsPrompt
		for skippedPath := range active.SkippedPaths {
			systemMessageText += fmt.Sprintf("- %s\n", skippedPath)
		}
	}

	state.messages = []openai.ChatCompletionMessage{
		systemMessage,
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

	var (
		numPromptTokens int
		promptTokens    int
	)
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
		promptTokens = prompts.PromptWrapperTokens + numPromptTokens
	}

	state.tokensBeforeConvo = prompts.CreateSysMsgNumTokens + modelContextTokens + state.latestSummaryTokens + promptTokens

	// print out breakdown of token usage
	log.Printf("System message tokens: %d\n", prompts.CreateSysMsgNumTokens)
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

			log.Println("User is continuing plan. Last message:\n\n", lastMessage.Content)

			if lastMessage.Role == openai.ChatMessageRoleUser {
				// if last message was a user message, we want to remove it from the messages array and then use that last message as the prompt so we can continue from where the user left off

				log.Println("User is continuing plan. Last message was user message. Using last user message as prompt")

				state.messages = state.messages[:len(state.messages)-1]
				promptMessage = &openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: prompts.GetWrappedPrompt(lastMessage.Content),
				}

				state.userPrompt = lastMessage.Content
			} else {

				// if the last message was an assistant message, we'll use the user continue prompt
				log.Println("User is continuing plan. Last message was assistant message. Using user continue prompt")

				// otherwise we'll use the continue prompt
				promptMessage = &openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: prompts.GetWrappedPrompt(prompts.UserContinuePrompt),
				}
			}

			state.messages = append(state.messages, *promptMessage)
		} else {
			var prompt string
			if iteration == 0 {
				prompt = req.Prompt
			} else {
				prompt = prompts.AutoContinuePrompt

				if autoContinueNextTask != "" {
					prompt += `
					Here is the next task:
				
					` + autoContinueNextTask + `
					
					Continue seamlessly with this task.
				`
				}
			}

			state.userPrompt = prompt

			promptMessage = &openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: prompts.GetWrappedPrompt(prompt),
			}
		}

		state.promptMessage = promptMessage
		state.messages = append(state.messages, *promptMessage)
	} else {
		log.Println("Missing file response:", missingFileResponse, "setting replyParser")

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
			res := state.replyParser.Read()

			state.messages = append(state.messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: prompts.GetSkipMissingFilePrompt(res.CurrentFilePath),
			})

		} else {
			state.messages = append(state.messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: prompts.MissingFileContinueGeneratingPrompt,
			})
		}
	}

	log.Printf("\n\nMessages: %d\n", len(state.messages))
	// for _, message := range state.messages {
	// 	log.Printf("%s: %s\n", message.Role, message.Content)
	// }

	modelReq := openai.ChatCompletionRequest{
		Model:       state.settings.ModelPack.Planner.BaseModelConfig.ModelName,
		Messages:    state.messages,
		Stream:      true,
		Temperature: state.settings.ModelPack.Planner.Temperature,
		TopP:        state.settings.ModelPack.Planner.TopP,
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
			// spew.Dump(pendingBuildsByPath)x

			buildState := &activeBuildStreamState{
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
