package plan

import (
	"fmt"
	"log"
	"plandex-server/hooks"
	"plandex-server/notify"
	"runtime/debug"

	"github.com/davecgh/go-spew/spew"
	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) handleUsageChunk(usage *openai.Usage) {
	auth := state.auth
	plan := state.plan
	generationId := state.generationId

	log.Println("Tell stream usage:")
	log.Println(spew.Sdump(usage))

	var cachedTokens int
	if usage.PromptTokensDetails != nil {
		cachedTokens = usage.PromptTokensDetails.CachedTokens
	}

	sessionId := state.activePlan.SessionId

	modelConfig := state.modelConfig

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in handleUsageChunk: %v\n%s", r, debug.Stack())
				go notify.NotifyErr(notify.SeverityError, fmt.Errorf("panic in handleUsageChunk: %v\n%s", r, debug.Stack()))
			}
		}()

		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:    usage.PromptTokens,
				OutputTokens:   usage.CompletionTokens,
				CachedTokens:   cachedTokens,
				ModelId:        modelConfig.BaseModelConfig.ModelId,
				ModelName:      modelConfig.BaseModelConfig.ModelName,
				ModelProvider:  modelConfig.BaseModelConfig.Provider,
				ModelPackName:  state.settings.ModelPack.Name,
				ModelRole:      modelConfig.Role,
				Purpose:        "Response",
				GenerationId:   generationId,
				PlanId:         plan.Id,
				ModelStreamId:  state.modelStreamId,
				ConvoMessageId: state.replyId,

				RequestStartedAt: state.requestStartedAt,
				Streaming:        true,
				FirstTokenAt:     state.firstTokenAt,
				Req:              state.originalReq,
				StreamResult:     state.activePlan.CurrentReplyContent,
				ModelConfig:      state.modelConfig,

				SessionId: sessionId,
			},
		})

		if apiErr != nil {
			log.Printf("handleUsageChunk - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()
}

func (state *activeTellStreamState) execHookOnStop(sendStreamErr bool) {
	generationId := state.generationId

	log.Printf("execHookOnStop - sendStreamErr: %t\n", sendStreamErr)

	planId := state.plan.Id
	branch := state.branch
	auth := state.auth
	plan := state.plan
	active := GetActivePlan(planId, branch)

	if active == nil {
		log.Printf(" Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return
	}

	modelConfig := state.modelConfig

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in execHookOnStop: %v\n%s", r, debug.Stack())
				go notify.NotifyErr(notify.SeverityError, fmt.Errorf("panic in execHookOnStop: %v\n%s", r, debug.Stack()))
			}
		}()

		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:     state.totalRequestTokens,
				OutputTokens:    active.NumTokens,
				ModelId:         modelConfig.BaseModelConfig.ModelId,
				ModelName:       modelConfig.BaseModelConfig.ModelName,
				ModelProvider:   modelConfig.BaseModelConfig.Provider,
				ModelPackName:   state.settings.ModelPack.Name,
				ModelRole:       modelConfig.Role,
				Purpose:         "Response",
				GenerationId:    generationId,
				PlanId:          plan.Id,
				ModelStreamId:   state.modelStreamId,
				ConvoMessageId:  state.replyId,
				StoppedEarly:    true,
				UserCancelled:   !sendStreamErr,
				HadError:        sendStreamErr,
				NoReportedUsage: true,

				RequestStartedAt: state.requestStartedAt,
				Streaming:        true,
				FirstTokenAt:     state.firstTokenAt,
				Req:              state.originalReq,
				StreamResult:     state.activePlan.CurrentReplyContent,
				ModelConfig:      state.modelConfig,

				SessionId: active.SessionId,
			},
		})

		if apiErr != nil {
			log.Printf("execHookOnStop - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

}
