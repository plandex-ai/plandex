package model

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
		var labelArg string

		if part.FilePath != "" {
			fmtStr = "\n\n- %s:\n\n```%s```"
			labelArg = part.FilePath
		} else if part.Url != "" {
			fmtStr = "\n\n- %s:\n\n```%s```"
			labelArg = part.Url
		} else {
			fmtStr = "\n\n- content%s:\n\n```%s```"
			labelArg = part.Name
		}

		numTokens += part.NumTokens + shared.GetNumTokens(fmt.Sprintf(fmtStr, labelArg, ""))

		message = fmt.Sprintf(fmtStr, labelArg, part.Body)

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
