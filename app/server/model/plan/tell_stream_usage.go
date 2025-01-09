package plan

import (
	"log"
	"plandex-server/hooks"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) handleUsageChunk(usage *openai.Usage) {
	planId := state.plan.Id
	branch := state.branch
	auth := state.auth
	plan := state.plan

	log.Println("Tell stream usage:")
	log.Println(spew.Sdump(usage))

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
		},
	})

	if apiErr != nil {
		log.Printf("Tell stream: error executing did send model request hook: %v\n", apiErr)

		// ensure the active plan is still available
		activePlan := GetActivePlan(planId, branch)

		if activePlan == nil {
			log.Printf(" Active plan not found for plan ID %s on branch %s\n", planId, branch)
			return
		}

		activePlan.StreamDoneCh <- apiErr
	}
}

func (state *activeTellStreamState) execHookOnStop(sendStreamErr bool) {
	planId := state.plan.Id
	branch := state.branch
	auth := state.auth
	plan := state.plan
	active := GetActivePlan(planId, branch)

	if active == nil {
		log.Printf(" Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return
	}

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
			Purpose:       "Generated plan reply",
		},
	})

	if apiErr != nil {
		log.Printf("Error executing did send model request hook after cancel or error: %v\n", apiErr)

		if sendStreamErr {
			activePlan := GetActivePlan(planId, branch)

			if activePlan == nil {
				log.Printf(" Active plan not found for plan ID %s on branch %s\n", planId, branch)
				return
			}

			activePlan.StreamDoneCh <- apiErr
		}
	}
}
