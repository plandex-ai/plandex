package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/schema"
	"plandex-cli/term"
	"strconv"

	shared "plandex-shared"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var customModelsOnly bool

var allProperties bool
var genTemplatePath string
var updateModels bool

func init() {
	RootCmd.AddCommand(modelsCmd)

	modelsCmd.Flags().BoolVarP(&allProperties, "all", "a", false, "Show all properties")

	modelsCmd.AddCommand(listAvailableModelsCmd)
	modelsCmd.AddCommand(addCustomModelCmd)
	modelsCmd.AddCommand(updateCustomModelCmd)
	modelsCmd.AddCommand(importCustomModelCmd)
	modelsCmd.AddCommand(deleteCustomModelCmd)
	modelsCmd.AddCommand(defaultModelsCmd)

	importCustomModelCmd.Flags().StringVarP(&genTemplatePath, "generate", "g", "", "Generate a template JSON file")
	importCustomModelCmd.Flags().BoolVarP(&updateModels, "update", "u", false, "Update existing models")

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

var importCustomModelCmd = &cobra.Command{
	Use:   "import [json-file]",
	Short: "Create or update custom models, providers, and model packs from a JSON file",
	Run:   importCustomModels,
	Args:  cobra.MaximumNArgs(1),
}

var addCustomModelCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"create"},
	Short:   "Add a custom model",
	Run:     customModelsNotImplemented,
}

var updateCustomModelCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"edit"},
	Short:   "Update a custom model",
	Run:     customModelsNotImplemented,
}

var deleteCustomModelCmd = &cobra.Command{
	Use:     "rm",
	Aliases: []string{"remove", "delete"},
	Short:   "Remove a custom model",
	Args:    cobra.MaximumNArgs(1),
	Run:     deleteCustomModel,
}

func importCustomModels(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	var jsonFile string

	if len(args) == 1 {
		jsonFile = args[0]
	} else if updateModels {
		term.StartSpinner("")

		errCh := make(chan *shared.ApiError, 3)
		var modelsInput *shared.ModelsInput

		go func() {
			models, apiErr := api.Client.ListCustomModels()
			if apiErr != nil {
				errCh <- apiErr
			}
			modelsInput.CustomModels = models
			errCh <- nil
		}()

		go func() {
			providers, apiErr := api.Client.ListCustomProviders()
			if apiErr != nil {
				errCh <- apiErr
			}
			modelsInput.CustomProviders = providers
			errCh <- nil
		}()

		go func() {
			modelPacks, apiErr := api.Client.ListModelPacks()
			if apiErr != nil {
				errCh <- apiErr
			}

			schemas := make([]*shared.ModelPackSchema, len(modelPacks))
			for i, modelPack := range modelPacks {
				schemas[i] = modelPack.ToModelPackSchema()
			}

			modelsInput.CustomModelPacks = schemas
			errCh <- nil
		}()

		term.StopSpinner()

		for i := 0; i < 3; i++ {
			err := <-errCh
			if err != nil {
				term.OutputErrorAndExit("Error fetching custom models: %v", err.Msg)
				return
			}
		}

		// Generate temp file with current state
		tmpFile, err := os.CreateTemp("", "plandex-models-update-*.json")
		if err != nil {
			term.OutputErrorAndExit("Error creating temp file: %v", err)
			return
		}

		jsonFile = tmpFile.Name()
		tmpFile.Close()

		jsonData, err := json.MarshalIndent(modelsInput, "", "  ")
		if err != nil {
			term.OutputErrorAndExit("Error marshalling models: %v", err)
			return
		}

		err = os.WriteFile(jsonFile, jsonData, 0644)
		if err != nil {
			term.OutputErrorAndExit("Error writing file: %v", err)
			return
		}

		fmt.Printf("✅ Created file with current configuration: %s\n",
			color.New(color.Bold, term.ColorHiGreen).Sprint(jsonFile))
		fmt.Println()

	} else {
		// Case 3: Create new models (default flow)
		fmt.Println(color.New(color.Bold, color.FgHiCyan).Sprint("Creating template for custom models..."))
		fmt.Println()

		// Generate temp file with examples
		tmpFile, err := os.CreateTemp("", "plandex-models-*.json")
		if err != nil {
			term.OutputErrorAndExit("Error creating template file: %v", err)
			return
		}

		jsonFile = tmpFile.Name()
		tmpFile.Close()

		jsonData, err := json.MarshalIndent(getExampleTemplate(), "", "  ")
		if err != nil {
			term.OutputErrorAndExit("Error marshalling template: %v", err)
			return
		}

		err = os.WriteFile(jsonFile, jsonData, 0644)
		if err != nil {
			term.OutputErrorAndExit("Error writing template file: %v", err)
			return
		}

		fmt.Printf("✅ Created template: %s\n",
			color.New(color.Bold, term.ColorHiGreen).Sprint(jsonFile))
		fmt.Println()
	}

	// For both add and update flows, open the editor
	if len(args) == 0 {
		selectedEditor := maybePromptAndOpen(jsonFile)

		if selectedEditor {
			fmt.Println("📝 Opened in editor")
			fmt.Println("👨‍💻 Edit the file, then come back here to finish importing")
			fmt.Println()

			confirmed, err := term.ConfirmYesNo("Done editing and ready to import?")
			if err != nil {
				term.OutputErrorAndExit("Error confirming: %v", err)
				return
			}

			if !confirmed {
				fmt.Println("🤷‍♂️ Import cancelled")
				fmt.Println()

				// Clean up temp file
				if looksLikeEphemeralTemplate(jsonFile) {
					os.Remove(jsonFile)
				}

				pf := "\\"
				if !term.IsRepl {
					pf = "plandex "
				}

				fmt.Println("When you're ready, run:")
				fmt.Printf("• %s\n", color.New(color.Bold, color.BgCyan, color.FgHiWhite).
					Sprintf(" %smodels import %s ", pf, jsonFile))
				return
			}
			fmt.Println()
		} else {
			// No editor available or user chose manual
			fmt.Println("👨‍💻 Edit the file in your JSON editor of choice")
			fmt.Println()

			pf := "\\"
			if !term.IsRepl {
				pf = "plandex "
			}

			fmt.Println("When you're ready, run:")
			fmt.Printf("• %s\n", color.New(color.Bold, color.BgCyan, color.FgHiWhite).
				Sprintf(" %smodels import %s ", pf, jsonFile))
			return
		}
	}

	// Import the JSON file
	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		term.OutputErrorAndExit("Error reading JSON file: %v", err)
		return
	}

	modelsInput, err := schema.ValidateModelsInputJSON(jsonData)
	if err != nil {
		color.New(color.Bold, term.ColorHiRed).Println("🚨 Error validating JSON file")
		fmt.Println(err.Error())
		return
	}

	term.StartSpinner("")
	apiErr := api.Client.CreateCustomModels(&modelsInput)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error importing models: %v", apiErr.Msg)
		return
	}

	action := "Imported"

	for _, provider := range modelsInput.CustomProviders {
		fmt.Printf("✅ %s custom %s → %s\n",
			action,
			color.New(term.ColorHiCyan).Sprint("provider"),
			color.New(color.Bold, term.ColorHiGreen).Sprint(provider.Name))
	}

	for _, model := range modelsInput.CustomModels {
		fmt.Printf("✅ %s custom %s → %s\n",
			action,
			color.New(term.ColorHiCyan).Sprint("model"),
			color.New(color.Bold, term.ColorHiGreen).Sprint(string(model.ModelId)))
	}

	for _, modelPack := range modelsInput.CustomModelPacks {
		fmt.Printf("✅ %s custom %s → %s\n",
			action,
			color.New(term.ColorHiCyan).Sprint("model pack"),
			color.New(color.Bold, term.ColorHiGreen).Sprint(modelPack.Name))
	}

	// Clean up temp file if it was generated
	if len(args) == 0 && looksLikeEphemeralTemplate(jsonFile) {
		err = os.Remove(jsonFile)
		if err != nil {
			term.OutputErrorAndExit("Error cleaning up temp file: %v", err)
		}
	}
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

