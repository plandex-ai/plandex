package plan

import (
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/model/parse"
	shared "plandex-shared"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

func (state *activeTellStreamState) formatSubtasks() string {
	subtasksText := "### LATEST PLAN TASKS ###\n\n"

	var current *db.Subtask

	for idx, subtask := range state.subtasks {
		subtasksText += fmt.Sprintf("%d. %s\n", idx+1, subtask.Title)
		if subtask.Description != "" {
			subtasksText += "\n" + subtask.Description + "\n"
		}
		if len(subtask.UsesFiles) > 0 {
			subtasksText += "Uses: "
			usesFiles := []string{}
			for _, file := range subtask.UsesFiles {
				usesFiles = append(usesFiles, fmt.Sprintf("`%s`", file))
			}
			subtasksText += strings.Join(usesFiles, ", ") + "\n"
		}
		subtasksText += "Done: "
		if subtask.IsFinished {
			subtasksText += "yes"
		} else {
			subtasksText += "no"
		}
		subtasksText += "\n"

		if state.currentSubtask != nil && subtask.Title == state.currentSubtask.Title && state.currentStage.TellStage == shared.TellStageImplementation {
			current = subtask
			subtasksText += "Current subtask: yes"
		}

		subtasksText += "\n"
	}

	if current != nil && state.currentStage.TellStage == shared.TellStageImplementation {
		subtasksText += fmt.Sprintf("\n### Current subtask\n%s\n", current.Title)
		if current.Description != "" {
			subtasksText += "\n" + current.Description + "\n"
		}
		if len(current.UsesFiles) > 0 {
			subtasksText += "Uses: "
			usesFiles := []string{}
			for _, file := range current.UsesFiles {
				usesFiles = append(usesFiles, fmt.Sprintf("`%s`", file))
			}
			subtasksText += strings.Join(usesFiles, ", ") + "\n"
		}
	} else if state.currentStage.TellStage == shared.TellStagePlanning {
		if state.currentStage.PlanningPhase == shared.PlanningPhasePlanning {
			subtasksText += `
			
			Remember, you are in the *PLANNING* phase and ABSOLUTELY MUST NOT implement any of the subtasks. You MUST NOT write any code or create any files. You can ONLY add or remove subtasks with a '### Tasks' section or a '### Remove Tasks' section. You CANNOT implement any of the subtasks in this response. Follow the PLANNING instructions. The existing subtasks are included for your reference so that you can see what has been planned so far, what has been done, and what is left to do, so that you can add or remove subtasks as needed. DO NOT implement any of the subtasks in this response-follow the instructions for the PLANNING phase.

		`
		} else if state.currentStage.PlanningPhase == shared.PlanningPhaseContext {
			subtasksText += `
			
			Remember, you are in the *CONTEXT* phase. You MUST NOT implement any of the subtasks. You MUST NOT write any code or create any files. You MUST NOT make a plan with a '### Tasks' section or a '### Remove Tasks' section. Follow the instructions for the CONTEXT phase-they are summarized for you in the [SUMMARY OF INSTRUCTIONS] section. The existing subtasks are included for your reference so that you can see what has been planned so far, what has been done, and what is left to do. DO NOT implement any of the subtasks in this response Do NOT add or remove subtasks. Follow the instructions for the CONTEXT phase.

		`
		}
	}

	return subtasksText
}

func (state *activeTellStreamState) checkNewSubtasks() []*db.Subtask {
	activePlan := GetActivePlan(state.plan.Id, state.branch)

	if activePlan == nil {
		return nil
	}

	content := activePlan.CurrentReplyContent

	subtasks := parse.ParseSubtasks(content)

	if len(subtasks) == 0 {
		log.Println("No new subtasks found")
		return nil
	}

	log.Println("Found new subtasks:", len(subtasks))
	// log.Println(spew.Sdump(subtasks))

	subtasksByName := map[string]*db.Subtask{}

	// Only index unfinished subtasks by name
	for _, subtask := range state.subtasks {
		if !subtask.IsFinished {
			subtasksByName[subtask.Title] = subtask
		}
	}

	var newSubtasks []*db.Subtask
	var updatedSubtasks []*db.Subtask

	// Keep finished subtasks
	for _, subtask := range state.subtasks {
		if subtask.IsFinished {
			updatedSubtasks = append(updatedSubtasks, subtask)
		}
	}

	// Add new subtasks if they don't exist
	for _, subtask := range subtasks {
		if subtasksByName[subtask.Title] == nil {
			newSubtasks = append(newSubtasks, subtask)
			updatedSubtasks = append(updatedSubtasks, subtask)
		}
	}

	state.subtasks = updatedSubtasks

	var currentSubtaskName string
	if state.currentSubtask != nil {
		currentSubtaskName = state.currentSubtask.Title
	}

	found := false
	for _, subtask := range state.subtasks {
		if subtask.Title == currentSubtaskName {
			found = true
			state.currentSubtask = subtask
			break
		}
	}
	if !found {
		state.currentSubtask = nil
	}

	if state.currentSubtask == nil {
		for _, subtask := range state.subtasks {
			if !subtask.IsFinished {
				state.currentSubtask = subtask
				break
			}
		}
	}

	// log.Println("state.subtasks:\n", spew.Sdump(state.subtasks))
	log.Println("state.currentSubtask:\n", spew.Sdump(state.currentSubtask))

	return newSubtasks
}

func (state *activeTellStreamState) checkRemoveSubtasks() []string {
	activePlan := GetActivePlan(state.plan.Id, state.branch)

	if activePlan == nil {
		return nil
	}

	content := activePlan.CurrentReplyContent

	// Parse tasks to remove
	tasksToRemove := parse.ParseRemoveSubtasks(content)

	if len(tasksToRemove) == 0 {
		log.Println("No tasks to remove found")
		return nil
	}

	log.Println("Found tasks to remove:", len(tasksToRemove))
	// log.Println(spew.Sdump(tasksToRemove))

	// Create a map of task titles to remove for efficient lookup
	removeMap := make(map[string]bool)
	for _, task := range tasksToRemove {
		removeMap[task] = true
	}

	var removedSubtasks []*db.Subtask
	var remainingSubtasks []*db.Subtask

	// Keep tasks that aren't in the remove list
	for _, subtask := range state.subtasks {
		if removeMap[subtask.Title] {
			// Only track unfinished tasks that are being removed
			if !subtask.IsFinished {
				removedSubtasks = append(removedSubtasks, subtask)
			}
		} else {
			remainingSubtasks = append(remainingSubtasks, subtask)
		}
	}

	state.subtasks = remainingSubtasks

	// Update current subtask if it was removed
	if state.currentSubtask != nil && removeMap[state.currentSubtask.Title] {
		state.currentSubtask = nil
		// Find the first unfinished subtask to set as current
		for _, subtask := range state.subtasks {
			if !subtask.IsFinished {
				state.currentSubtask = subtask
				break
			}
		}
	}

	removedSubtaskTitles := []string{}
	for _, subtask := range removedSubtasks {
		removedSubtaskTitles = append(removedSubtaskTitles, subtask.Title)
	}
	log.Println("removedSubtaskTitles:\n", spew.Sdump(removedSubtaskTitles))

	return removedSubtaskTitles
}
