package plan

import (
	"fmt"
	"strings"

	"github.com/plandex/plandex/shared"
)

func (state *activeTellStreamState) formatModelContext(includeMaps, includeTrees bool) (string, int, error) {
	var contextMessages []string = []string{
		"### LATEST PLAN CONTEXT ###",
	}
	var numTokens int
	addedFilesSet := map[string]bool{}

	for _, part := range state.modelContext {
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
			contextMessages = append(contextMessages, fmt.Sprintf("\n\n- %s:\n\n```\n%s\n```", filePath, body))
		}
	}

	return strings.Join(contextMessages, "\n### END OF CONTEXT ###\n"), numTokens, nil
}