func customModelsNotImplemented(cmd *cobra.Command, args []string) {
	color.New(color.Bold, color.FgHiRed).Println("⛔️ Not implemented")
	fmt.Println()
	fmt.Println("Use " + color.New(color.BgCyan, color.FgHiWhite).Sprint(" plandex models import ") + " to add or update custom models, providers, and model packs with a JSON file")
	os.Exit(1)
}

type editorCandidate struct {
	name        string
	cmd         string
	args        []string
	isJetBrains bool
}

const maxEditorOpts = 5

func detectEditors() []editorCandidate {
	guess := []editorCandidate{
		// Popular non-JetBrains launchers
		{"VS Code", "code", nil, false},
		{"Cursor", "cursor", nil, false},
		{"Zed", "zed", nil, false},
		{"Neovim", "nvim", nil, false},

		// JetBrains IDE-specific launchers
		{"IntelliJ IDEA", "idea", nil, true},
		{"GoLand", "goland", nil, true},
		{"PyCharm", "pycharm", nil, true},
		{"CLion", "clion", nil, true},
		{"WebStorm", "webstorm", nil, true},
		{"PhpStorm", "phpstorm", nil, true},
		{"DataGrip", "datagrip", nil, true},
		{"RubyMine", "rubymine", nil, true},
		{"Rider", "rider", nil, true},
		{"DataSpell", "dataspell", nil, true},

		// JetBrains universal CLI (2023.2+)
		{"JetBrains (jb)", "jb", []string{"open"}, true},

		{"Vim", "vim", nil, false},
		{"Nano", "nano", nil, false},
		{"Helix", "hx", nil, false},
		{"Micro", "micro", nil, false},
		{"Sublime Text", "subl", nil, false},
		{"TextMate", "mate", nil, false},
		{"Kakoune", "kak", nil, false},
		{"Emacs", "emacs", nil, false},
		{"Kate", "kate", nil, false},
	}
	pref := map[string]bool{}
	for _, env := range []string{"VISUAL", "EDITOR"} {
		if v := os.Getenv(env); v != "" {
			// keep only the binary name, drop path/flags
			cmd := filepath.Base(strings.Fields(v)[0])
			pref[cmd] = true
		}
	}

	_, err := exec.LookPath("jb") // true if universal launcher exists
	jbOnPath := err == nil

	var found []editorCandidate
	for _, c := range guess {
		if _, err := exec.LookPath(c.cmd); err != nil {
			continue // not on PATH
		}

		// If jb is present, drop per-IDE launchers *unless* this exact cmd
		// is marked preferred by VISUAL/EDITOR.
		if jbOnPath && c.isJetBrains && !pref[c.cmd] {
			continue
		}
		found = append(found, c)
	}

	for cmd := range pref {
		if _, err := exec.LookPath(cmd); err == nil {
			already := false
			for _, c := range found {
				if c.cmd == cmd {
					already = true
					break
				}
			}
			if !already {
				found = append(found, editorCandidate{name: cmd, cmd: cmd})
			}
		}
	}
	sort.SliceStable(found, func(i, j int) bool {
		pi, pj := pref[found[i].cmd], pref[found[j].cmd]
		if pi == pj {
			return false // keep original order
		}
		return pi // true → i comes before j
	})
	if len(found) > maxEditorOpts {
		found = found[:maxEditorOpts]
	}

	return found
}

