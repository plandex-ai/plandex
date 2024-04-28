package lib

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/term"

	"github.com/fatih/color"
)

func MustVerifyApiKeys() map[string]string {
	return mustVerifyApiKeys(false)
}

func MustVerifyApiKeysSilent() map[string]string {
	return mustVerifyApiKeys(true)
}

func mustVerifyApiKeys(silent bool) map[string]string {
	if !silent {
		term.StartSpinner("")
	}
	planSettings, apiErr := api.Client.GetSettings(CurrentPlanId, CurrentBranch)
	if !silent {
		term.StopSpinner()
	}

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting current settings: %v", apiErr)
	}

	requiredEnvVars := planSettings.GetRequiredEnvVars()

	apiKeys := make(map[string]string)

	if len(requiredEnvVars) == 1 && requiredEnvVars["OPENAI_API_KEY"] {
		if os.Getenv("OPENAI_API_KEY") == "" {
			term.OutputNoOpenAIApiKeyMsgAndExit()
		}
		apiKeys["OPENAI_API_KEY"] = os.Getenv("OPENAI_API_KEY")
		return apiKeys
	}

	missingAny := false
	for envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			fmt.Fprintln(os.Stderr, color.New(color.Bold, term.ColorHiRed).Sprintf("🚨 %s environment variable is not set.\n", envVar))
			missingAny = true
		} else {
			apiKeys[envVar] = os.Getenv(envVar)
		}
	}

	if missingAny {
		os.Exit(1)
	}

	return apiKeys
}
