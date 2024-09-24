package term

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
)

var openUnauthenticatedCloudURL func(msg, path string)
var openAuthenticatedURL func(msg, path string)

func SetOpenUnauthenticatedCloudURLFn(fn func(msg, path string)) {
	openUnauthenticatedCloudURL = fn
}

func SetOpenAuthenticatedURLFn(fn func(msg, path string)) {
	openAuthenticatedURL = fn
}

func OutputNoOpenAIApiKeyMsgAndExit() {
	fmt.Fprintln(os.Stderr, color.New(color.Bold, ColorHiRed).Sprintln("\n🚨 OPENAI_API_KEY environment variable is not set.")+color.New().Sprintln("\nSet it with:\n\nexport OPENAI_API_KEY=your-api-key\n\nThen try again.\n\n👉 If you don't have an OpenAI account, sign up here → https://platform.openai.com/signup\n\n🔑 Generate an api key here → https://platform.openai.com/api-keys"))
	os.Exit(1)
}

func OutputSimpleError(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	fmt.Fprintln(os.Stderr, color.New(ColorHiRed, color.Bold).Sprint("🚨 "+shared.Capitalize(msg)))
}

func OutputErrorAndExit(msg string, args ...interface{}) {
	StopSpinner()
	msg = fmt.Sprintf(msg, args...)

	displayMsg := ""
	errorParts := strings.Split(msg, ": ")

	addedErrors := map[string]bool{}

	if len(errorParts) > 1 {
		var lastPart string
		i := 0
		for _, part := range errorParts {
			// don't repeat the same error message
			if _, ok := addedErrors[strings.ToLower(part)]; ok {
				continue
			}

			if len(lastPart) < 10 && i > 0 {
				lastPart = lastPart + ": " + part
				displayMsg += ": " + part
				addedErrors[strings.ToLower(lastPart)] = true
				addedErrors[strings.ToLower(part)] = true
				continue
			}

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

			s := shared.Capitalize(part)
			if i == 0 {
				s = color.New(ColorHiRed, color.Bold).Sprint("🚨 " + s)
			}

			displayMsg += s

			addedErrors[strings.ToLower(part)] = true
			lastPart = part
			i++
		}
	} else {
		displayMsg = color.New(ColorHiRed, color.Bold).Sprint("🚨 " + msg)
	}

	fmt.Fprintln(os.Stderr, color.New(ColorHiRed, color.Bold).Sprint(displayMsg))
	os.Exit(1)
}

func OutputUnformattedErrorAndExit(msg string) {
	StopSpinner()
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func OutputNoCurrentPlanErrorAndExit() {
	fmt.Println("🤷‍♂️ No current plan")
	fmt.Println()
	PrintCmds("", "new", "cd")
	os.Exit(1)
}

func HandleApiError(apiError *shared.ApiError) {
	if apiError.Type == shared.ApiErrorTypeCloudSubscriptionPaused {
		if apiError.BillingError.HasBillingPermission {
			StopSpinner()
			fmt.Println("Your org's Plandex Cloud subscription is paused.")
			res, err := ConfirmYesNo("Go to billing settings?")
			if err != nil {
				OutputErrorAndExit("error getting confirmation")
			}
			if res {
				openAuthenticatedURL("Opening billing settings in your browser.", "/settings/billing")
				os.Exit(0)
			} else {
				os.Exit(0)
			}
		} else {
			OutputErrorAndExit("Your org's subscription is paused. Please contact an org owner to continue.")
		}
	}

	if apiError.Type == shared.ApiErrorTypeCloudSubscriptionOverdue {
		if apiError.BillingError.HasBillingPermission {
			StopSpinner()
			OutputSimpleError("Your org's Plandex Cloud subscription is overdue.")
			res, err := ConfirmYesNo("Go to billing settings?")
			if err != nil {
				OutputErrorAndExit("error getting confirmation")
			}
			if res {
				openAuthenticatedURL("Opening billing settings in your browser.", "/settings/billing")
				os.Exit(0)
			} else {
				os.Exit(0)
			}
		} else {
			OutputErrorAndExit("Your org's subscription is overdue. Please contact an org owner to continue.")
		}
	}

	if apiError.Type == shared.ApiErrorTypeCloudMonthlyMaxReached {
		if apiError.BillingError.HasBillingPermission {
			StopSpinner()
			OutputSimpleError("Your org has reached its monthly limit for Plandex Cloud.")
			res, err := ConfirmYesNo("Go to billing settings?")
			if err != nil {
				OutputErrorAndExit("error getting confirmation")
			}
			if res {
				openAuthenticatedURL("Opening billing settings in your browser.", "/settings/billing")
				os.Exit(0)
			} else {
				os.Exit(0)
			}
		} else {
			OutputErrorAndExit("Your org has reached its monthly limit for Plandex Cloud.")
		}
	}

	if apiError.Type == shared.ApiErrorTypeCloudInsufficientCredits {
		if apiError.BillingError.HasBillingPermission {
			StopSpinner()
			OutputSimpleError("Insufficient credits.")
			res, err := ConfirmYesNo("Go to billing settings?")
			if err != nil {
				OutputErrorAndExit("error getting confirmation")
			}
			if res {
				openAuthenticatedURL("Opening billing settings in your browser.", "/settings/billing")
				os.Exit(0)
			} else {
				os.Exit(0)
			}
		} else {
			OutputErrorAndExit("Insufficient credits.")
		}
	}
}
