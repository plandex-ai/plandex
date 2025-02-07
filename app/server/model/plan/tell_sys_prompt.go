package plan

import (
	"fmt"
	"log"
	"plandex-server/model/prompts"
)

func (state *activeTellStreamState) getTellSysPrompt(autoContextEnabled, smartContextEnabled, didLoadChatOnlyContext bool, modelContextText string) (string, error) {
	req := state.req
	isPlanningStage := state.isPlanningStage
	isContextStage := state.isContextStage
	isFollowUp := state.isFollowUp
	active := state.activePlan

	var sysCreate string

	if req.IsChatOnly {
		sysCreate = prompts.GetChatSysPrompt(prompts.CreatePromptParams{
			ExecMode:                  req.ExecEnabled,
			AutoContext:               autoContextEnabled,
			LastResponseLoadedContext: didLoadChatOnlyContext,
		})
	} else {

		if isPlanningStage {
			log.Println("isPlanningStage")
			log.Println("autoContextEnabled", autoContextEnabled)
			log.Println("isContextStage", isContextStage)
			if autoContextEnabled && isContextStage {
				log.Println("autoContextEnabled && isContextStage -- adding auto context preamble")
				sysCreate = prompts.AutoContextTellPreamble
			} else if autoContextEnabled || smartContextEnabled {
				log.Println("autoContextEnabled || smartContextEnabled -- adding auto context preamble")
				sysCreate = prompts.SysPlanningAutoContext

				if isFollowUp && !req.IsApplyDebug && !req.IsUserDebug {
					log.Println("isFollowUp && !req.IsApplyDebug && !req.IsUserDebug -- adding follow up classifier prompt")
					sysCreate = prompts.FollowUpPlanClassifierPrompt + "\n\n" + sysCreate
				}
			} else {
				log.Println("sysPlanningBasic")
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
	}

	// log.Println("sysCreate before context:\n", sysCreate)

	sysCreate += modelContextText

	if !req.IsChatOnly {
		if len(active.SkippedPaths) > 0 {
			skippedPrompt := prompts.SkippedPathsPrompt
			for skippedPath := range active.SkippedPaths {
				skippedPrompt += fmt.Sprintf("- %s\n", skippedPath)
			}
			sysCreate += skippedPrompt
		}
	}

	if len(state.subtasks) > 0 {
		subtasksPrompt := state.formatSubtasks()
		// log.Println("subtasksPrompt:\n", subtasksPrompt)
		sysCreate += subtasksPrompt
	}

	return sysCreate, nil
}