func maybePromptAndOpen(path string) bool {
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

	choice, err := term.SelectFromList("Edit the template now?", opts)
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

func looksLikeEphemeralTemplate(p string) bool {
	// covers cases where the user pasted the temp name manually
	tmp := os.TempDir()
	return strings.HasPrefix(p, tmp+string(os.PathSeparator)) &&
		strings.HasPrefix(filepath.Base(p), "plandex-models-") &&
		strings.HasSuffix(p, ".json")
}

func getExampleTemplate() shared.ModelsInput {
	exampleProviderName := "togetherai"

	return shared.ModelsInput{
		SchemaUrl: shared.SchemaUrlInputConfig,
		CustomProviders: []*shared.CustomProvider{
			{
				Name:         exampleProviderName,
				BaseUrl:      "https://api.together.xyz/v1",
				ApiKeyEnvVar: "TOGETHER_API_KEY",
			},
		},
		CustomModels: []*shared.CustomModel{
			{
				ModelId:     shared.ModelId("meta-llama/llama-4-maverick"),
				Publisher:   shared.ModelPublisher("meta-llama"),
				Description: "Meta Llama 4 Maverick",

				BaseModelShared: shared.BaseModelShared{
					DefaultMaxConvoTokens: 75000,
					MaxTokens:             1048576,
					MaxOutputTokens:       16000,
					ReservedOutputTokens:  16000,
					ModelCompatibility:    shared.FullCompatibility,
					PreferredOutputFormat: shared.ModelOutputFormatXml,
				},

				Providers: []shared.BaseModelUsesProvider{
					{
						Provider:       shared.ModelProviderCustom,
						CustomProvider: &exampleProviderName,
						ModelName:      "meta-llama/Llama-4-Maverick-17B-128E-Instruct-FP8",
					},
					{
						Provider:  shared.ModelProviderOpenRouter,
						ModelName: "meta-llama/llama-4-maverick",
					},
				},
			},
		},
		CustomModelPacks: []*shared.ModelPackSchema{
			{
				Name:        "example-model-pack",
				Description: "Example model pack",
				Planner: shared.ModelRoleConfigSchema{
					ModelId: shared.ModelId("deepseek/r1-reasoning-visible"),
				},
				Architect: &shared.ModelRoleConfigSchema{
					ModelId: shared.ModelId("deepseek/r1-reasoning-visible"),
				},
				Coder: &shared.ModelRoleConfigSchema{
					ModelId: shared.ModelId("deepseek/v3-0324"),
				},
				PlanSummary: shared.ModelRoleConfigSchema{
					ModelId: shared.ModelId("meta-llama/llama-4-maverick"),
				},
				Builder: shared.ModelRoleConfigSchema{
					ModelId: shared.ModelId("deepseek/r1-reasoning-hidden"),
				},
				WholeFileBuilder: &shared.ModelRoleConfigSchema{
					ModelId: shared.ModelId("deepseek/r1-reasoning-hidden"),
				},
				ExecStatus: shared.ModelRoleConfigSchema{
					ModelId: shared.ModelId("deepseek/r1-reasoning-hidden"),
				},
				Namer: shared.ModelRoleConfigSchema{
					ModelId: shared.ModelId("meta-llama/llama-4-maverick"),
				},
				CommitMsg: shared.ModelRoleConfigSchema{
					ModelId: shared.ModelId("meta-llama/llama-4-maverick"),
				},
			},
		},
	}
}
