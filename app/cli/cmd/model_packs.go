package cmd

import (
	"fmt"
	"os"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/lib"
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
	Run:   createModelPack,
}

var deleteModelPackCmd = &cobra.Command{
	Use:     "delete [name-or-index]",
	Aliases: []string{"rm"},
	Short:   "Delete a model pack by name or index",
	Args:    cobra.MaximumNArgs(1),
	Run:     deleteModelPack,
}

var updateModelPackCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update a model pack by name",
	Args:  cobra.MaximumNArgs(1),
	Run:   updateModelPack,
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

func createModelPack(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	mp := &shared.ModelPack{}

	name, err := term.GetRequiredUserStringInput("Enter model pack name:")
	if err != nil {
		term.OutputErrorAndExit("Error reading model pack name: %v", err)
		return
	}
	mp.Name = name

	description, err := term.GetUserStringInput("Enter description:")
	if err != nil {
		term.OutputErrorAndExit("Error reading description: %v", err)
		return
	}
	mp.Description = description

	term.StartSpinner("")
	customModels, apiErr := api.Client.ListCustomModels()
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error fetching models: %v", apiErr)
	}

	// Selecting models for each role
	mp.Planner = getPlannerRoleConfig(customModels)

	contextLoader := getModelRoleConfig(customModels, shared.ModelRoleArchitect)
	mp.Architect = &contextLoader

	coder := getModelRoleConfig(customModels, shared.ModelRoleCoder)
	mp.Coder = &coder

	mp.Builder = getModelRoleConfig(customModels, shared.ModelRoleBuilder)

	wholeFileBuilder := getModelRoleConfig(customModels, shared.ModelRoleWholeFileBuilder)
	mp.WholeFileBuilder = &wholeFileBuilder

	mp.Namer = getModelRoleConfig(customModels, shared.ModelRoleName)
	mp.CommitMsg = getModelRoleConfig(customModels, shared.ModelRoleCommitMsg)

	mp.PlanSummary = getModelRoleConfig(customModels, shared.ModelRolePlanSummary)
	mp.ExecStatus = getModelRoleConfig(customModels, shared.ModelRoleExecStatus)

	term.StartSpinner("")
	apiErr = api.Client.CreateModelPack(mp)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error creating model pack: %v", apiErr.Msg)
		return
	}

	fmt.Println("✅ Created model pack", color.New(color.Bold, term.ColorHiCyan).Sprint(mp.Name))

	fmt.Println()

	term.PrintCmds("", "model-packs", "model-packs --custom", "model-packs delete")
}

func updateModelPack(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	modelPacks, apiErr := api.Client.ListModelPacks()

	if apiErr != nil {
		term.StopSpinner()
		term.OutputErrorAndExit("Error fetching models: %v", apiErr)
	}

	if len(modelPacks) == 0 {
		term.StopSpinner()
		fmt.Println("🤷‍♂️ No model packs")
		return
	}

	customModels, apiErr := api.Client.ListCustomModels()
	term.StopSpinner()
	if apiErr != nil {
		term.OutputErrorAndExit("Error fetching models: %v", apiErr)
	}

	var toUpdate *shared.ModelPack

	var name string
	if len(args) > 0 {
		name = args[0]
	}

	if name == "" {
		opts := make([]string, len(modelPacks))
		for i, mp := range modelPacks {
			opts[i] = mp.Name
		}

		selected, err := term.SelectFromList("Select a custom model pack:", opts)
		if err != nil {
			term.OutputErrorAndExit("Error selecting model pack: %v", err)
		}

		for _, mp := range modelPacks {
			if mp.Name == selected {
				toUpdate = mp
				break
			}
		}
	}

	if toUpdate == nil {
		term.OutputErrorAndExit("Model pack not found")
		return
	}

	var role shared.ModelRole

	roleOpts := make([]string, len(shared.AllModelRoles))
	for i, role := range shared.AllModelRoles {
		roleOpts[i] = string(role)
	}

	renderModelPack(toUpdate)

	for {
		selected, err := term.SelectFromList("Select a role to update:", roleOpts)
		if err != nil {
			term.OutputErrorAndExit("Error selecting role: %v", err)
		}

		role = shared.ModelRole(selected)

		switch role {
		case shared.ModelRolePlanner:
			toUpdate.Planner = getPlannerRoleConfig(customModels)
		case shared.ModelRoleArchitect:
			contextLoader := getModelRoleConfig(customModels, shared.ModelRoleArchitect)
			toUpdate.Architect = &contextLoader
		case shared.ModelRoleCoder:
			coder := getModelRoleConfig(customModels, shared.ModelRoleCoder)
			toUpdate.Coder = &coder
		case shared.ModelRoleBuilder:
			builder := getModelRoleConfig(customModels, shared.ModelRoleBuilder)
			toUpdate.Builder = builder
		case shared.ModelRoleWholeFileBuilder:
			wholeFileBuilder := getModelRoleConfig(customModels, shared.ModelRoleWholeFileBuilder)
			toUpdate.WholeFileBuilder = &wholeFileBuilder
		case shared.ModelRolePlanSummary:
			toUpdate.PlanSummary = getModelRoleConfig(customModels, role)
		case shared.ModelRoleExecStatus:
			toUpdate.ExecStatus = getModelRoleConfig(customModels, role)
		case shared.ModelRoleCommitMsg:
			toUpdate.CommitMsg = getModelRoleConfig(customModels, role)
		case shared.ModelRoleName:
			toUpdate.Namer = getModelRoleConfig(customModels, role)
		}

		updateOpt := "Update another role"
		saveOpt := "Save model pack and exit"
		opts := []string{updateOpt, saveOpt}

		selected, err = term.SelectFromList("Finished editing?", opts)
		if err != nil {
			term.OutputErrorAndExit("Error selecting option: %v", err)
		}

		if selected == saveOpt {
			break
		}
	}

	term.StartSpinner("")
	apiErr = api.Client.UpdateModelPack(toUpdate)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error updating model pack: %v", apiErr.Msg)
	}

	fmt.Println("✅ Model pack updated")

	fmt.Println()

	term.PrintCmds("", "model-packs show", "set-model", "set-model default")
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
	modelPacks = append(modelPacks, shared.BuiltInModelPacks...)

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

	renderModelPack(modelPack)

	fmt.Println()

	term.PrintCmds("", "model-packs update", "model-packs delete", "set-model", "set-model default")
}

