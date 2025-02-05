package parse

import (
	"plandex-server/db"
	"regexp"
	"strings"
)

func ParseSubtasks(replyContent string) []*db.Subtask {
	split := strings.Split(replyContent, "### Tasks")
	if len(split) < 2 {
		split = strings.Split(replyContent, "### Task")
		if len(split) < 2 {
			return nil
		}
	}

	lines := strings.Split(split[1], "\n")

	var subtasks []*db.Subtask
	var currentTask *db.Subtask
	var descLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for any number followed by a period and space
		if matched, _ := regexp.MatchString(`^\d+\.\s`, line); matched {
			// Save previous task if exists
			if currentTask != nil {
				currentTask.Description = strings.Join(descLines, "\n")
				subtasks = append(subtasks, currentTask)
			}

			// Start new task
			parts := strings.SplitN(line, ". ", 2)
			if len(parts) == 2 {
				title := parts[1]
				currentTask = &db.Subtask{
					Title: title,
				}
				descLines = nil
			}
			continue
		}

		// Handle Uses: section
		if strings.HasPrefix(line, "Uses:") {
			if currentTask != nil {
				usesStr := strings.TrimPrefix(line, "Uses:")
				for _, use := range strings.Split(usesStr, ",") {
					use = strings.TrimSpace(use)
					use = strings.Trim(use, "`")
					if use != "" {
						currentTask.UsesFiles = append(currentTask.UsesFiles, use)
					}
				}
			}
			continue
		}

		// Add to description if we have a current task
		if currentTask != nil {
			// Remove bullet point if present, but don't require it
			line = strings.TrimPrefix(line, "-")
			line = strings.TrimSpace(line)
			if line != "" {
				descLines = append(descLines, line)
			}
		}
	}

	// Save final task if exists
	if currentTask != nil {
		currentTask.Description = strings.Join(descLines, "\n")
		subtasks = append(subtasks, currentTask)
	}

	return subtasks
}
