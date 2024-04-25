package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(modelsCmd)
	RootCmd.AddCommand(modelsCmd)
	modelsCmd.AddCommand(listAvailableModelsCmd)
	modelsCmd.AddCommand(createCustomModelCmd)
}

var createCustomModelCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a custom model",
	Run:   createCustomModel,
}

func createCustomModel(cmd *cobra.Command, args []string) {
	model := shared.CustomModel{}
	term.Ask("Enter model name:", &model.ModelName)
	term.Ask("Enter provider:", &model.Provider)
	term.Ask("Enter base URL:", &model.BaseUrl)
	term.Ask("Enter max tokens:", &model.MaxTokens)
	term.Ask("Enter API key environment variable:", &model.ApiKeyEnvVar)
	term.Ask("Enter description (optional):", &model.Description)

	term.StartSpinner("Creating model...")
	err := api.Client.CreateCustomModel(model)
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error creating model: %v", err)
		return
	}

	fmt.Println("Model created successfully.")
}

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Show model settings",
	Run:   models,
	Run:   models,
}

var listAvailableModelsCmd = &cobra.Command{
	Use:   "available",
	Short: "List all available models",
	Run:   listAvailableModels,
}

func models(cmd *cobra.Command, args []string) {

	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	term.StartSpinner("")
	settings, err := api.Client.GetSettings(lib.CurrentPlanId, lib.CurrentBranch)
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error getting settings: %v", err)
		return
	}

	modelSet := settings.ModelSet
	if modelSet == nil {
		modelSet = shared.DefaultModelSet
	}

	color.New(color.Bold, term.ColorHiCyan).Println("🎛️  Current Model Set")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(true)
	table.SetColWidth(64)
	table.SetHeader([]string{modelSet.Name})
	table.Append([]string{modelSet.Description})
	table.Render()
	fmt.Println()

	color.New(color.Bold, term.ColorHiCyan).Println("🤖 Models")
	table = tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Role", "Provider", "Model", "Temperature", "Top P"})

	addModelRow := func(role string, config shared.ModelRoleConfig) {
		table.Append([]string{
			role,
			string(config.BaseModelConfig.Provider),
			config.BaseModelConfig.ModelName,
			fmt.Sprintf("%.1f", config.Temperature),
			fmt.Sprintf("%.1f", config.TopP),
		})
	}

	addModelRow(string(shared.ModelRolePlanner), modelSet.Planner.ModelRoleConfig)
	addModelRow(string(shared.ModelRolePlanSummary), modelSet.PlanSummary)
	addModelRow(string(shared.ModelRoleBuilder), modelSet.Builder)
	addModelRow(string(shared.ModelRoleName), modelSet.Namer)
	addModelRow(string(shared.ModelRoleCommitMsg), modelSet.CommitMsg)
	addModelRow(string(shared.ModelRoleExecStatus), modelSet.ExecStatus)
	table.Render()

	fmt.Println()

	color.New(color.Bold, term.ColorHiCyan).Println("🧠 Planner Defaults")
	table = tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Max Tokens", "Max Convo Tokens", "Reserved Output Tokens"})
	table.Append([]string{
		fmt.Sprintf("%d", modelSet.Planner.BaseModelConfig.MaxTokens),
		fmt.Sprintf("%d", modelSet.Planner.MaxConvoTokens),
		fmt.Sprintf("%d", modelSet.Planner.ReservedOutputTokens),
	})
	table.Render()
	fmt.Println()

	color.New(color.Bold, term.ColorHiCyan).Println("⚙️  Planner Overrides")
	table = tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Name", "Value"})
	if settings.ModelOverrides.MaxTokens == nil {
		table.Append([]string{"Max Tokens", "no override"})
	} else {
		table.Append([]string{"Max Tokens", fmt.Sprintf("%d", *settings.ModelOverrides.MaxTokens)})
	}
	if settings.ModelOverrides.MaxConvoTokens == nil {
		table.Append([]string{"Max Convo Tokens", "no override"})
	} else {
		table.Append([]string{"Max Convo Tokens", fmt.Sprintf("%d", *settings.ModelOverrides.MaxConvoTokens)})
	}
	if settings.ModelOverrides.ReservedOutputTokens == nil {
		table.Append([]string{"Reserved Output Tokens", "no override"})
	} else {
		table.Append([]string{"Reserved Output Tokens", fmt.Sprintf("%d", *settings.ModelOverrides.ReservedOutputTokens)})
	}
	table.Render()

	fmt.Println()
	term.PrintCmds("", "set-model")

	term.PrintCmds("", "set-model")
}

func listAvailableModels(cmd *cobra.Command, args []string) {
	term.StartSpinner("Fetching models...")
	models, err := api.Client.ListAvailableModels()
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error fetching models: %v", err)
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Provider", "Max Tokens", "Description"})
	for _, model := range models {
		table.Append([]string{model.ModelName, string(model.Provider), strconv.Itoa(model.MaxTokens), model.Description})
	}
	table.Render()
}