func getModelRoleConfig(customModels []*shared.AvailableModel, modelRole shared.ModelRole) shared.ModelRoleConfig {
	_, modelConfig := getModelWithRoleConfig(customModels, modelRole)
	return modelConfig
}

func getModelWithRoleConfig(customModels []*shared.AvailableModel, modelRole shared.ModelRole) (*shared.AvailableModel, shared.ModelRoleConfig) {
	role := string(modelRole)

	model := getModelForRole(customModels, modelRole)

	temperatureStr, err := term.GetUserStringInputWithDefault("Temperature for "+role+":", fmt.Sprintf("%.1f", shared.DefaultConfigByRole[modelRole].Temperature))
	if err != nil {
		term.OutputErrorAndExit("Error reading temperature: %v", err)
	}
	temperature, err := strconv.ParseFloat(temperatureStr, 32)
	if err != nil {
		term.OutputErrorAndExit("Invalid number for temperature: %v", err)
	}

	topPStr, err := term.GetUserStringInputWithDefault("Top P for "+role+":", fmt.Sprintf("%.1f", shared.DefaultConfigByRole[modelRole].TopP))
	if err != nil {
		term.OutputErrorAndExit("Error reading top P: %v", err)
	}
	topP, err := strconv.ParseFloat(topPStr, 32)
	if err != nil {
		term.OutputErrorAndExit("Invalid number for top P: %v", err)
	}

	var reservedOutputTokens int
	if modelRole == shared.ModelRoleBuilder || modelRole == shared.ModelRolePlanner || modelRole == shared.ModelRoleWholeFileBuilder {
		reservedOutputTokensStr, err := term.GetUserStringInputWithDefault("Reserved output tokens for "+role+":", fmt.Sprintf("%d", model.ReservedOutputTokens))
		if err != nil {
			term.OutputErrorAndExit("Error reading reserved output tokens: %v", err)
		}
		reservedOutputTokens, err = strconv.Atoi(reservedOutputTokensStr)
		if err != nil {
			term.OutputErrorAndExit("Invalid number for reserved output tokens: %v", err)
		}
	}

	return model, shared.ModelRoleConfig{
		Role:                 modelRole,
		BaseModelConfig:      model.BaseModelConfig,
		Temperature:          float32(temperature),
		TopP:                 float32(topP),
		ReservedOutputTokens: reservedOutputTokens,
	}
}

func getPlannerRoleConfig(customModels []*shared.AvailableModel) shared.PlannerRoleConfig {
	model, modelConfig := getModelWithRoleConfig(customModels, shared.ModelRolePlanner)

	return shared.PlannerRoleConfig{
		ModelRoleConfig: modelConfig,
		PlannerModelConfig: shared.PlannerModelConfig{
			MaxConvoTokens: model.DefaultMaxConvoTokens,
		},
	}
}

func getModelForRole(customModels []*shared.AvailableModel, role shared.ModelRole) *shared.AvailableModel {
	color.New(color.Bold).Printf("Select a model for the %s role 👇\n", role)
	return lib.SelectModelForRole(customModels, role, false)
}
