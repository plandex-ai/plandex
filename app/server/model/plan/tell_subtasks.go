package plan

import (
	"fmt"
	"strings"

	"github.com/plandex/plandex/shared"
)

func (state *activeTellStreamState) formatSubtasks() (string, int, error) {
	subtasksText := "### LATEST PLAN TASKS ###\n\n"

	var numTokens int

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
			subtasksText += "Current subtask: yes"
		}

		subtasksText += "\n"
	}

	numTokens, err := shared.GetNumTokens(subtasksText)
	if err != nil {
		return "", 0, fmt.Errorf("error getting num tokens for subtasks: %v", err)
	}

	return subtasksText, numTokens, nil
}
