package lib

import (
	"fmt"
	"strings"

	"github.com/plandex/plandex/shared"
)

func FormatModelContext(context shared.ModelContext) (string, int) {
	var contextMessages []string
	var numTokens int
	for _, part := range context {
		var message string
		var fmtStr string
		var args []any

		if part.Type == shared.ContextDirectoryTreeType {
			fmtStr = "\n\n- %s | directory tree:\n\n```\n%s\n```"
			args = append(args, part.FilePath, part.Body)
		} else if part.Type == shared.ContextFileType {
			fmtStr = "\n\n- %s:\n\n```\n%s\n```"
			args = append(args, part.FilePath, part.Body)
		} else if part.Url != "" {
			fmtStr = "\n\n- %s:\n\n```\n%s\n```"
			args = append(args, part.Url, part.Body)
		} else {
			fmtStr = "\n\n- content%s:\n\n```\n%s\n```"
			args = append(args, part.Name, part.Body)
		}

		numTokens += part.NumTokens + shared.GetNumTokens(fmt.Sprintf(fmtStr, ""))

		message = fmt.Sprintf(fmtStr, args...)

		contextMessages = append(contextMessages, message)
	}
	return strings.Join(contextMessages, "\n"), numTokens
}

func FormatCurrentPlan(plan shared.CurrentPlanFiles) string {
	var planMessages []string
	for path, content := range plan.Files {
		planMessages = append(planMessages, fmt.Sprintf("\n\n- %s:\n\n```%s```", path, content))
	}

	return strings.Join(planMessages, "\n")
}
