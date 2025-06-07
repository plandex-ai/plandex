package cmd

import (
	"fmt"
	"os"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/term"
	"strconv"

	shared "plandex-shared"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var customModelPacksOnly bool

var modelPacksCmd = &cobra.Command{
	Use:   "model-packs",
	Short: "List all model packs",
	Run:   listModelPacks,
}

var createModelPackCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a model pack",
	Run:   customModelsCreateNotImplemented,
}

var deleteModelPackCmd = &cobra.Command{
	Use:     "delete [name-or-index]",
	Aliases: []string{"rm"},
	Short:   "Delete a model pack by name or index",
	Args:    cobra.MaximumNArgs(1),
	Run:     deleteModelPack,
}

var updateModelPackCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a model pack by name",
	Run:   customModelsCreateNotImplemented,
}

var showModelPackCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show a model pack by name",
	Args:  cobra.MaximumNArgs(1),
	Run:   showModelPack,
}

func init() {
	RootCmd.AddCommand(modelPacksCmd)
	modelPacksCmd.AddCommand(createModelPackCmd)
	modelPacksCmd.AddCommand(deleteModelPackCmd)
	modelPacksCmd.AddCommand(updateModelPackCmd)
	modelPacksCmd.AddCommand(showModelPackCmd)
	modelPacksCmd.Flags().BoolVarP(&customModelPacksOnly, "custom", "c", false, "Only show custom model packs")
	modelPacksCmd.Flags().BoolVarP(&allProperties, "all", "a", false, "Show all properties")
}

func deleteModelPack(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	modelPacks, err := api.Client.ListModelPacks()
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error fetching model packs: %v", err)
		return
	}

	if len(modelPacks) == 0 {
		fmt.Println("🤷‍♂️ No custom model packs")
		return
	}

	var setToDelete *shared.ModelPack

	if len(args) == 1 {
		input := args[0]
		// Try to parse input as index
		index, err := strconv.Atoi(input)
		if err == nil && index > 0 && index <= len(modelPacks) {
			setToDelete = modelPacks[index-1]
		} else {
			// Search by name
			for _, s := range modelPacks {
				if s.Name == input {
					setToDelete = s
					break
				}
			}
		}
	}

	if setToDelete == nil {
		opts := make([]string, len(modelPacks))
		for i, mp := range modelPacks {
			opts[i] = mp.Name
		}

		selected, err := term.SelectFromList("Select a custom model pack:", opts)

		if err != nil {
			term.OutputErrorAndExit("Error selecting model pack: %v", err)
		}

		var selectedIndex int
		for i, opt := range opts {
			if opt == selected {
				selectedIndex = i
				break
			}
		}

		setToDelete = modelPacks[selectedIndex]
	}

	term.StartSpinner("")
	err = api.Client.DeleteModelPack(setToDelete.Id)
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error deleting model pack: %v", err)
		return
	}

	fmt.Printf("✅ Deleted model pack %s\n", color.New(color.Bold, term.ColorHiCyan).Sprint(setToDelete.Name))

	fmt.Println()

	term.PrintCmds("", "model-packs", "model-packs --custom", "model-packs create")
}

func listModelPacks(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	builtInModelPacks := shared.BuiltInModelPacks

	if auth.Current.IsCloud {
		filtered := []*shared.ModelPack{}
		for _, mp := range builtInModelPacks {
			if mp.LocalProvider == "" {
				filtered = append(filtered, mp)
			}
		}
		builtInModelPacks = filtered
	}

	customModelPacks, err := api.Client.ListModelPacks()
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error fetching model packs: %v", err)
		return
	}

	if !customModelPacksOnly {
		color.New(color.Bold, term.ColorHiCyan).Println("🏠 Built-in Model Packs")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(true)
		table.SetRowLine(true)
		table.SetHeader([]string{"Name", "Description"})
		for _, set := range builtInModelPacks {
			table.Append([]string{set.Name, set.Description})
		}
		table.Render()
		fmt.Println()
	}

	if len(customModelPacks) > 0 {
		color.New(color.Bold, term.ColorHiCyan).Println("🛠️  Custom Model Packs")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(true)
		table.SetRowLine(true)
		table.SetHeader([]string{"#", "Name", "Description"})
		for i, set := range customModelPacks {
			table.Append([]string{fmt.Sprintf("%d", i+1), set.Name, set.Description})
		}
		table.Render()

		fmt.Println()
	} else if customModelPacksOnly {
		fmt.Println("🤷‍♂️ No custom model packs")
		fmt.Println()
	}

	if customModelPacksOnly && len(customModelPacks) > 0 {
		term.PrintCmds("", "model-packs create", "model-packs show", "model-packs update", "model-packs delete")
	} else if len(customModelPacks) > 0 {
		term.PrintCmds("", "model-packs --custom", "model-packs create", "model-packs show", "model-packs update", "model-packs delete")
	} else {
		term.PrintCmds("", "model-packs create", "model-packs show")
	}

}

