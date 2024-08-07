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

var customModelsOnly bool

func init() {
	RootCmd.AddCommand(modelsCmd)
	modelsCmd.AddCommand(listAvailableModelsCmd)
	modelsCmd.AddCommand(createCustomModelCmd)
	modelsCmd.AddCommand(deleteCustomModelCmd)
	modelsCmd.AddCommand(defaultModelsCmd)

	listAvailableModelsCmd.Flags().BoolVarP(&customModelsOnly, "custom", "c", false, "List custom models only")
}

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Show plan model settings",
	Run:   models,
}

var defaultModelsCmd = &cobra.Command{
	Use:   "default",
	Short: "Show default model settings for new plans",
	Run:   defaultModels,
}

var listAvailableModelsCmd = &cobra.Command{
	Use:     "available",
	Aliases: []string{"avail"},
	Short:   "List all available models",
	Run:     listAvailableModels,
}

var createCustomModelCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"create"},
	Short:   "Add a custom model",
	Run:     createCustomModel,
}

var deleteCustomModelCmd = &cobra.Command{
	Use:     "rm",
	Aliases: []string{"remove", "delete"},
	Short:   "Remove a custom model",
	Args:    cobra.MaximumNArgs(1),
	Run:     deleteCustomModel,
}

func createCustomModel(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	model := &shared.AvailableModel{
		BaseModelConfig: shared.BaseModelConfig{
			ModelCompatibility: shared.ModelCompatibility{
				IsOpenAICompatible: true,
			},
		},
	}

	opts := shared.AllModelProviders
	provider, err := term.SelectFromList("Select provider:", opts)

	if err != nil {
		term.OutputErrorAndExit("Error selecting provider: %v", err)
		return
	}
	model.Provider = shared.ModelProvider(provider)

	if model.Provider == shared.ModelProviderCustom {
		customProvider, err := term.GetRequiredUserStringInput("Custom provider:")
		if err != nil {
			term.OutputErrorAndExit("Error reading custom provider: %v", err)
			return
		}
		model.CustomProvider = &customProvider
	}

	fmt.Println("For model name, be sure to enter the exact, case-sensitive name of the model as it appears in the provider's API docs. Ex: 'gpt-4-turbo', 'meta-llama/Llama-3-70b-chat-hf'")
	modelName, err := term.GetRequiredUserStringInput("Model name:")
	if err != nil {
		term.OutputErrorAndExit("Error reading model name: %v", err)
		return
	}
	model.ModelName = modelName

	fmt.Println("Add a human friendly description if you want to.")
	description, err := term.GetUserStringInput("Description (optional):")
	if err != nil {
		term.OutputErrorAndExit("Error reading description: %v", err)
		return
	}
	model.Description = description

	if model.Provider == shared.ModelProviderCustom {
		baseUrl, err := term.GetRequiredUserStringInput("Base URL:")
		if err != nil {
			term.OutputErrorAndExit("Error reading base URL: %v", err)
			return
		}
		model.BaseUrl = baseUrl
	} else {
		model.BaseUrl = shared.BaseUrlByProvider[model.Provider]
	}

	apiKeyDefault := shared.ApiKeyByProvider[model.Provider]
	var apiKeyEnvVar string
	if apiKeyDefault == "" {
		apiKeyEnvVar, err = term.GetRequiredUserStringInput("API key environment variable:")
	} else {
		apiKeyEnvVar, err = term.GetUserStringInputWithDefault("API key environment variable:", apiKeyDefault)
	}

	if err != nil {
		term.OutputErrorAndExit("Error reading API key environment variable: %v", err)
		return
	}
	model.ApiKeyEnvVar = apiKeyEnvVar

	fmt.Println("Max Tokens is the total maximum context size of the model.")

	maxTokensStr, err := term.GetRequiredUserStringInput("Max Tokens:")
	if err != nil {
		term.OutputErrorAndExit("Error reading max tokens: %v", err)
		return
	}
	maxTokens, err := strconv.Atoi(maxTokensStr)
	if err != nil {
		term.OutputErrorAndExit("Invalid number for max tokens: %v", err)
		return
	}
	model.MaxTokens = maxTokens

	fmt.Println("'Default Max Convo Tokens' is the default maximum size a conversation can reach in the 'planner' role before it is shortened by summarization. For models with 8k context, ~2500 is recommended. For 128k context, ~10000 is recommended.")
	maxConvoTokensStr, err := term.GetRequiredUserStringInput("Default Max Convo Tokens:")
	if err != nil {
		term.OutputErrorAndExit("Error reading max convo tokens: %v", err)
		return
	}
	maxConvoTokens, err := strconv.Atoi(maxConvoTokensStr)
	if err != nil {
		term.OutputErrorAndExit("Invalid number for max convo tokens: %v", err)
		return
	}
	model.DefaultMaxConvoTokens = maxConvoTokens

	fmt.Println("'Default Reserved Output Tokens' is the default number of tokens reserved for model output in the 'planner' role. This ensures the model has enough tokens to generate a response. For models with 8k context, ~1000 is recommended. For 128k context, ~4000 is recommended.")
	reservedOutputTokensStr, err := term.GetRequiredUserStringInput("Default Reserved Output Tokens:")
	if err != nil {
		term.OutputErrorAndExit("Error reading reserved output tokens: %v", err)
		return
	}
	reservedOutputTokens, err := strconv.Atoi(reservedOutputTokensStr)
	if err != nil {
		term.OutputErrorAndExit("Invalid number for reserved output tokens: %v", err)
		return
	}
	model.DefaultReservedOutputTokens = reservedOutputTokens

	model.ModelCompatibility.HasStreaming, err = term.ConfirmYesNo("Is streaming supported?")
	if err != nil {
		term.OutputErrorAndExit("Error confirming streaming support: %v", err)
		return
	}
	model.ModelCompatibility.HasJsonResponseMode, err = term.ConfirmYesNo("Is JSON mode supported?")
	if err != nil {
		term.OutputErrorAndExit("Error confirming JSON mode support: %v", err)
		return
	}
	model.ModelCompatibility.HasFunctionCalling, err = term.ConfirmYesNo("Is function calling supported?")
	if err != nil {
		term.OutputErrorAndExit("Error confirming function calling support: %v", err)
		return
	}
	model.ModelCompatibility.HasStreamingFunctionCalls, err = term.ConfirmYesNo("Are streaming function calls supported?")
	if err != nil {
		term.OutputErrorAndExit("Error confirming streaming function calls support: %v", err)
		return
	}

	model.ModelCompatibility.HasImageSupport, err = term.ConfirmYesNo("Is multi-modal image support enabled?")
	if err != nil {
		term.OutputErrorAndExit("Error confirming image support: %v", err)
		return
	}

	term.StartSpinner("")
	apiErr := api.Client.CreateCustomModel(model)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error adding model: %v", apiErr.Msg)
		return
	}

	fmt.Println("✅ Added custom model", color.New(color.Bold, term.ColorHiCyan).Sprint(string(model.Provider)+" → "+model.ModelName))
}

