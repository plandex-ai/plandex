package lib

import (
	"fmt"
	"plandex-cli/term"

	shared "plandex-shared"
)

const GoBack = "← Go back"

func SelectModelForRole(customModels []*shared.CustomModel, role shared.ModelRole) *shared.CustomModel {
	builtInModels := shared.FilterAvailableCompatibleModels(shared.AvailableModels, role)
	customModels = shared.FilterCustomCompatibleModels(customModels, role)

	for {
		var selectableModels []*shared.AvailableModel
		opts := []string{}

		for _, m := range builtInModels {
			label := fmt.Sprintf("%s | max %d 🪙", m.ModelString(), m.MaxTokens)
			opts = append(opts, label)
			selectableModels = append(selectableModels, m)
		}

		for _, m := range customModels {
			label := fmt.Sprintf("%s → %s | max %d 🪙", p, m.ModelName, m.MaxTokens)
			opts = append(opts, label)
			selectableModels = append(selectableModels, m)
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
