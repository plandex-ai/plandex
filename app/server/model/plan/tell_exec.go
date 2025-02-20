package plan

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/types"

	shared "plandex-shared"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

func Tell(clients map[string]model.ClientInfo, plan *db.Plan, branch string, auth *types.ServerAuth, req *shared.TellPlanRequest) error {
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

	go execTellPlan(execTellPlanParams{
		clients:            clients,
		plan:               plan,
		branch:             branch,
		auth:               auth,
		req:                req,
		iteration:          0,
		shouldBuildPending: !req.IsChatOnly && req.BuildMode == shared.BuildModeAuto,
	})

	log.Printf("Tell: Tell operation completed successfully for plan ID %s on branch %s\n", plan.Id, branch)
	return nil
}

type execTellPlanParams struct {
	clients                    map[string]model.ClientInfo
	plan                       *db.Plan
	branch                     string
	auth                       *types.ServerAuth
	req                        *shared.TellPlanRequest
	iteration                  int
	missingFileResponse        shared.RespondMissingFileChoice
	shouldBuildPending         bool
	numErrorRetry              int
	unfinishedSubtaskReasoning string
}

func execTellPlan(params execTellPlanParams) {
	clients := params.clients
	plan := params.plan
	branch := params.branch
	auth := params.auth
	req := params.req
	iteration := params.iteration
	missingFileResponse := params.missingFileResponse
	shouldBuildPending := params.shouldBuildPending
	unfinishedSubtaskReasoning := params.unfinishedSubtaskReasoning

	log.Printf("[TellExec] Starting iteration %d for plan %s on branch %s", iteration, plan.Id, branch)
	currentUserId := auth.User.Id
	currentOrgId := auth.OrgId

	active := GetActivePlan(plan.Id, branch)

	if active == nil {
		log.Printf("execTellPlan: Active plan not found for plan ID %s on branch %s\n", plan.Id, branch)
		return
	}

	// Load existing subtasks to log their state
	// subtasks, err := db.GetPlanSubtasks(currentOrgId, plan.Id)
	// if err != nil {
	// 	log.Printf("[TellExec] Error loading subtasks: %v", err)
	// } else {
	// 	var unfinished []string
	// 	var finished []string
	// 	for _, task := range subtasks {
	// 		if task.IsFinished {
	// 			finished = append(finished, task.Title)
	// 		} else {
	// 			unfinished = append(unfinished, task.Title)
	// 		}
	// 	}
	// 	log.Printf("[TellExec] Current subtask state - Total: %d, Finished: %d, Unfinished: %d", len(subtasks), len(finished), len(unfinished))
	// 	log.Printf("[TellExec] Finished tasks: %v", finished)
	// 	log.Printf("[TellExec] Unfinished tasks: %v", unfinished)
	// 	log.Printf("[TellExec] Unfinished subtask reasoning: %s", unfinishedSubtaskReasoning)
	// }

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
		execTellPlanParams:  params,
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

	log.Println("execTellPlan - Loading tell plan")
	err = state.loadTellPlan()
	if err != nil {
		return
	}
	log.Println("execTellPlan - Tell plan loaded")

	activatedPaths := state.resolveCurrentStage()

	var basicContextMsg *types.ExtendedChatMessagePart
	var autoContextMsg *types.ExtendedChatMessagePart
	var smartContextMsg *types.ExtendedChatMessagePart

	if state.currentStage.TellStage == shared.TellStageImplementation {
		smartContextMsg = state.formatModelContext(formatModelContextParams{
			includeMaps:         false,
			smartContextEnabled: req.SmartContext,
			execEnabled:         req.ExecEnabled,
		})
	} else if state.currentStage.TellStage == shared.TellStagePlanning {
		basicContextMsg = state.formatModelContext(formatModelContextParams{
			includeMaps:         true,
			smartContextEnabled: req.SmartContext,
			execEnabled:         req.ExecEnabled,
			basicOnly:           true,
			cache:               true,
		})

		if state.currentStage.PlanningPhase == shared.PlanningPhasePlanning {
			if req.AutoContext {
				autoContextMsg = state.formatModelContext(formatModelContextParams{
					includeMaps:         false,
					smartContextEnabled: req.SmartContext,
					execEnabled:         req.ExecEnabled,
					activeOnly:          true,
					activatedPaths:      activatedPaths,
				})
			} else {
				// if auto context is disabled, just dump in everything, both basic and auto, that may have accumulated
				autoContextMsg = state.formatModelContext(formatModelContextParams{
					includeMaps:         false,
					smartContextEnabled: req.SmartContext,
					execEnabled:         req.ExecEnabled,
					autoOnly:            true,
				})
			}
		}
	}

	getTellSysPromptParams := getTellSysPromptParams{
		autoContextEnabled:  state.currentStage.TellStage == shared.TellStagePlanning && state.currentStage.PlanningPhase == shared.PlanningPhaseContext,
		smartContextEnabled: req.SmartContext,
		basicContextMsg:     basicContextMsg,
		autoContextMsg:      autoContextMsg,
		smartContextMsg:     smartContextMsg,
	}
	sysParts, err := state.getTellSysPrompt(getTellSysPromptParams)
	if err != nil {
		log.Printf("Error getting tell sys prompt: %v\n", err)
		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    err.Error(),
		}
		return
	}

	// log.Println("**sysPrompt:**\n", spew.Sdump(sysParts))

	state.messages = []types.ExtendedChatMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: sysParts,
		},
	}

	// Add a separate message for image contexts
	imageContextTokens, ok := state.addImageContext()
	if !ok {
		return
	}

	promptMessage, ok := state.resolvePromptMessage(unfinishedSubtaskReasoning)
	if !ok {
		return
	}

	// log.Println("promptMessage:", spew.Sdump(promptMessage))

	state.tokensBeforeConvo =
		model.GetMessagesTokenEstimate(state.messages...) +
			model.GetMessagesTokenEstimate(*promptMessage) +
			state.latestSummaryTokens +
			imageContextTokens +
			model.TokensPerRequest

	// print out breakdown of token usage
	log.Printf("Image context tokens: %d\n", imageContextTokens)
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

	if !state.addConversationMessages() {
		return
	}

	state.replyId = uuid.New().String()
	state.replyParser = types.NewReplyParser()

	if missingFileResponse == "" {
		state.messages = append(state.messages, *promptMessage)
	} else if !state.handleMissingFileResponse(unfinishedSubtaskReasoning) {
		return
	}

	log.Printf("\n\nMessages: %d\n", len(state.messages))
	// for _, message := range state.messages {
	// 	log.Printf("%s: %s\n", message.Role, message.Content)
	// }

	requestTokens := model.GetMessagesTokenEstimate(state.messages...) + imageContextTokens + model.TokensPerRequest
	state.totalRequestTokens = requestTokens

	stop := []string{"<PlandexFinish/>"}
	var modelConfig shared.ModelRoleConfig
	if state.currentStage.TellStage == shared.TellStagePlanning {
		plannerConfig := state.settings.ModelPack.Planner.GetRoleForTokens(requestTokens)
		modelConfig = plannerConfig.ModelRoleConfig
		if state.currentStage.PlanningPhase == shared.PlanningPhaseContext {
			log.Println("Tell plan - isContextStage - setting modelConfig to context loader")
			modelConfig = state.settings.ModelPack.GetArchitect().GetRoleForInputTokens(requestTokens)
		}
	} else if state.currentStage.TellStage == shared.TellStageImplementation {
		modelConfig = state.settings.ModelPack.GetCoder().GetRoleForInputTokens(requestTokens)
	}

	log.Println("totalRequestTokens:", requestTokens)

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		Auth: auth,
		Plan: plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  requestTokens,
			OutputTokens: modelConfig.GetReservedOutputTokens(),
			ModelName:    modelConfig.BaseModelConfig.ModelName,
		},
	})
	if apiErr != nil {
		active.StreamDoneCh <- apiErr
		return
	}

	// log.Println("Stop:", stop)
	// spew.Dump(state.messages)

	log.Println("modelConfig:", spew.Sdump(modelConfig))

	modelReq := types.ExtendedChatCompletionRequest{
		Model:    modelConfig.BaseModelConfig.ModelName,
		Messages: state.messages,
		Stream:   true,
		StreamOptions: &openai.StreamOptions{
			IncludeUsage: true,
		},
		Temperature: modelConfig.Temperature,
		TopP:        modelConfig.TopP,
		Stop:        stop,
	}

	state.requestStartedAt = time.Now()
	state.originalReq = &modelReq
	state.modelConfig = &modelConfig

	// output the modelReq to a json file
	// if jsonData, err := json.MarshalIndent(modelReq, "", "  "); err == nil {
	// 	timestamp := time.Now().Format("2006-01-02-150405")
	// 	filename := fmt.Sprintf("generations/model-request-%s.json", timestamp)
	// 	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
	// 		log.Printf("Error writing model request to file: %v\n", err)
	// 	}
	// } else {
	// 	log.Printf("Error marshaling model request to JSON: %v\n", err)
	// }

	stream, err := model.CreateChatCompletionStreamWithRetries(clients, &modelConfig, active.ModelStreamCtx, modelReq)
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
		go state.queuePendingBuilds()
	}

	UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
		ap.CurrentStreamingReplyId = state.replyId
		ap.CurrentReplyDoneCh = make(chan bool, 1)
	})

	go state.listenStream(stream)
}
