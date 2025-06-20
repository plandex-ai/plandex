package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/fs"
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
var saveCustomModels bool
var customModelsPath string

func init() {
	RootCmd.AddCommand(modelsCmd)

	modelsCmd.Flags().BoolVarP(&allProperties, "all", "a", false, "Show all properties")

	modelsCmd.AddCommand(listAvailableModelsCmd)
	modelsCmd.AddCommand(addCustomModelCmd)
	modelsCmd.AddCommand(updateCustomModelCmd)
	modelsCmd.AddCommand(manageCustomModelsCmd)
	modelsCmd.AddCommand(deleteCustomModelCmd)
	modelsCmd.AddCommand(defaultModelsCmd)

	manageCustomModelsCmd.Flags().BoolVar(&saveCustomModels, "save", false, "Save custom models")
	manageCustomModelsCmd.Flags().StringVarP(&customModelsPath, "file", "f", "", "Path to custom models file")

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

var manageCustomModelsCmd = &cobra.Command{
	Use:   "custom",
	Short: "Manage custom models, providers, and model packs",
	Run:   manageCustomModels,
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
	Run:     customModelsNotImplemented,
}

func manageCustomModels(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")

	var serverModelsInput *shared.ModelsInput
	var defaultConfig *shared.PlanConfig

	errCh := make(chan error, 2)

	go func() {
		var err error
		serverModelsInput, err = lib.GetServerModelsInput()
		if err != nil {
			errCh <- fmt.Errorf("error getting server models input: %v", err)
			return
		}
		errCh <- nil
	}()

	go func() {
		var apiErr *shared.ApiError
		defaultConfig, apiErr = api.Client.GetDefaultPlanConfig()
		if apiErr != nil {
			errCh <- fmt.Errorf("error getting default config: %v", apiErr.Msg)
			return
		}
		errCh <- nil
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			term.OutputErrorAndExit(err.Error())
			return
		}
	}

	usingDefaultPath := false
	if customModelsPath == "" {
		usingDefaultPath = true
		customModelsPath = lib.CustomModelsDefaultPath
	}

	exists, err := fs.FileExists(customModelsPath)
	if err != nil {
		term.OutputErrorAndExit("Error checking custom models file: %v", err)
		return
	}

	if saveCustomModels {
		if !exists {
			term.OutputErrorAndExit("File not found: %s", customModelsPath)
			return
		}
	} else {
		if serverModelsInput.IsEmpty() {
			jsonData, err := json.MarshalIndent(getExampleTemplate(auth.Current.IsCloud, auth.Current.IntegratedModelsMode), "", "  ")
			if err != nil {
				term.OutputErrorAndExit("Error marshalling template: %v", err)
				return
			}

			err = os.MkdirAll(filepath.Dir(customModelsPath), 0755)
			if err != nil {
				term.OutputErrorAndExit("Error creating directory: %v", err)
				return
			}

			err = os.WriteFile(customModelsPath, jsonData, 0644)
			if err != nil {
				term.OutputErrorAndExit("Error writing template file: %v", err)
				return
			}

			term.StopSpinner()

			fmt.Printf("🧠 Example models file → %s\n", customModelsPath)
			fmt.Println("👨‍💻 Edit it, then come back here to save")
			fmt.Println()
		} else {
			serverClientModelsInput := serverModelsInput.ToClientModelsInput()
			serverClientModelsInput.PrepareUpdate()

			var localModelsInput shared.ModelsInput

			if exists {
				// we only do a conflict check on the default path in the home dir
				// if user specifies the path and the file exists, just open the file without checking for conflicts
				if usingDefaultPath {
					res, err := lib.CustomModelsCheckLocalChanges(customModelsPath)
					if err != nil {
						term.OutputErrorAndExit("Error checking local changes: %v", err)
						return
					}

					localModelsInput = res.LocalModelsInput
					if res.HasLocalChanges {
						term.StopSpinner()

						res, err := warnModelsFileLocalChanges(customModelsPath, "models custom")
						if err != nil {
							term.OutputErrorAndExit("Error confirming: %v", err)
							return
						}
						if !res {
							return
						}
						fmt.Println()
						term.StartSpinner("")
					}
				}
			}

			if !serverModelsInput.Equals(localModelsInput) {
				err := lib.WriteCustomModelsFile(customModelsPath, serverModelsInput, true)
				if err != nil {
					term.OutputErrorAndExit("Error saving custom models file: %v", err)
					return
				}
			}

			term.StopSpinner()

			fmt.Printf("🧠 %s → %s\n", color.New(color.Bold, term.ColorHiCyan).Sprint("Models file"), customModelsPath)
			fmt.Println("👨‍💻 Edit it, then come back here to save")
			fmt.Println()
		}

		pathArg := ""
		if !usingDefaultPath {
			pathArg = fmt.Sprintf(" --file %s", customModelsPath)
		}

		res := maybePromptAndOpenModelsFile(customModelsPath, pathArg, "models custom", defaultConfig, nil)
		if res.shouldReturn {
			return
		}
	}

	didUpdate := lib.MustSyncCustomModels(customModelsPath, serverModelsInput, saveCustomModels)

	if !didUpdate {
		fmt.Println("🤷‍♂️ No changes to custom models/providers/model packs")
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

	term.PrintCmds("", "set-model", "models available", "models default", "models custom")
}

func defaultModels(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	settings, err := api.Client.GetOrgDefaultSettings()

	if err != nil {
		term.OutputErrorAndExit("Error getting default model settings: %v", err)
		return
	}

	term.StopSpinner()

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

	if customModelsOnly && auth.Current.IntegratedModelsMode {
		term.OutputErrorAndExit("Custom models are not supported in Integrated Models mode on Plandex Cloud")
		return
	}

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
			term.PrintCmds("", "models", "set-model", "models custom")
		} else {
			term.PrintCmds("", "models custom")
		}
	} else {
		term.PrintCmds("", "models available --custom", "models", "set-model", "models custom")
	}
}

