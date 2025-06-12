package lib

import (
	"fmt"
	"os/exec"
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

func MaybePromptAndOpen(path string) bool {
	editors := detectEditors()
	if len(editors) == 0 {
		// just exit if there are no editors available
		return false
	}
	opts := []string{}
	for _, c := range editors {
		opts = append(opts, "Open with "+c.name)
	}

	const openManually = "Open manually"
	opts = append(opts, openManually)

	choice, err := term.SelectFromList("Open the file now?", opts)
	if err != nil {
		term.OutputErrorAndExit("Error selecting editor: %v", err)
	}

	if choice == openManually {
		return false
	}

	var idx int
	for i, c := range opts {
		if c == choice {
			idx = i
			break
		}
	}

	if idx < len(editors) {
		sel := editors[idx]
		err = exec.Command(sel.cmd, append(sel.args, path)...).Start()
		if err != nil {
			term.OutputErrorAndExit("Error opening template: %v", err)
		}
		return true
	}

	return false
}
