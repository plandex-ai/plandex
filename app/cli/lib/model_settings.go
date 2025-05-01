package lib

import (
	"fmt"
	"os"
	"plandex-cli/api"
	"plandex-cli/term"
	"strings"

	shared "plandex-shared"

	"github.com/fatih/color"
)

const GoBack = "← Go back"

func SelectModelForRole(customModels []*shared.AvailableModel, role shared.ModelRole, includeProviderGoBack bool) *shared.AvailableModel {
	var providers []string
	addedProviders := map[string]bool{}

	builtInModels := shared.FilterCompatibleModels(shared.AvailableModels, role)

	for _, m := range builtInModels {
		var p string
		if m.Provider == shared.ModelProviderCustom {
			p = *m.CustomProvider
		} else {
			p = string(m.Provider)
		}

		if !addedProviders[p] {
			providers = append(providers, p)
			addedProviders[p] = true
		}
	}

	customModels = shared.FilterCompatibleModels(customModels, role)

	for _, m := range customModels {
		var p string
		if m.Provider == shared.ModelProviderCustom {
			p = *m.CustomProvider
		} else {
			p = string(m.Provider)
		}

		if !addedProviders[p] {
			providers = append(providers, p)
			addedProviders[p] = true
		}
	}

	for {
		var opts []string
		opts = append(opts, providers...)
		if includeProviderGoBack {
			opts = append(opts, GoBack)
		}
		provider, err := term.SelectFromList("Select a provider:", opts)
		if err != nil {
			if err.Error() == "interrupt" {
				return nil
			}

			term.OutputErrorAndExit("Error selecting provider: %v", err)
			return nil
		}

		if provider == GoBack {
			break
		}

		var selectableModels []*shared.AvailableModel
		opts = []string{}

		for _, m := range builtInModels {
			var p string
			if m.Provider == shared.ModelProviderCustom {
				p = *m.CustomProvider
			} else {
				p = string(m.Provider)
			}

			if p == provider {
				label := fmt.Sprintf("%s | max %d 🪙", m.ModelString(), m.MaxTokens)
				opts = append(opts, label)
				selectableModels = append(selectableModels, m)
			}
		}

		for _, m := range customModels {
			var p string
			if m.Provider == shared.ModelProviderCustom {
				p = *m.CustomProvider
			} else {
				p = string(m.Provider)
			}

			if p == provider {
				label := fmt.Sprintf("%s → %s | max %d 🪙", p, m.ModelName, m.MaxTokens)
				opts = append(opts, label)
				selectableModels = append(selectableModels, m)
			}
		}

		opts = append(opts, GoBack)

		selection, err := term.SelectFromList("Select a model:", opts)

		if err != nil {
			if err.Error() == "interrupt" {
				return nil
			}

			term.OutputErrorAndExit("Error selecting model: %v", err)
			return nil
		}

		if selection == GoBack {
			continue
		}

		var idx int
		for i := range opts {
			if opts[i] == selection {
				idx = i
				break
			}
		}

		return selectableModels[idx]

	}

	return nil

}

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

	missingAny := false
	if len(requiredEnvVars.RequiresAll) > 0 {
		for envVar := range requiredEnvVars.RequiresAll {
			if os.Getenv(envVar) == "" {
				fmt.Fprintln(os.Stderr, color.New(color.Bold, term.ColorHiRed).Sprintf("🚨 %s environment variable is not set.\n", envVar))
				delete(requiredEnvVars.RequiresEither, envVar)
				missingAny = true
			} else {
				apiKeys[envVar] = os.Getenv(envVar)
			}
		}
	}

	if len(requiredEnvVars.RequiresEither) > 0 {
		vars := []string{}
		for envVar := range requiredEnvVars.RequiresEither {
			if os.Getenv(envVar) == "" {
				vars = append(vars, envVar)
			} else {
				apiKeys[envVar] = os.Getenv(envVar)
			}
		}
		if len(vars) > 1 {
			s := "🚨 Either "
			if len(vars) == 2 {
				s += strings.Join(vars, " or ") + " must be set as an environment variable."
			} else {
				withoutLast := vars[:len(vars)-1]
				s += strings.Join(withoutLast, ", ") + ", or " + vars[len(vars)-1] + " must be set as an environment variable."
			}
			fmt.Fprintln(os.Stderr, color.New(color.Bold, term.ColorHiRed).Sprint(s))
			missingAny = true
		}
	}

	if missingAny {
		os.Exit(1)
	}

	return apiKeys
}