func models(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	term.StartSpinner("")

	plan, err := api.Client.GetPlan(lib.CurrentPlanId)

	if err != nil {
		term.StopSpinner()
		term.OutputErrorAndExit("Error getting plan: %v", err)
		return
	}

	settings, err := api.Client.GetSettings(lib.CurrentPlanId, lib.CurrentBranch)
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error getting settings: %v", err)
		return
	}

	title := fmt.Sprintf("%s Model Settings", color.New(color.Bold, term.ColorHiGreen).Sprint(plan.Name))

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.Append([]string{title})
	table.Render()
	fmt.Println()

	renderSettings(settings)

	term.PrintCmds("", "set-model", "models available", "models default")
}

func defaultModels(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	settings, err := api.Client.GetOrgDefaultSettings()
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error getting default model settings: %v", err)
		return
	}

	title := fmt.Sprintf("%s Model Settings", color.New(color.Bold, term.ColorHiGreen).Sprint("Org-Wide Default"))
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.Append([]string{title})
	table.Render()
	fmt.Println()

	renderSettings(settings)

	term.PrintCmds("", "set-model default", "models available", "models")
}

func listAvailableModels(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")

	customModels, err := api.Client.ListCustomModels()

	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error fetching custom models: %v", err)
		return
	}

	if !customModelsOnly {
		color.New(color.Bold, term.ColorHiCyan).Println("🏠 Built-in Models")
		builtIn := shared.AvailableModels
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(false)
		table.SetHeader([]string{"Provider", "Name", "🪙", "🔑"})
		for _, model := range builtIn {
			table.Append([]string{string(model.Provider), model.ModelName, strconv.Itoa(model.MaxTokens), model.ApiKeyEnvVar})
		}
		table.Render()
		fmt.Println()
	}

	if len(customModels) > 0 {
		color.New(color.Bold, term.ColorHiCyan).Println("🛠️  Custom Models")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(false)
		table.SetHeader([]string{"#", "Provider", "Name", "🪙", "🔑"})
		for i, model := range customModels {
			var provider string
			p := model.Provider
			if p == shared.ModelProviderCustom {
				provider = *model.CustomProvider
			} else {
				provider = string(p)
			}

			table.Append([]string{fmt.Sprintf("%d", i+1), provider, model.ModelName, strconv.Itoa(model.MaxTokens), model.ApiKeyEnvVar})
		}
		table.Render()
	} else if customModelsOnly {
		fmt.Println("🤷‍♂️ No custom models")
	}
	fmt.Println()

	if customModelsOnly {
		if len(customModels) > 0 {
			term.PrintCmds("", "models", "set-model", "models add", "models delete")
		} else {
			term.PrintCmds("", "models add")
		}
	} else {
		term.PrintCmds("", "models available --custom", "models", "set-model", "models add", "models delete")
	}
}

