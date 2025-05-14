package lib

import (
	"fmt"
	"plandex-cli/term"

	shared "plandex-shared"
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
