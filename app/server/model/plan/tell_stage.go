package plan

import (
	"log"
	"plandex-server/db"
	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) resolveCurrentStage() (activatedPaths map[string]bool) {
	req := state.req
	iteration := state.iteration
	hasContextMap := state.hasContextMap
	convo := state.convo
	contextMapEmpty := state.contextMapEmpty

	log.Printf("[resolveCurrentStage] Initial state: hasContextMap: %v, convo len: %d", hasContextMap, len(convo))

	var lastConvoMsg *db.ConvoMessage
	if len(convo) > 0 {
		lastConvoMsg = convo[len(convo)-1]
		log.Printf("[resolveCurrentStage] Last convo message - Role: %s, Flags: %+v", lastConvoMsg.Role, lastConvoMsg.Flags)
	} else {
		log.Println("[resolveCurrentStage] No previous conversation messages")
	}

	activatedPaths = map[string]bool{}

	isContinueFromAssistantMsg := false

	if lastConvoMsg != nil {
		isContinueFromAssistantMsg = iteration == 0 && req.IsUserContinue && lastConvoMsg.Role == openai.ChatMessageRoleAssistant
		log.Printf("[resolveCurrentStage] isContinueFromAssistantMsg: %v (IsUserContinue: %v, LastMsgRole: %s)",
			isContinueFromAssistantMsg, req.IsUserContinue, lastConvoMsg.Role)
	}

	isUserPrompt := false

	if !isContinueFromAssistantMsg {
		isUserPrompt = lastConvoMsg == nil || lastConvoMsg.Role == openai.ChatMessageRoleUser
		log.Printf("[resolveCurrentStage] isUserPrompt: %v", isUserPrompt)
	}

	var tellStage shared.TellStage
	var planningPhase shared.PlanningPhase

	if isUserPrompt {
		tellStage = shared.TellStagePlanning
		log.Println("[resolveCurrentStage] Set tellStage to Planning due to user prompt")
	} else {
		if lastConvoMsg != nil && lastConvoMsg.Flags.DidMakePlan {
			tellStage = shared.TellStageImplementation
			log.Println("[resolveCurrentStage] Set tellStage to Implementation - DidMakePlan: true, IsChatOnly: false")
		} else if lastConvoMsg != nil && lastConvoMsg.Flags.CurrentStage.TellStage == shared.TellStageImplementation {
			tellStage = shared.TellStageImplementation
			log.Println("[resolveCurrentStage] Set tellStage to Implementation - CurrentStage: implementation")
		} else {
			tellStage = shared.TellStagePlanning
			log.Printf("[resolveCurrentStage] Set tellStage to Planning - DidMakePlan: %v, IsChatOnly: %v",
				lastConvoMsg != nil && lastConvoMsg.Flags.DidMakePlan, req.IsChatOnly)
		}
	}

	wasContextStage := false
	if lastConvoMsg != nil {
		flags := lastConvoMsg.Flags
		log.Printf("[resolveCurrentStage] Last convo message flags: %+v", flags)
		if flags.CurrentStage.TellStage == shared.TellStagePlanning && flags.CurrentStage.PlanningPhase == shared.PlanningPhaseContext {
			wasContextStage = true
			activatedPaths = lastConvoMsg.ActivatedPaths
			log.Printf("[resolveCurrentStage] Was context stage, copied activatedPaths: %v", activatedPaths)
		}
	}

	if tellStage == shared.TellStagePlanning {
		if req.AutoContext && hasContextMap && !contextMapEmpty && !wasContextStage && !req.IsApplyDebug {
			planningPhase = shared.PlanningPhaseContext
			log.Printf("[resolveCurrentStage] Set planningPhase to Context - AutoContext: %v, hasContextMap: %v, contextMapEmpty: %v, wasContextStage: %v, IsApplyDebug: %v",
				req.AutoContext, hasContextMap, contextMapEmpty, wasContextStage, req.IsApplyDebug)
		} else {
			planningPhase = shared.PlanningPhasePlanning
			log.Printf("[resolveCurrentStage] Set planningPhase to Planning - AutoContext: %v, hasContextMap: %v, contextMapEmpty: %v, wasContextStage: %v",
				req.AutoContext, hasContextMap, contextMapEmpty, wasContextStage)
		}
	}

	state.currentStage = shared.CurrentStage{
		TellStage:     tellStage,
		PlanningPhase: planningPhase,
	}
	log.Printf("[resolveCurrentStage] Final state - TellStage: %s, PlanningPhase: %s", tellStage, planningPhase)

	return activatedPaths
}
