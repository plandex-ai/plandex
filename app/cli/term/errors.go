package term

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	shared "plandex-shared"

	"github.com/fatih/color"
)

var openUnauthenticatedCloudURL func(msg, path string)
var openAuthenticatedURL func(msg, path string)
var convertTrial func()

func SetOpenUnauthenticatedCloudURLFn(fn func(msg, path string)) {
	openUnauthenticatedCloudURL = fn
}
func SetOpenAuthenticatedURLFn(fn func(msg, path string)) {
	openAuthenticatedURL = fn
}
func SetConvertTrialFn(fn func()) {
	convertTrial = fn
}

func OutputSimpleError(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	fmt.Fprintln(os.Stderr, color.New(ColorHiRed, color.Bold).Sprint("🚨 "+shared.Capitalize(msg)))
}

func OutputErrorAndExit(msg string, args ...interface{}) {
	StopSpinner()

	msg = fmt.Sprintf(msg, args...)

	msg = strings.ReplaceAll(msg, "status code:", "status code")
	msg = strings.ReplaceAll(msg, ", body:", ":")

	displayMsg := ""
	errorParts := strings.Split(msg, ": ")

	addedErrors := map[string]bool{}

	if len(errorParts) > 1 {
		var lastPart string
		i := 0
		for idx, part := range errorParts {
			// don't repeat the same error message
			if _, ok := addedErrors[strings.ToLower(part)]; ok {
				continue
			}

			tail := strings.Join(errorParts[idx:], ": ")
			if maybeJSON(tail) {
				prettyJSON := prettyJSON(tail)
				indent := strings.Repeat("  ", i)

				// prepend indent to **each** line in the pretty JSON
				indentedJSON := strings.ReplaceAll(prettyJSON, "\n", "\n"+indent+"  ")

				// now write the block
				displayMsg += "\n" + indent + "→ " + indentedJSON
				break
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
			OutputSimpleError("Insufficient credits")
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
			OutputErrorAndExit("Insufficient credits")
		}
	}

	if apiError.Type == shared.ApiErrorTypeTrialMessagesExceeded {
		StopSpinner()
		fmt.Fprintf(os.Stderr, "\n🚨 You've reached the Plandex Cloud trial limit of %d messages per plan\n", apiError.TrialMessagesExceededError.MaxReplies)

		res, err := ConfirmYesNo("Upgrade now?")

		if err != nil {
			OutputErrorAndExit("Error prompting upgrade trial: %v", err)
		}

		if res {
			convertTrial()
			PrintCmds("", "continue")
			os.Exit(0)
		}
	}

	StopSpinner()
	OutputErrorAndExit(apiError.Msg)
}

func maybeJSON(s string) bool {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		return true
	}
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return true
	}
	return false
}

func prettyJSON(s string) string {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s // not JSON
	}
	out, _ := json.MarshalIndent(v, "", "  ")
	return string(out)
}
