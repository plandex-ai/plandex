package plan

import (
	"fmt"
	"strings"

	"github.com/plandex/plandex/shared"
)

func (state *activeTellStreamState) formatModelContext(includeMaps, includeTrees, isImplementationStage, execEnabled bool) (string, int, error) {
	var contextMessages []string = []string{
		"### LATEST PLAN CONTEXT ###",
	}
	var numTokens int
	addedFilesSet := map[string]bool{}

	uses := map[string]bool{}
	if isImplementationStage && state.currentSubtask != nil {
		for _, path := range state.currentSubtask.UsesFiles {
			uses[path] = true
		}

	}

	for _, part := range state.modelContext {
		if isImplementationStage && state.currentSubtask != nil && part.ContextType == shared.ContextFileType && !uses[part.FilePath] {
			continue
		}

		var message string
		var fmtStr string
		var args []any

		if part.ContextType == shared.ContextDirectoryTreeType {
			if !includeTrees {
				continue
			}
			fmtStr = "\n\n- %s | directory tree:\n\n```\n%s\n```"
			args = append(args, part.FilePath, part.Body)
		} else if part.ContextType == shared.ContextFileType {
			fmtStr = "\n\n- %s:\n\n```\n%s\n```"

			var body string
			var found bool
			if state.currentPlanState.CurrentPlanFiles != nil {
				res, ok := state.currentPlanState.CurrentPlanFiles.Files[part.FilePath]
				if ok {
					body = res
					found = true
				}
			}
			if !found {
				body = part.Body
			}
			addedFilesSet[part.FilePath] = true

			args = append(args, part.FilePath, body)
		} else if part.ContextType == shared.ContextMapType {
			if !includeMaps {
				continue
			}
			fmtStr = "\n\n- %s | map:\n\n```\n%s\n```"
			args = append(args, part.FilePath, part.Body)
		} else if part.Url != "" {
			fmtStr = "\n\n- %s:\n\n```\n%s\n```"
			args = append(args, part.Url, part.Body)
		} else if part.ContextType != shared.ContextImageType {
			fmtStr = "\n\n- content%s:\n\n```\n%s\n```"
			args = append(args, part.Name, part.Body)
		}

		if part.ContextType == shared.ContextImageType {
			numTokens += part.NumTokens
		} else {
			numContextTokens, err := shared.GetNumTokens(fmt.Sprintf(fmtStr, ""))
			if err != nil {
				err = fmt.Errorf("failed to get the number of tokens in the context: %v", err)
				return "", 0, err
			}

			numTokens += part.NumTokens + numContextTokens
			message = fmt.Sprintf(fmtStr, args...)
			contextMessages = append(contextMessages, message)
		}

	}

	// Add any current files in plan that weren't added to the context
	for filePath, body := range state.currentPlanState.CurrentPlanFiles.Files {
		if !addedFilesSet[filePath] {

			if isImplementationStage && !uses[filePath] {
				continue
			}

			if filePath == "_apply.sh" {
				continue
			}

			contextMessages = append(contextMessages, fmt.Sprintf("\n\n- %s:\n\n```\n%s\n```", filePath, body))
		}
	}

	if len(state.currentPlanState.CurrentPlanFiles.Removed) > 0 {
		contextMessages = append(contextMessages, "*Removed files:*\n")
		for path := range state.currentPlanState.CurrentPlanFiles.Removed {
			contextMessages = append(contextMessages, fmt.Sprintf("- %s", path))
		}
		contextMessages = append(contextMessages, "These files have been *removed* and are no longer in the plan. If you want to re-add them to the plan, you must explicitly create them again.")
	}

	if execEnabled {
		contextMessages = append(contextMessages, state.currentPlanState.ExecHistory())

		scriptContent, ok := state.currentPlanState.CurrentPlanFiles.Files["_apply.sh"]
		if !ok || scriptContent == "" {
			scriptContent = "[empty]"
		}

		contextMessages = append(contextMessages, "*Current* state of _apply.sh script:")
		contextMessages = append(contextMessages, fmt.Sprintf("\n\n- _apply.sh:\n\n```\n%s\n```", scriptContent))
	}

	return strings.Join(contextMessages, "\n### END OF CONTEXT ###\n"), numTokens, nil
}

func (state *activeTellStreamState) checkAutoLoadContext() []string {
	activePlan := GetActivePlan(state.plan.Id, state.branch)

	if activePlan == nil {
		return nil
	}

	if !activePlan.AutoContext {
		return nil
	}

	if state.req.IsUserContinue {
		return nil
	}

	// only load context on the first iteration of a non-continue prompt
	if state.iteration > 0 {
		return nil
	}

	split := strings.Split(activePlan.CurrentReplyContent, "### Load Context")

	if len(split) < 2 {
		return nil
	}

	req := state.req

	list := strings.Split(split[1], "\n")
	files := []string{}

	for _, line := range list {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "-") {
			trimmed = strings.TrimPrefix(trimmed, "-")
			trimmed = strings.ReplaceAll(trimmed, "`", "")
			trimmed = strings.TrimSpace(trimmed)

			if req.ProjectPaths[trimmed] {
				files = append(files, trimmed)
			}
		}
	}

	return files
}