func showModelPack(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	customModelPacks, apiErr := api.Client.ListModelPacks()
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error fetching models: %v", apiErr)
	}

	modelPacks := []*shared.ModelPack{}
	modelPacks = append(modelPacks, customModelPacks...)

	builtInModelPacks := shared.BuiltInModelPacks
	if auth.Current.IsCloud {
		filtered := []*shared.ModelPack{}
		for _, mp := range builtInModelPacks {
			if mp.LocalProvider == "" {
				filtered = append(filtered, mp)
			}
		}
		builtInModelPacks = filtered
	}
	modelPacks = append(modelPacks, builtInModelPacks...)

	var name string
	if len(args) > 0 {
		name = args[0]
	}

	var modelPack *shared.ModelPack

	if name == "" {
		opts := make([]string, len(modelPacks))
		for i, mp := range modelPacks {
			opts[i] = mp.Name
		}

		selected, err := term.SelectFromList("Select a model pack:", opts)
		if err != nil {
			term.OutputErrorAndExit("Error selecting model pack: %v", err)
		}

		for _, mp := range modelPacks {
			if mp.Name == selected {
				modelPack = mp
				break
			}
		}
	} else {
		for _, mp := range modelPacks {
			if mp.Name == name {
				modelPack = mp
				break
			}
		}
	}

	if modelPack == nil {
		term.OutputErrorAndExit("Model pack not found")
		return
	}

	renderModelPack(modelPack, allProperties)

	fmt.Println()

	term.PrintCmds("", "model-packs update", "model-packs delete", "set-model", "set-model default")
}

// func getModelRoleConfig(customModels []*shared.CustomModel, modelRole shared.ModelRole) shared.ModelRoleConfig {
// 	_, modelConfig := getModelWithRoleConfig(customModels, modelRole)
// 	return modelConfig
// }

// func getModelWithRoleConfig(customModels []*shared.CustomModel, modelRole shared.ModelRole) (*shared.CustomModel, shared.ModelRoleConfig) {
// 	role := string(modelRole)

// 	modelId := getModelIdForRole(customModels, modelRole)

// 	temperatureStr, err := term.GetUserStringInputWithDefault("Temperature for "+role+":", fmt.Sprintf("%.1f", shared.DefaultConfigByRole[modelRole].Temperature))
// 	if err != nil {
// 		term.OutputErrorAndExit("Error reading temperature: %v", err)
// 	}
// 	temperature, err := strconv.ParseFloat(temperatureStr, 32)
// 	if err != nil {
// 		term.OutputErrorAndExit("Invalid number for temperature: %v", err)
// 	}

// 	topPStr, err := term.GetUserStringInputWithDefault("Top P for "+role+":", fmt.Sprintf("%.1f", shared.DefaultConfigByRole[modelRole].TopP))
// 	if err != nil {
// 		term.OutputErrorAndExit("Error reading top P: %v", err)
// 	}
// 	topP, err := strconv.ParseFloat(topPStr, 32)
// 	if err != nil {
// 		term.OutputErrorAndExit("Invalid number for top P: %v", err)
// 	}

// 	var reservedOutputTokens int
// 	if modelRole == shared.ModelRoleBuilder || modelRole == shared.ModelRolePlanner || modelRole == shared.ModelRoleWholeFileBuilder {
// 		reservedOutputTokensStr, err := term.GetUserStringInputWithDefault("Reserved output tokens for "+role+":", fmt.Sprintf("%d", model.ReservedOutputTokens))
// 		if err != nil {
// 			term.OutputErrorAndExit("Error reading reserved output tokens: %v", err)
// 		}
// 		reservedOutputTokens, err = strconv.Atoi(reservedOutputTokensStr)
// 		if err != nil {
// 			term.OutputErrorAndExit("Invalid number for reserved output tokens: %v", err)
// 		}
// 	}

// 	return model, shared.ModelRoleConfig{
// 		ModelId:              model.ModelId,
// 		Role:                 modelRole,
// 		Temperature:          float32(temperature),
// 		TopP:                 float32(topP),
// 		ReservedOutputTokens: reservedOutputTokens,
// 	}
// }

// func getPlannerRoleConfig(customModels []*shared.CustomModel) shared.PlannerRoleConfig {
// 	model, modelConfig := getModelWithRoleConfig(customModels, shared.ModelRolePlanner)

// 	return shared.PlannerRoleConfig{
// 		ModelRoleConfig: modelConfig,
// 		PlannerModelConfig: shared.PlannerModelConfig{
// 			MaxConvoTokens: model.DefaultMaxConvoTokens,
// 		},
// 	}
// }

// func getModelIdForRole(customModels []*shared.CustomModel, role shared.ModelRole) shared.ModelId {
// 	color.New(color.Bold).Printf("Select a model for the %s role 👇\n", role)
// 	return lib.SelectModelIdForRole(customModels, role)
// }
