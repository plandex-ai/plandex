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

	autoContextEnabled := req.AutoContext && state.hasContextMap
	smartContextEnabled := req.AutoContext
	isPlanningStage := req.IsChatOnly || (!req.IsUserContinue && (iteration == 0 || (autoContextEnabled && iteration == 1)))
	isImplementationStage := !isPlanningStage
	isContextStage := isPlanningStage && autoContextEnabled && !state.contextMapEmpty && iteration == 0

	log.Printf("isPlanningStage: %t, isImplementationStage: %t, isContextStage: %t\n", isPlanningStage, isImplementationStage, isContextStage)

	state.isPlanningStage = isPlanningStage
	state.isImplementationStage = isImplementationStage
	state.isContextStage = isContextStage

	// if auto context is enabled, we only include maps on the first iteration, which is the context-gathering step, and the second iteration, which is the planning step
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

	ok, sysPrompt, sysPromptTokens := state.getTellSysPrompt(isPlanningStage, isContextStage, autoContextEnabled, smartContextEnabled, modelContextText)
	if !ok {
		return
	}

	// log.Println("**sysPrompt:**\n", sysPrompt)

	state.messages = []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: sysPrompt,
		},
	}

	// Add a separate message for image contexts
	if !state.addImageContext() {
		return
	}

	osDetailsTokens, err := shared.GetNumTokens(req.OsDetails)
	if err != nil {
		log.Printf("Error getting num tokens for os details: %v\n", err)
		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error getting num tokens for os details",
		}
		return
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

	log.Printf("wrapperTokens: %d\n", wrapperTokens)

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

		log.Printf("numPromptTokens: %d\n", numPromptTokens)
		log.Printf("osDetailsTokens: %d\n", osDetailsTokens)
		log.Printf("wrapperTokens: %d\n", wrapperTokens)

		promptTokens = wrapperTokens + numPromptTokens + osDetailsTokens
	}

	if req.ExecEnabled {
		log.Printf("Apply script summary num tokens: %d\n", prompts.ApplyScriptSummaryNumTokens)
		promptTokens += prompts.ApplyScriptSummaryNumTokens
	}

	state.tokensBeforeConvo = sysPromptTokens + modelContextTokens + state.latestSummaryTokens + promptTokens

	// print out breakdown of token usage
	log.Printf("System message tokens: %d\n", sysPromptTokens)
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

	if !state.addConversationMessages() {
		return
	}

	var applyScriptSummary string
	if req.ExecEnabled {
		applyScriptSummary = prompts.ApplyScriptPromptSummary
	}

	state.replyId = uuid.New().String()
	state.replyParser = types.NewReplyParser()

	if missingFileResponse == "" {
		if !state.setPromptMessage(isPlanningStage, isContextStage, applyScriptSummary) {
			return
		}
	} else if !state.handleMissingFileResponse(isPlanningStage, applyScriptSummary) {
		return
	}

	log.Printf("\n\nMessages: %d\n", len(state.messages))
	// for _, message := range state.messages {
	// 	log.Printf("%s: %s\n", message.Role, message.Content)
	// }

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		Auth: auth,
		Plan: plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  state.totalRequestTokens,
			OutputTokens: state.settings.ModelPack.Planner.GetReservedOutputTokens(),
			ModelName:    state.settings.ModelPack.Planner.BaseModelConfig.ModelName,
		},
	})
	if apiErr != nil {
		active.StreamDoneCh <- apiErr
		return
	}

	var stop []string
	if isPlanningStage {
		stop = []string{"<EndPlandexTasks/>"}
	} else if isImplementationStage {
		stop = []string{"<PlandexSubtaskDone/>"}
	}

	// log.Println("Stop:", stop)
	// spew.Dump(state.messages)

	modelReq := openai.ChatCompletionRequest{
		Model:    state.settings.ModelPack.Planner.BaseModelConfig.ModelName,
		Messages: state.messages,
		Stream:   true,
		StreamOptions: &openai.StreamOptions{
			IncludeUsage: true,
		},
		Temperature: state.settings.ModelPack.Planner.Temperature,
		TopP:        state.settings.ModelPack.Planner.TopP,
		Stop:        stop,
	}

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
		go state.queuePendingBuilds()
	}

	UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
		ap.CurrentStreamingReplyId = state.replyId
		ap.CurrentReplyDoneCh = make(chan bool, 1)
	})

	go state.listenStream(stream)
}
