package cmd

import (
	"encoding/json"
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

var customModelsOnly bool

var allProperties bool
var jsonFile string

func init() {
	RootCmd.AddCommand(modelsCmd)

	modelsCmd.Flags().BoolVarP(&allProperties, "all", "a", false, "Show all properties")

	modelsCmd.AddCommand(listAvailableModelsCmd)
	modelsCmd.AddCommand(createCustomModelCmd)
	modelsCmd.AddCommand(deleteCustomModelCmd)
	modelsCmd.AddCommand(defaultModelsCmd)

	createCustomModelCmd.Flags().StringVar(&jsonFile, "json", "", "Path to a JSON file containing model configuration")

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

	var model *shared.CustomModel

	if jsonFile != "" {
		// Load model from JSON file
		jsonData, err := os.ReadFile(jsonFile)
		if err != nil {
			term.OutputErrorAndExit("Error reading JSON file: %v", err)
			return
		}

		// Parse JSON into a map
		var jsonMap map[string]interface{}
		if err := json.Unmarshal(jsonData, &jsonMap); err != nil {
			term.OutputErrorAndExit("Error parsing JSON: %v", err)
			return
		}

		// Convert to CustomModel
		model = &shared.CustomModel{
			BaseModelShared: shared.BaseModelShared{
				ModelCompatibility: shared.ModelCompatibility{},
			},
		}

		// Extract required fields
		if modelId, ok := jsonMap["modelId"].(string); ok {
			model.ModelId = shared.ModelId(modelId)
		} else {
			term.OutputErrorAndExit("Missing or invalid 'modelId' in JSON")
			return
		}

		// Extract optional fields
		if description, ok := jsonMap["description"].(string); ok {
			model.Description = description
		}

		// Extract numeric fields
		if maxTokens, ok := jsonMap["maxTokens"].(float64); ok {
			model.MaxTokens = int(maxTokens)
		} else {
			term.OutputErrorAndExit("Missing or invalid 'maxTokens' in JSON")
			return
		}

		if maxConvoTokens, ok := jsonMap["defaultMaxConvoTokens"].(float64); ok {
			model.DefaultMaxConvoTokens = int(maxConvoTokens)
		} else {
			term.OutputErrorAndExit("Missing or invalid 'defaultMaxConvoTokens' in JSON")
			return
		}

		if maxOutputTokens, ok := jsonMap["maxOutputTokens"].(float64); ok {
			model.MaxOutputTokens = int(maxOutputTokens)
		} else {
			term.OutputErrorAndExit("Missing or invalid 'maxOutputTokens' in JSON")
			return
		}

		if reservedOutputTokens, ok := jsonMap["reservedOutputTokens"].(float64); ok {
			model.ReservedOutputTokens = int(reservedOutputTokens)
		} else {
			term.OutputErrorAndExit("Missing or invalid 'reservedOutputTokens' in JSON")
			return
		}

		// Extract output format
		if outputFormat, ok := jsonMap["preferredOutputFormat"].(string); ok {
			model.PreferredOutputFormat = shared.ModelOutputFormat(outputFormat)
		} else {
			term.OutputErrorAndExit("Missing or invalid 'preferredOutputFormat' in JSON")
			return
		}

		// Extract image support
		if hasImageSupport, ok := jsonMap["hasImageSupport"].(bool); ok {
			model.HasImageSupport = hasImageSupport
		} else {
			// Default to false if not specified
			model.HasImageSupport = false
		}
	} else {
		// Interactive mode
		model = &shared.CustomModel{
			BaseModelShared: shared.BaseModelShared{
				ModelCompatibility: shared.ModelCompatibility{},
			},
		}

		fmt.Println("For model id, use something like 'openai/gpt-4.1' that uniquely identifies the model. This doesn't need to match the model name for any particular provider (this is set when you add providers for the model).")
		modelId, err := term.GetRequiredUserStringInput("Model id:")
		if err != nil {
			term.OutputErrorAndExit("Error reading model id: %v", err)
			return
		}
		model.ModelId = shared.ModelId(modelId)

		fmt.Println("Add a human friendly description if you want to.")
		description, err := term.GetUserStringInput("Description (optional):")
		if err != nil {
			term.OutputErrorAndExit("Error reading description: %v", err)
			return
		}
		model.Description = description

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

		fmt.Println("'Default Max Convo Tokens' is the default maximum size a conversation can reach in the 'planner' role before it is shortened by summarization. For 128k context, ~10000 is recommended. For 200k context, ~15000 is recommended.")

		var defaultMaxConvoTokens int

		if maxTokens >= 180000 {
			defaultMaxConvoTokens = 15000
		} else if maxTokens >= 100000 {
			defaultMaxConvoTokens = 10000
		}

		var maxConvoTokensStr string
		if defaultMaxConvoTokens == 0 {
			maxConvoTokensStr, err = term.GetRequiredUserStringInput("Default Max Convo Tokens:")
		} else {
			maxConvoTokensStr, err = term.GetRequiredUserStringInputWithDefault("Default Max Convo Tokens:", strconv.Itoa(defaultMaxConvoTokens))
		}

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

		fmt.Println("'Max Output Tokens' is the hard limit on output length for the model. Check with the model provider for the recommended value. 8k is a reasonable default if it's not documented, though some models have no max output limit—in that case 'Max Output Tokens' should be set to the same value as 'Max Tokens'.")

		maxOutputTokensStr, err := term.GetRequiredUserStringInputWithDefault("Max Output Tokens:", "8192")
		if err != nil {
			term.OutputErrorAndExit("Error reading reserved output tokens: %v", err)
			return
		}
		maxOutputTokens, err := strconv.Atoi(maxOutputTokensStr)
		if err != nil {
			term.OutputErrorAndExit("Invalid number for reserved output tokens: %v", err)
			return
		}
		model.MaxOutputTokens = maxOutputTokens

		fmt.Println("'Reserved Output Tokens' is the default number of tokens reserved for model output. This ensures the model has enough tokens to generate a response. It can be lower than the 'Max Output Tokens' limit and should be set to what a *realistic* output could reach under normal circumstances. If the 'Max Output Tokens' limit is fairly low, just use that. If the 'Max Output Tokens' is very high, or is equal to the 'Max Tokens' input limit, set a lower value so that there's enough room for input. For reasoning models, make sure enough space is included for reasoning tokens.")

		var defaultReservedOutputTokens int
		if maxOutputTokens <= int(float64(maxTokens)*0.2) {
			defaultReservedOutputTokens = maxOutputTokens
		}

		var reservedOutputTokensStr string
		if defaultReservedOutputTokens == 0 {
			reservedOutputTokensStr, err = term.GetRequiredUserStringInputWithDefault("Reserved Output Tokens:", "8192")
		} else {
			reservedOutputTokensStr, err = term.GetRequiredUserStringInputWithDefault("Reserved Output Tokens:", strconv.Itoa(defaultReservedOutputTokens))
		}

		if err != nil {
			term.OutputErrorAndExit("Error reading reserved output tokens: %v", err)
			return
		}

		reservedOutputTokens, err := strconv.Atoi(reservedOutputTokensStr)
		if err != nil {
			term.OutputErrorAndExit("Invalid number for reserved output tokens: %v", err)
			return
		}
		model.ReservedOutputTokens = reservedOutputTokens

		fmt.Println("'Preferred Output Format' is the format for roles needing structured output. Currently, OpenAI models do best with 'Tool Call JSON' and other models generally do better with 'XML'. Choose 'XML' if you're unsure as it offers the widest compatibility. 'Tool Call JSON' requires tool call support and reliable JSON generation.")

		outputFormatLabels := map[string]string{
			string(shared.ModelOutputFormatXml):          "XML",
			string(shared.ModelOutputFormatToolCallJson): "Tool Call JSON",
		}

		res, err := term.SelectFromList("Preferred Output Format:", []string{
			outputFormatLabels[string(shared.ModelOutputFormatXml)],
			outputFormatLabels[string(shared.ModelOutputFormatToolCallJson)],
		})
		if err != nil {
			term.OutputErrorAndExit("Error selecting output format: %v", err)
			return
		}
		for key, label := range outputFormatLabels {
			if label == res {
				model.PreferredOutputFormat = shared.ModelOutputFormat(key)
				break
			}
		}

		model.HasImageSupport, err = term.ConfirmYesNo("Is multi-modal image support enabled?")
		if err != nil {
			term.OutputErrorAndExit("Error confirming image support: %v", err)
			return
		}
	}

	term.StartSpinner("")
	apiErr := api.Client.CreateCustomModel(model)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error adding model: %v", apiErr.Msg)
		return
	}

	fmt.Println("✅ Added custom model", color.New(color.Bold, term.ColorHiCyan).Sprint(string(model.ModelId)))
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

	renderSettings(settings, allProperties)

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

	renderSettings(settings, allProperties)

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
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(false)
		table.SetHeader([]string{"Model", "Input", "Output", "Reserved"})
		for _, model := range shared.BuiltInBaseModels {
			table.Append([]string{string(model.ModelId), fmt.Sprintf("%d 🪙", model.MaxTokens), fmt.Sprintf("%d 🪙", model.MaxOutputTokens), fmt.Sprintf("%d 🪙", model.ReservedOutputTokens)})
		}
		table.Render()
		fmt.Println()
	}

	if len(customModels) > 0 {
		color.New(color.Bold, term.ColorHiCyan).Println("🛠️  Custom Models")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(false)
		table.SetHeader([]string{"#", "ID", "🪙"})
		for i, model := range customModels {
			table.Append([]string{fmt.Sprintf("%d", i+1), string(model.ModelId), strconv.Itoa(model.MaxTokens)})
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

	var modelToDelete *shared.CustomModel

	if len(args) == 1 {
		input := args[0]
		// Try to parse input as index
		index, err := strconv.Atoi(input)
		if err == nil && index > 0 && index <= len(models) {
			modelToDelete = models[index-1]
		} else {
			// Search by name
			for _, m := range models {
				if string(m.ModelId) == input {
					modelToDelete = m
					break
				}
			}
		}
	}

	if modelToDelete == nil {
		opts := make([]string, len(models))
		for i, model := range models {
			opts[i] = string(model.ModelId)
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
	apiErr = api.Client.DeleteCustomModel(modelToDelete.Id)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error deleting custom model: %v", apiErr)
		return
	}

	fmt.Printf("✅ Deleted custom model %s\n", color.New(color.Bold, term.ColorHiCyan).Sprint(string(modelToDelete.ModelId)))

	fmt.Println()
	term.PrintCmds("", "models available", "models add")
}

func renderSettings(settings *shared.PlanSettings, allProperties bool) {
	modelPack := settings.ModelPack

	color.New(color.Bold, term.ColorHiCyan).Println("🎛️  Current Model Pack")
	renderModelPack(modelPack, allProperties)

	if allProperties {
		color.New(color.Bold, term.ColorHiCyan).Println("🧠 Planner Defaults")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(false)
		table.SetHeader([]string{"Max Tokens", "Max Convo Tokens"})
		table.Append([]string{
			fmt.Sprintf("%d", modelPack.Planner.GetFinalLargeContextFallback().GetSharedBaseConfig().MaxTokens),
			fmt.Sprintf("%d", modelPack.Planner.GetMaxConvoTokens()),
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
		table.Render()
		fmt.Println()
	}
}

func renderModelPack(modelPack *shared.ModelPack, allProperties bool) {
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
	cols := []string{
		"Role",
		"Model",
	}
	align := []int{
		tablewriter.ALIGN_LEFT, // Role
		tablewriter.ALIGN_LEFT, // Model
	}
	if allProperties {
		cols = append(cols, []string{
			"Temperature",
			"Top P",
			"Max Input",
		}...)
		align = append(align, []int{
			tablewriter.ALIGN_RIGHT, // Temperature
			tablewriter.ALIGN_RIGHT, // Top P
			tablewriter.ALIGN_RIGHT, // Max Input
		}...)
	}
	table.SetHeader(cols)
	table.SetColumnAlignment(align)

	anyRoleParamsDisabled := false

	var addModelRow func(role string, config shared.ModelRoleConfig, indent int)
	addModelRow = func(role string, config shared.ModelRoleConfig, indent int) {
		if indent > 0 {
			role = "└─ " + role
			for i := 0; i < indent-1; i++ {
				role = " " + role
			}
		}

		var temp float32
		var topP float32
		var disabled bool

		if config.GetSharedBaseConfig().RoleParamsDisabled {
			temp = 1
			topP = 1
			disabled = true
			anyRoleParamsDisabled = true
		} else {
			temp = config.Temperature
			topP = config.TopP
		}

		tempStr := fmt.Sprintf("%.1f", temp)
		if disabled {
			tempStr = "*" + tempStr
		}

		topPStr := fmt.Sprintf("%.1f", topP)
		if disabled {
			topPStr = "*" + topPStr
		}

		row := []string{
			role,
			string(config.GetModelId()),
		}

		if allProperties {
			row = append(row, []string{
				tempStr,
				topPStr,
				fmt.Sprintf("%d 🪙", config.GetSharedBaseConfig().MaxTokens-config.GetReservedOutputTokens()),
			}...)
		}
		table.Append(row)

		// Add large context and large output fallback(s) if present
		if config.LargeContextFallback != nil {
			addModelRow("large-context", *config.LargeContextFallback, indent+1)
		}

		if config.LargeOutputFallback != nil {
			addModelRow("large-output", *config.LargeOutputFallback, indent+1)
		}

		if config.StrongModel != nil {
			addModelRow("strong", *config.StrongModel, indent+1)
		}

		if config.ErrorFallback != nil {
			addModelRow("error", *config.ErrorFallback, indent+1)
		}
	}

	addModelRow(string(shared.ModelRolePlanner), modelPack.Planner.ModelRoleConfig, 0)

	addModelRow(string(shared.ModelRoleArchitect), modelPack.GetArchitect(), 0)
	addModelRow(string(shared.ModelRoleCoder), modelPack.GetCoder(), 0)
	addModelRow(string(shared.ModelRolePlanSummary), modelPack.PlanSummary, 0)
	addModelRow(string(shared.ModelRoleBuilder), modelPack.Builder, 0)
	addModelRow(string(shared.ModelRoleWholeFileBuilder), modelPack.GetWholeFileBuilder(), 0)
	addModelRow(string(shared.ModelRoleName), modelPack.Namer, 0)
	addModelRow(string(shared.ModelRoleCommitMsg), modelPack.CommitMsg, 0)
	addModelRow(string(shared.ModelRoleExecStatus), modelPack.ExecStatus, 0)
	table.Render()

	if anyRoleParamsDisabled && allProperties {
		fmt.Println("* these models do not support changing temperature or top p")
	}

	fmt.Println()

}
