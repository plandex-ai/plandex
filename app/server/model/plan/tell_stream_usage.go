package plan

import (
	"log"
	"plandex-server/hooks"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) handleUsageChunk(usage *openai.Usage) {
	auth := state.auth
	plan := state.plan
	generationId := state.generationId

	log.Println("Tell stream usage:")
	log.Println(spew.Sdump(usage))

	go func() {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:   usage.PromptTokens,
				OutputTokens:  usage.CompletionTokens,
				ModelName:     state.settings.ModelPack.Planner.BaseModelConfig.ModelName,
				ModelProvider: state.settings.ModelPack.Planner.BaseModelConfig.Provider,
				ModelPackName: state.settings.ModelPack.Name,
				ModelRole:     shared.ModelRolePlanner,
				Purpose:       "Generated plan reply",
				GenerationId:  generationId,
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

	go func() {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:   state.totalRequestTokens,
				OutputTokens:  active.NumTokens,
				ModelName:     state.settings.ModelPack.Planner.BaseModelConfig.ModelName,
				ModelProvider: state.settings.ModelPack.Planner.BaseModelConfig.Provider,
				ModelPackName: state.settings.ModelPack.Name,
				ModelRole:     shared.ModelRolePlanner,
				Purpose:       "Generated plan reply (stopped)",
				GenerationId:  generationId,
			},
		})

		if apiErr != nil {
			log.Printf("execHookOnStop - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

}
