package term

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
)

func OutputNoApiKeyMsgAndExit() {
	fmt.Fprintln(os.Stderr, color.New(color.Bold, color.FgHiRed).Sprintln("\n🚨 OPENAI_API_KEY environment variable is not set.")+color.New().Sprintln("\nSet it with:\n\nexport OPENAI_API_KEY=your-api-key\n\nThen try again.\n\n👉 If you don't have an OpenAI account, sign up here → https://platform.openai.com/signup\n\n🔑 Generate an api key here → https://platform.openai.com/api-keys"))
	os.Exit(1)
}

func OutputErrorAndExit(msg string, args ...interface{}) {
	StopSpinner()
	msg = fmt.Sprintf(msg, args...)

	displayMsg := ""
	errorParts := strings.Split(msg, ": ")

	if len(errorParts) > 1 {
		for i, part := range errorParts {
			if i != 0 {
				displayMsg += "\n"
			}
			// indent the error message
			for n := 0; n < i; n++ {
				displayMsg += "  "
			}
			if i > 0 {
				displayMsg += "→ "
			}

			displayMsg += shared.Capitalize(part)
		}
	} else {
		displayMsg = shared.Capitalize(msg)
	}

	fmt.Fprintln(os.Stderr, color.New(color.FgHiRed, color.Bold).Sprint("🚨 "+displayMsg))
	os.Exit(1)
}
