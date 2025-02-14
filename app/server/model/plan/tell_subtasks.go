package plan

import (
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/model/parse"
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

		if state.currentSubtask != nil && subtask.Title == state.currentSubtask.Title {
			current = subtask
			subtasksText += "Current subtask: yes"
		}

		subtasksText += "\n"
	}

	if current != nil {
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

	log.Println("Found new subtasks:")
	log.Println(spew.Sdump(subtasks))

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
	// log.Println("state.currentSubtask:\n", spew.Sdump(state.currentSubtask))

	return newSubtasks
}
