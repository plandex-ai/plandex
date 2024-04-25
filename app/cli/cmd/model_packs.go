package cmd

import (
	"fmt"
	"plandex/api"
	"plandex/term"
	"strconv"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var modelSetsCmd = &cobra.Command{
	Use:   "model-packs",
	Short: "Manage model packs",
}

var listModelSetsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all model packs",
	Run:   listModelSets,
}

var createModelSetCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a model pack",
	Run:   createModelSet,
}

func init() {
	RootCmd.AddCommand(modelSetsCmd)
	modelSetsCmd.AddCommand(listModelSetsCmd)
	modelSetsCmd.AddCommand(createModelSetCmd)
	modelSetsCmd.AddCommand(deleteModelSetCmd)
}

var deleteModelSetCmd = &cobra.Command{
	Use:   "delete [name-or-index]",
	Short: "Delete a model set by name or index",
	Args:  cobra.MaximumNArgs(1),
	Run:   deleteModelSet,
}

func deleteModelSet(cmd *cobra.Command, args []string) {
	term.StartSpinner("")
	modelSets, err := api.Client.ListModelSets()
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error fetching model sets: %v", err)
		return
	}

	if len(modelSets) == 0 {
		fmt.Println("No model sets available to delete.")
		return
	}

	var setToDelete *shared.ModelSet

	if len(args) == 1 {
		input := args[0]
		// Try to parse input as index
		index, err := strconv.Atoi(input)
		if err == nil && index > 0 && index <= len(modelSets) {
			setToDelete = modelSets[index-1]
		} else {
			// Search by name
			for _, s := range modelSets {
				if s.Name == input {
					setToDelete = s
					break
				}
			}
		}
	}

	if setToDelete == nil {
		fmt.Println("Select a model set to delete:")
		for i, set := range modelSets {
			fmt.Printf("%d: %s\n", i+1, set.Name)
		}
		var selectedIndex int
		fmt.Scanln(&selectedIndex)
		if selectedIndex < 1 || selectedIndex > len(modelSets) {
			fmt.Println("Invalid selection.")
			return
		}
		setToDelete = modelSets[selectedIndex-1]
	}

	term.StartSpinner("")
	err = api.Client.DeleteModelSet(setToDelete.Id)
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error deleting model set: %v", err)
		return
	}

	fmt.Printf("Model set '%s' deleted successfully.\n", setToDelete.Name)
}

func listModelSets(cmd *cobra.Command, args []string) {
	term.StartSpinner("Fetching model sets...")
	builtInModelSets := shared.BuiltInModelSets // Assuming this is already defined in the shared package
	customModelSets, err := api.Client.ListModelSets()
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error fetching model sets: %v", err)
		return
	}

	fmt.Println("Available Model Sets:")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Description", "Type"})

	for _, set := range builtInModelSets {
		table.Append([]string{set.Name, set.Description, "Built-in"})
	}

	for _, set := range customModelSets {
		table.Append([]string{set.Name, set.Description, "Custom"})
	}

	table.Render()
}

func createModelSet(cmd *cobra.Command, args []string) {
	set := &shared.ModelSet{}

	name, err := term.GetUserStringInput("Enter model set name:")
	if err != nil {
		term.OutputErrorAndExit("Error reading model set name: %v", err)
		return
	}
	set.Name = name

	description, err := term.GetUserStringInput("Enter description:")
	if err != nil {
		term.OutputErrorAndExit("Error reading description: %v", err)
		return
	}
	set.Description = description

	// Selecting models for each role
	set.Planner = getPlannerRoleConfig()
	set.PlanSummary = getModelRoleConfig(shared.ModelRolePlanSummary)
	set.Builder = getModelRoleConfig(shared.ModelRoleBuilder)
	set.Namer = getModelRoleConfig(shared.ModelRoleName)
	set.CommitMsg = getModelRoleConfig(shared.ModelRoleCommitMsg)
	set.ExecStatus = getModelRoleConfig(shared.ModelRoleExecStatus)

	term.StartSpinner("")
	apiErr := api.Client.CreateModelSet(set)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error creating model set: %v", apiErr.Msg)
		return
	}

	fmt.Println("✅ Created model pack", color.New(color.Bold, term.ColorHiCyan).Sprint())
}

func getModelRoleConfig(modelRole shared.ModelRole) shared.ModelRoleConfig {
	role := string(modelRole)

	baseModelConfig := selectModelForRole(modelRole)

	temperatureStr, err := term.GetUserStringInput("Enter the temperature for " + role + ":")
	if err != nil {
		term.OutputErrorAndExit("Error reading temperature: %v", err)
	}
	temperature, err := strconv.ParseFloat(temperatureStr, 32)
	if err != nil {
		term.OutputErrorAndExit("Invalid number for temperature: %v", err)
	}

	topPStr, err := term.GetUserStringInput("Enter the top P for " + role + ":")
	if err != nil {
		term.OutputErrorAndExit("Error reading top P: %v", err)
	}
	topP, err := strconv.ParseFloat(topPStr, 32)
	if err != nil {
		term.OutputErrorAndExit("Invalid number for top P: %v", err)
	}

	return shared.ModelRoleConfig{
		BaseModelConfig: baseModelConfig,
		Temperature:     float32(temperature),
		TopP:            float32(topP),
	}
}

func getPlannerRoleConfig() shared.PlannerRoleConfig {
	modelConfig := getModelRoleConfig(shared.ModelRolePlanner)

	maxConvoTokensStr, err := term.GetUserStringInput("Max Convo Tokens:")
	if err != nil {
		term.OutputErrorAndExit("Error reading max-convo-tokens: %v", err)
	}
	maxConvoTokens, err := strconv.Atoi(maxConvoTokensStr)
	if err != nil {
		term.OutputErrorAndExit("Invalid number for max-convo-tokens: %v", err)
	}

	reservedOutputTokensStr, err := term.GetUserStringInput("Reserved Output Tokens:")
	if err != nil {
		term.OutputErrorAndExit("Error reading reserved-output-tokens: %v", err)
	}
	reservedOutputTokens, err := strconv.Atoi(reservedOutputTokensStr)
	if err != nil {
		term.OutputErrorAndExit("Invalid number for reserved-output-tokens: %v", err)
	}

	return shared.PlannerRoleConfig{
		ModelRoleConfig: modelConfig,
		PlannerModelConfig: shared.PlannerModelConfig{
			MaxConvoTokens:       maxConvoTokens,
			ReservedOutputTokens: reservedOutputTokens,
		},
	}
}

func selectModelForRole(role shared.ModelRole) shared.BaseModelConfig {
	term.StartSpinner("")
	customModels, apiErr := api.Client.ListCustomModels()
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error fetching models: %v", apiErr)
	}

	opts := make([]string, len(customModels))
	for i, model := range customModels {
		opts[i] = fmt.Sprintf("%s (%s)", model.ModelName, model.Provider)
	}

	selected, err := term.SelectFromList(fmt.Sprintf("Select a model for the %s role:", role), opts)
	if err != nil {
		term.OutputErrorAndExit("Error selecting model: %v", err)
	}

	var idx int
	for i, opt := range opts {
		if opt == selected {
			idx = i
			break
		}
	}

	return customModels[idx].BaseModelConfig
}

