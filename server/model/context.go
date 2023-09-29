package model

import (
	"fmt"
	"strings"

	"github.com/plandex/plandex/shared"
)

func FormatModelContext(context shared.ModelContext) string {
	var contextMessages []string
	for _, part := range context {
		var message string
		if part.FilePath != "" {
			message = fmt.Sprintf("\n\n-file: %s\n\n```%s```", part.FilePath, part.Body)
		} else if part.Url != "" {
			message = fmt.Sprintf("\n\n-url: %s\n\n```%s```", part.Url, part.Body)
		} else {
			message = fmt.Sprintf("\n\n-content%s:\n\n```%s```", part.Name, part.Body)
		}

		contextMessages = append(contextMessages, message)
	}
	return strings.Join(contextMessages, "\n")
}

func FormatCurrentPlan(plan shared.CurrentPlanFiles) string {
	var planMessages []string
	for path, content := range plan.Files {
		planMessages = append(planMessages, fmt.Sprintf("\n\n-file: %s\n\n```%s```", path, content))
	}

	if plan.Exec != "" {
		planMessages = append(planMessages, fmt.Sprintf("\n\n-exec: %s", plan.Exec))
	}

	return strings.Join(planMessages, "\n")
}
