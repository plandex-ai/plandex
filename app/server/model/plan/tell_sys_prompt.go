package plan

import (
	"fmt"
	"log"
	"net/http"
	"plandex-server/model/prompts"

	"github.com/plandex/plandex/shared"
)

func (state *activeTellStreamState) getTellSysPrompt(isPlanningStage, isContextStage, autoContextEnabled, smartContextEnabled bool, modelContextText string) (bool, string, int) {
	req := state.req

	active := GetActivePlan(state.plan.Id, state.branch)

	if active == nil {
		log.Printf("execTellPlan: Active plan not found for plan ID %s on branch %s\n", state.plan.Id, state.branch)
		return false, "", 0
	}

	var sysCreate string
	var sysCreateTokens int

	if isPlanningStage {
		log.Println("isPlanningStage")
		if autoContextEnabled && isContextStage {
			sysCreate = prompts.AutoContextPreamble
			sysCreateTokens = prompts.AutoContextPreambleNumTokens
		} else if autoContextEnabled || smartContextEnabled {
			sysCreate = prompts.SysPlanningAutoContext
			sysCreateTokens = prompts.SysPlanningAutoContextTokens
		} else {
			sysCreate = prompts.SysPlanningBasic
			sysCreateTokens = prompts.SysPlanningBasicTokens
		}
	} else {
		log.Println("isImplementationStage")
		if state.currentSubtask == nil {
			err := fmt.Errorf("no current subtask")
			log.Println(err)
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "No current subtask",
			}
			return false, "", 0
		}
		sysCreate = prompts.GetImplementationPrompt(state.currentSubtask.Title)
	}

	if !isContextStage {
		if req.ExecEnabled {
			sysCreate += prompts.ApplyScriptPrompt
			sysCreateTokens += prompts.ApplyScriptPromptNumTokens
		} else {
			sysCreate += prompts.NoApplyScriptPrompt
			sysCreateTokens += prompts.NoApplyScriptPromptNumTokens
		}
	}

	// log.Println("sysCreate before context:\n", sysCreate)

	sysCreate += modelContextText

	if len(active.SkippedPaths) > 0 {
		skippedPrompt := prompts.SkippedPathsPrompt
		for skippedPath := range active.SkippedPaths {
			skippedPrompt += fmt.Sprintf("- %s\n", skippedPath)
		}

		numTokens, err := shared.GetNumTokens(skippedPrompt)
		if err != nil {
			log.Printf("Error getting num tokens for skipped paths: %v\n", err)
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Error getting num tokens for skipped paths",
			}
			return false, "", 0
		}

		sysCreateTokens += numTokens
	}

	if len(state.subtasks) > 0 {
		subtasksPrompt, subtaskTokens, err := state.formatSubtasks()

		if err != nil {
			err = fmt.Errorf("error formatting subtasks: %v", err)
			log.Println(err)
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Error formatting subtasks",
			}
			return false, "", 0
		}

		// log.Println("subtasksPrompt:\n", subtasksPrompt)

		sysCreate += subtasksPrompt
		sysCreateTokens += subtaskTokens
	}

	return true, sysCreate, sysCreateTokens
}
