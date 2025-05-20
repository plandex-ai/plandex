package lib

import (
	"fmt"
	"plandex-cli/term"

	shared "plandex-shared"
)

const GoBack = "← Go back"

func SelectModelIdForRole(customModels []*shared.CustomModel, role shared.ModelRole) shared.ModelId {
	builtInModels := shared.FilterBuiltInCompatibleModels(shared.BuiltInBaseModels, role)
	customModels = shared.FilterCustomCompatibleModels(customModels, role)

	for {
		var selectableModelIds []shared.ModelId
		opts := []string{}

		for _, m := range builtInModels {
			label := fmt.Sprintf("%s | max %d 🪙", m.ModelId, m.MaxTokens)
			opts = append(opts, label)
			selectableModelIds = append(selectableModelIds, m.ModelId)
		}

		for _, m := range customModels {
			label := fmt.Sprintf("%s | max %d 🪙", m.ModelId, m.MaxTokens)
			opts = append(opts, label)
			selectableModelIds = append(selectableModelIds, m.ModelId)
		}

		opts = append(opts, GoBack)

		selection, err := term.SelectFromList("Select a model:", opts)

		if err != nil {
			if err.Error() == "interrupt" {
				return ""
			}

			term.OutputErrorAndExit("Error selecting model: %v", err)
			return ""
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

		id := selectableModelIds[idx]

		return id
	}
}
