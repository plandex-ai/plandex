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

	"github.com/davecgh/go-spew/spew"
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
	clients                   map[string]*openai.Client
	plan                      *db.Plan
	branch                    string
	auth                      *types.ServerAuth
	req                       *shared.TellPlanRequest
	iteration                 int
	missingFileResponse       shared.RespondMissingFileChoice
	shouldBuildPending        bool
	numErrorRetry             int
	shouldLoadFollowUpContext bool
	didLoadFollowUpContext    bool
	didMakeFollowUpPlan       bool
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
	numErrorRetry := params.numErrorRetry
	shouldLoadFollowUpContext := params.shouldLoadFollowUpContext
	didLoadFollowUpContext := params.didLoadFollowUpContext
	didMakeFollowUpPlan := params.didMakeFollowUpPlan

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

	isFollowUp := shouldLoadFollowUpContext || (iteration == 0 && (len(state.subtasks) > 0 || (req.IsChatOnly && state.hasAssistantReply)))

	isPlanningStage := req.IsChatOnly ||
		shouldLoadFollowUpContext ||
		didLoadFollowUpContext ||
		(!req.IsUserContinue &&
			!didMakeFollowUpPlan &&
			(iteration == 0 ||
				(autoContextEnabled && iteration == 1)))

	isImplementationStage := !isPlanningStage

	isContextStage := isPlanningStage &&
		(shouldLoadFollowUpContext ||
			(!isFollowUp && autoContextEnabled && !state.contextMapEmpty && iteration == 0))

	log.Printf("isPlanningStage: %t, isImplementationStage: %t, isContextStage: %t, isFollowUp: %t\n", isPlanningStage, isImplementationStage, isContextStage, isFollowUp)

	state.isFollowUp = isFollowUp
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

	modelContextText, err := state.formatModelContext(includeMaps, true, isImplementationStage, smartContextEnabled, req.ExecEnabled)
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

	sysPrompt, err := state.getTellSysPrompt(autoContextEnabled, smartContextEnabled, modelContextText)
	if err != nil {
		log.Printf("Error getting tell sys prompt: %v\n", err)
		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    err.Error(),
		}
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
	imageContextTokens, ok := state.addImageContext()
	if !ok {
		return
	}

	var applyScriptSummary string
	if req.ExecEnabled {
		applyScriptSummary = prompts.ApplyScriptPromptSummary
	}

	promptMessage, ok := state.resolvePromptMessage(isPlanningStage, isContextStage, applyScriptSummary)
	if !ok {
		return
	}

	state.tokensBeforeConvo =
		shared.GetMessagesTokenEstimate(state.messages...) +
			shared.GetMessagesTokenEstimate(*promptMessage) +
			state.latestSummaryTokens +
			imageContextTokens +
			shared.TokensPerRequest

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
	} else if !state.handleMissingFileResponse(applyScriptSummary) {
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
	var modelConfig shared.ModelRoleConfig
	if isPlanningStage {
		stop = []string{"<EndPlandexTasks/>"}
		modelConfig = state.settings.ModelPack.Planner.ModelRoleConfig
		if isFollowUp {
			log.Println("Tell plan - isFollowUp - setting stop to <PlandexDecideContext/>")
			stop = append(stop, "<PlandexDecideContext/>")
		} else if isContextStage {
			log.Println("Tell plan - isContextStage - setting modelConfig to context loader")
			modelConfig = state.settings.ModelPack.GetContextLoader()
		}
	} else if isImplementationStage {
		stop = []string{"<PlandexSubtaskDone/>"}
		modelConfig = state.settings.ModelPack.GetCoder()
	}

	// log.Println("Stop:", stop)
	// spew.Dump(state.messages)

	log.Println("modelConfig:", spew.Sdump(modelConfig))

	modelReq := openai.ChatCompletionRequest{
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