func deleteCustomModel(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	models, apiErr := api.Client.ListCustomModels()
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error fetching custom models: %v", apiErr)
		return
	}

	if len(models) == 0 {
		fmt.Println("🤷‍♂️ No custom models")
		fmt.Println()
		term.PrintCmds("", "models add")
		return
	}

	var modelToDelete *shared.AvailableModel

	if len(args) == 1 {
		input := args[0]
		// Try to parse input as index
		index, err := strconv.Atoi(input)
		if err == nil && index > 0 && index <= len(models) {
			modelToDelete = models[index-1]
		} else {
			// Search by name
			for _, m := range models {
				if m.ModelName == input {
					modelToDelete = m
					break
				}
			}
		}
	}

	if modelToDelete == nil {
		opts := make([]string, len(models))
		for i, model := range models {
			var provider string
			p := model.Provider
			if p == shared.ModelProviderCustom {
				provider = *model.CustomProvider
			} else {
				provider = string(p)
			}
			opts[i] = provider + " → " + model.ModelName
		}

		selected, err := term.SelectFromList("Select model to delete:", opts)

		if err != nil {
			term.OutputErrorAndExit("Error selecting model: %v", err)
			return
		}

		var selectedIndex int
		for i, opt := range opts {
			if opt == selected {
				selectedIndex = i
				break
			}
		}

		modelToDelete = models[selectedIndex]
	}

	term.StartSpinner("")
	apiErr = api.Client.DeleteAvailableModel(modelToDelete.Id)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error deleting custom model: %v", apiErr)
		return
	}

	fmt.Printf("✅ Deleted custom model %s\n", color.New(color.Bold, term.ColorHiCyan).Sprint(string(modelToDelete.Provider)+" → "+modelToDelete.ModelName))

	fmt.Println()
	term.PrintCmds("", "models available", "models add")
}

func renderSettings(settings *shared.PlanSettings) {
	modelPack := settings.ModelPack

	color.New(color.Bold, term.ColorHiCyan).Println("🎛️  Current Model Pack")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(true)
	table.SetColWidth(64)
	table.SetHeader([]string{modelPack.Name})
	table.Append([]string{modelPack.Description})
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

	addModelRow(string(shared.ModelRolePlanner), modelPack.Planner.ModelRoleConfig)
	addModelRow(string(shared.ModelRolePlanSummary), modelPack.PlanSummary)
	addModelRow(string(shared.ModelRoleBuilder), modelPack.Builder)
	addModelRow(string(shared.ModelRoleName), modelPack.Namer)
	addModelRow(string(shared.ModelRoleCommitMsg), modelPack.CommitMsg)
	addModelRow(string(shared.ModelRoleExecStatus), modelPack.ExecStatus)
	addModelRow(string(shared.ModelRoleVerifier), modelPack.GetVerifier())
	addModelRow(string(shared.ModelRoleAutoFix), modelPack.GetAutoFix())
	table.Render()

	fmt.Println()

	color.New(color.Bold, term.ColorHiCyan).Println("🧠 Planner Defaults")
	table = tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Max Tokens", "Max Convo Tokens", "Reserved Output Tokens"})
	table.Append([]string{
		fmt.Sprintf("%d", modelPack.Planner.BaseModelConfig.MaxTokens),
		fmt.Sprintf("%d", modelPack.Planner.MaxConvoTokens),
		fmt.Sprintf("%d", modelPack.Planner.ReservedOutputTokens),
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
}
