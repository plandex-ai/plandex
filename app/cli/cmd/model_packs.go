package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"
	"strconv"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/plandex/plandex/shared"
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

func init() {
	RootCmd.AddCommand(modelPacksCmd)
	modelPacksCmd.AddCommand(createModelPackCmd)
	modelPacksCmd.AddCommand(deleteModelPackCmd)

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
		term.PrintCmds("", "model-packs create", "model-packs delete")
	} else if len(customModelPacks) > 0 {
		term.PrintCmds("", "model-packs --custom", "model-packs create", "model-packs delete")
	} else {
		term.PrintCmds("", "model-packs create")
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
	mp.PlanSummary = getModelRoleConfig(customModels, shared.ModelRolePlanSummary)
	mp.Builder = getModelRoleConfig(customModels, shared.ModelRoleBuilder)
	mp.Namer = getModelRoleConfig(customModels, shared.ModelRoleName)
	mp.CommitMsg = getModelRoleConfig(customModels, shared.ModelRoleCommitMsg)
	mp.ExecStatus = getModelRoleConfig(customModels, shared.ModelRoleExecStatus)
	verifier := getModelRoleConfig(customModels, shared.ModelRoleVerifier)
	mp.Verifier = &verifier
	autoFix := getModelRoleConfig(customModels, shared.ModelRoleAutoFix)
	mp.AutoFix = &autoFix

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

	return model, shared.ModelRoleConfig{
		Role:            modelRole,
		BaseModelConfig: model.BaseModelConfig,
		Temperature:     float32(temperature),
		TopP:            float32(topP),
	}
}

func getPlannerRoleConfig(customModels []*shared.AvailableModel) shared.PlannerRoleConfig {
	model, modelConfig := getModelWithRoleConfig(customModels, shared.ModelRolePlanner)

	return shared.PlannerRoleConfig{
		ModelRoleConfig: modelConfig,
		PlannerModelConfig: shared.PlannerModelConfig{
			MaxConvoTokens:       model.DefaultMaxConvoTokens,
			ReservedOutputTokens: model.DefaultReservedOutputTokens,
		},
	}
}

func getModelForRole(customModels []*shared.AvailableModel, role shared.ModelRole) *shared.AvailableModel {
	color.New(color.Bold).Printf("Select a model for the %s role 👇\n", role)
	return lib.SelectModelForRole(customModels, role, false)
}