func renderSettings(settings *shared.PlanSettings, allProperties bool) {
	modelPack := settings.GetModelPack()

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
	fmt.Println("Use " + color.New(color.BgCyan, color.FgHiWhite).Sprint(" plandex models custom ") + " to manage custom models, providers, and model packs")
	os.Exit(1)
}

func getExampleTemplate(isCloud, isCloudIntegratedModels bool) shared.ClientModelsInput {
	exampleProviderName := "togetherai"

	var customProviders []*shared.CustomProvider
	usesProviders := []shared.BaseModelUsesProvider{}
	if !isCloud {
		customProviders = append(customProviders, &shared.CustomProvider{
			Name:         exampleProviderName,
			BaseUrl:      "https://api.together.xyz/v1",
			ApiKeyEnvVar: "TOGETHER_API_KEY",
		})

		usesProviders = append(usesProviders, shared.BaseModelUsesProvider{
			Provider:       shared.ModelProviderCustom,
			CustomProvider: &exampleProviderName,
			ModelName:      "meta-llama/Llama-4-Maverick-17B-128E-Instruct-FP8",
		})
	}
	usesProviders = append(usesProviders, shared.BaseModelUsesProvider{
		Provider:  shared.ModelProviderOpenRouter,
		ModelName: "meta-llama/llama-4-maverick",
	})

	var customModels []*shared.CustomModel
	if !isCloudIntegratedModels {
		customModels = []*shared.CustomModel{
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

				Providers: usesProviders,
			},
		}
	}

	return shared.ClientModelsInput{
		SchemaUrl:       shared.SchemaUrlInputConfig,
		CustomProviders: customProviders,
		CustomModels:    customModels,
		CustomModelPacks: []*shared.ClientModelPackSchema{
			{
				Name:        "example-model-pack",
				Description: "Example model pack",
				ClientModelPackSchemaRoles: shared.ClientModelPackSchemaRoles{
					Planner:          "deepseek/r1",
					Architect:        "deepseek/r1",
					Coder:            "deepseek/v3-0324",
					PlanSummary:      "meta-llama/llama-4-maverick",
					Builder:          "deepseek/r1-hidden",
					WholeFileBuilder: "deepseek/r1-hidden",
					ExecStatus:       "deepseek/r1-hidden",
					Namer:            "meta-llama/llama-4-maverick",
					CommitMsg:        "meta-llama/llama-4-maverick",
				},
			},
		},
	}
}
