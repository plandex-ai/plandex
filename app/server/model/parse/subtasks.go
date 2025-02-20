package parse

import (
	"log"
	"plandex-server/db"
	"regexp"
	"strings"
)

func ParseSubtasks(replyContent string) []*db.Subtask {
	split := strings.Split(replyContent, "### Tasks")
	if len(split) < 2 {
		split = strings.Split(replyContent, "### Task")
		if len(split) < 2 {
			log.Println("[Subtasks] No tasks section found in reply")
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
				log.Printf("[Subtasks] Adding subtask: %q with %d uses files", currentTask.Title, len(currentTask.UsesFiles))
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
				log.Printf("[Subtasks] Added uses files for %q: %v", currentTask.Title, currentTask.UsesFiles)
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
		log.Printf("[Subtasks] Adding final subtask: %q with %d uses files", currentTask.Title, len(currentTask.UsesFiles))
		subtasks = append(subtasks, currentTask)
	}

	log.Printf("[Subtasks] Parsed %d total subtasks", len(subtasks))
	return subtasks
}

func ParseRemoveSubtasks(replyContent string) []string {
	split := strings.Split(replyContent, "### Remove Tasks")
	if len(split) < 2 {
		return nil
	}

	section := split[1]
	lines := strings.Split(section, "\n")
	var tasksToRemove []string

	sawEmptyLine := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			sawEmptyLine = true
			continue
		}
		if sawEmptyLine && !strings.HasPrefix(line, "-") {
			break
		}
		if strings.HasPrefix(line, "- ") {
			title := strings.TrimPrefix(line, "- ")
			title = strings.TrimSpace(title)
			if title != "" {
				tasksToRemove = append(tasksToRemove, title)
			}
		}
	}

	return tasksToRemove
}
