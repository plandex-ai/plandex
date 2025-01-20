package plan

import (
	"fmt"
	"log"
	"plandex-server/model/prompts"
)

func (state *activeTellStreamState) getTellSysPrompt(autoContextEnabled, smartContextEnabled bool, modelContextText string) (string, error) {
	req := state.req
	isPlanningStage := state.isPlanningStage
	isContextStage := state.isContextStage
	isFollowUp := state.isFollowUp
	active := state.activePlan

	var sysCreate string

	if isPlanningStage {
		log.Println("isPlanningStage")
		if autoContextEnabled && isContextStage {
			sysCreate = prompts.AutoContextPreamble
		} else if autoContextEnabled || smartContextEnabled {
			sysCreate = prompts.SysPlanningAutoContext

			if isFollowUp {
				sysCreate = prompts.FollowUpPlanClassifierPrompt + "\n\n" + sysCreate
			}
		} else {
			sysCreate = prompts.SysPlanningBasic
		}
	} else {
		log.Println("isImplementationStage")
		if state.currentSubtask == nil {
			return "", fmt.Errorf("no current subtask during implementation stage")
		}
		sysCreate = prompts.GetImplementationPrompt(state.currentSubtask.Title)
	}

	if !isContextStage {
		if req.ExecEnabled {
			if isPlanningStage {
				sysCreate += prompts.ApplyScriptPlanningPrompt
			} else {
				sysCreate += prompts.ApplyScriptImplementationPrompt
			}
		} else {
			sysCreate += prompts.NoApplyScriptPrompt
		}
	}

	// log.Println("sysCreate before context:\n", sysCreate)

	sysCreate += modelContextText

	if len(active.SkippedPaths) > 0 {
		skippedPrompt := prompts.SkippedPathsPrompt
		for skippedPath := range active.SkippedPaths {
			skippedPrompt += fmt.Sprintf("- %s\n", skippedPath)
		}
	}

	if len(state.subtasks) > 0 {
		subtasksPrompt := state.formatSubtasks()

		// log.Println("subtasksPrompt:\n", subtasksPrompt)

		sysCreate += subtasksPrompt
	}

	return sysCreate, nil
}
