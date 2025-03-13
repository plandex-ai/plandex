package plan

import (
	"log"
	"plandex-server/db"
	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) lastSuccessfulConvoMessage() *db.ConvoMessage {
	for i := len(state.convo) - 1; i >= 0; i-- {
		msg := state.convo[i]
		if msg.Stopped || msg.Flags.HasError {
			continue
		}
		return msg
	}
	return nil
}

func (state *activeTellStreamState) resolveCurrentStage() (activatePaths map[string]bool, activatePathsOrdered []string) {
	req := state.req
	iteration := state.iteration
	hasContextMap := state.hasContextMap
	convo := state.convo
	contextMapEmpty := state.contextMapEmpty

	log.Printf("[resolveCurrentStage] Initial state: hasContextMap: %v, convo len: %d", hasContextMap, len(convo))

	lastConvoMsg := state.lastSuccessfulConvoMessage()

	activatePaths = map[string]bool{}
	activatePathsOrdered = []string{}

	isContinueFromAssistantMsg := false

	if lastConvoMsg != nil {
		isContinueFromAssistantMsg = iteration == 0 && req.IsUserContinue && lastConvoMsg.Role == openai.ChatMessageRoleAssistant
		log.Printf("[resolveCurrentStage] isContinueFromAssistantMsg: %v (IsUserContinue: %v, LastMsgRole: %s)",
			isContinueFromAssistantMsg, req.IsUserContinue, lastConvoMsg.Role)
	} else {
		log.Println("[resolveCurrentStage] No previous successful conversation message found")
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
			activatePaths = lastConvoMsg.ActivatedPaths
			activatePathsOrdered = lastConvoMsg.ActivatedPathsOrdered
			log.Printf("[resolveCurrentStage] Was context stage, copied activatePaths: %v", activatePaths)
		}
	}

	if tellStage == shared.TellStagePlanning {
		if req.AutoContext && hasContextMap && !contextMapEmpty && !wasContextStage {
			planningPhase = shared.PlanningPhaseContext
			log.Printf("[resolveCurrentStage] Set planningPhase to Context - AutoContext: %v, hasContextMap: %v, contextMapEmpty: %v, wasContextStage: %v",
				req.AutoContext, hasContextMap, contextMapEmpty, wasContextStage)
		} else {
			planningPhase = shared.PlanningPhaseTasks
			log.Printf("[resolveCurrentStage] Set planningPhase to Tasks - AutoContext: %v, hasContextMap: %v, contextMapEmpty: %v, wasContextStage: %v",
				req.AutoContext, hasContextMap, contextMapEmpty, wasContextStage)
		}
	}

	state.currentStage = shared.CurrentStage{
		TellStage:     tellStage,
		PlanningPhase: planningPhase,
	}
	log.Printf("[resolveCurrentStage] Final state - TellStage: %s, PlanningPhase: %s", tellStage, planningPhase)

	return activatePaths, activatePathsOrdered
}
