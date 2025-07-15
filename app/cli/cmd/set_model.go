package cmd

import (
	"fmt"
	"os"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/fs"
	"plandex-cli/lib"
	"plandex-cli/term"
	"strings"

	shared "plandex-shared"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var setModelUseJsonFile bool
var setModelJsonFilePath string
var setModelSave bool

func init() {
	RootCmd.AddCommand(modelsSetCmd)

	modelsSetCmd.AddCommand(defaultModelSetCmd)

	modelsSetCmd.Flags().BoolVar(&setModelUseJsonFile, "json", false, "Use a JSON file to set model settings")
	modelsSetCmd.Flags().StringVarP(&setModelJsonFilePath, "file", "f", "", "Path to model settings JSON file")
	modelsSetCmd.Flags().BoolVar(&setModelSave, "save", false, "Save model settings from JSON file")

	defaultModelSetCmd.Flags().BoolVar(&setModelUseJsonFile, "json", false, "Use a JSON file to set model settings")
	defaultModelSetCmd.Flags().StringVarP(&setModelJsonFilePath, "file", "f", "", "Path to model settings JSON file")
	defaultModelSetCmd.Flags().BoolVar(&setModelSave, "save", false, "Save model settings from JSON file")
}

var modelsSetCmd = &cobra.Command{
	Use:     "set-model [model-pack-name]",
	Aliases: []string{"set-models"},
	Short:   "Update current plan model settings",
	Run:     modelsSet,
	Args:    cobra.MaximumNArgs(1),
}

var defaultModelSetCmd = &cobra.Command{
	Use:   "default [model-pack-name]",
	Short: "Update org-wide default model settings",
	Run:   defaultModelsSet,
	Args:  cobra.MaximumNArgs(1),
}

func modelsSet(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	term.StartSpinner("")
	originalSettings, apiErr := api.Client.GetSettings(lib.CurrentPlanId, lib.CurrentBranch)

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting current settings: %v", apiErr)
		return
	}

	defaultPath := lib.GetPlanModelSettingsPath(lib.CurrentPlanId)

	settings := updateModelSettings(args, originalSettings, defaultPath)

	if settings == nil {
		return
	}

	res, apiErr := api.Client.UpdateSettings(
		lib.CurrentPlanId,
		lib.CurrentBranch,
		shared.UpdateSettingsRequest{
			ModelPackName: settings.ModelPackName,
			ModelPack:     settings.ModelPack,
		})
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error updating settings: %v", apiErr)
		return
	}

	if res == nil {
		return
	}

	fmt.Println(res.Msg)
	fmt.Println()
	term.PrintCmds("", "models", "set-model default", "log")
}

func defaultModelsSet(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	originalSettings, apiErr := api.Client.GetOrgDefaultSettings()
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting current settings: %v", apiErr)
		return
	}

	defaultPath := lib.DefaultModelSettingsPath

	settings := updateModelSettings(args, originalSettings, defaultPath)

	if settings == nil {
		return
	}

	term.StartSpinner("")
	res, apiErr := api.Client.UpdateOrgDefaultSettings(
		shared.UpdateSettingsRequest{
			ModelPackName: settings.ModelPackName,
			ModelPack:     settings.ModelPack,
		})
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error updating settings: %v", apiErr)
		return
	}

	if res == nil {
		return
	}

	fmt.Println(res.Msg)
	fmt.Println()
	term.PrintCmds("", "models", "set-model default", "log")
}

func updateModelSettings(args []string, originalSettings *shared.PlanSettings, defaultPath string) *shared.PlanSettings {
	settings, err := originalSettings.DeepCopy()
	if err != nil {
		term.OutputErrorAndExit("Error copying settings: %v", err)
		return nil
	}

	builtInModelPacks := shared.BuiltInModelPacks
	if auth.Current.IsCloud {
		filtered := []*shared.ModelPack{}
		for _, ms := range builtInModelPacks {
			if ms.LocalProvider == "" {
				filtered = append(filtered, ms)
			}
		}
		builtInModelPacks = filtered
	}

	var customModelPacks []*shared.ModelPack
	var defaultConfig *shared.PlanConfig
	var planConfig *shared.PlanConfig

	errCh := make(chan error, 3)

	go func() {
		var apiErr *shared.ApiError
		customModelPacks, apiErr = api.Client.ListModelPacks()
		if apiErr != nil {
			errCh <- fmt.Errorf("error getting custom model packs: %v", apiErr.Msg)
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

	go func() {
		if lib.CurrentPlanId != "" {
			var apiErr *shared.ApiError
			planConfig, apiErr = api.Client.GetPlanConfig(lib.CurrentPlanId)
			if apiErr != nil {
				errCh <- fmt.Errorf("error getting plan config: %v", apiErr.Msg)
				return
			}
		}
		errCh <- nil
	}()

	for i := 0; i < 3; i++ {
		err := <-errCh
		if err != nil {
			term.OutputErrorAndExit(err.Error())
			return nil
		}
	}

	useJsonFile := setModelUseJsonFile || setModelSave

	var nameArg string
	if len(args) > 0 {
		nameArg = args[0]
	}

	if !useJsonFile {
		if nameArg == "" {
			term.StopSpinner()
			const modelPackOpt = "Select a model pack"
			const jsonOpt = "Edit model settings JSON"

			selection, err := term.SelectFromList("Select a model pack or edit settings?", []string{modelPackOpt, jsonOpt})
			if err != nil {
				if err.Error() == "interrupt" {
					return nil
				}
			}

			if selection == modelPackOpt {
				useJsonFile = false
			} else {
				useJsonFile = true
				term.StartSpinner("")
			}
		}
	}

	if useJsonFile {
		usingDefaultPath := false
		if setModelJsonFilePath == "" {
			usingDefaultPath = true
			setModelJsonFilePath = defaultPath
		}

		exists, err := fs.FileExists(setModelJsonFilePath)
		if err != nil {
			term.OutputErrorAndExit("Error checking model settings file: %v", err)
			return nil
		}

		if setModelSave {
			if !exists {
				term.OutputErrorAndExit("File not found: %s", customModelsPath)
			}
		} else {
			if usingDefaultPath && exists {
				modelSettingsCheckLocalChangesResult, err := lib.ModelSettingsCheckLocalChanges(setModelJsonFilePath)
				if err != nil {
					term.OutputErrorAndExit("Error checking model settings file: %v", err)
					return nil
				}

				if modelSettingsCheckLocalChangesResult.HasLocalChanges {
					term.StopSpinner()

					res, err := warnModelsFileLocalChanges(setModelJsonFilePath, "set-model")
					if err != nil {
						term.OutputErrorAndExit("Error confirming: %v", err)
						return nil
					}
					if !res {
						return nil
					}

					fmt.Println()
					term.StartSpinner("")
				}
			}

			err = lib.WriteModelSettingsFile(setModelJsonFilePath, originalSettings)
			if err != nil {
				term.OutputErrorAndExit("Error writing model settings file: %v", err)
				return nil
			}

			term.StopSpinner()
			fmt.Printf("🧠 %s → %s\n", color.New(color.Bold, term.ColorHiCyan).Sprint("Models file"), setModelJsonFilePath)
			fmt.Println("👨‍💻 Edit it, then come back here to save")
			fmt.Println()

			pathArg := ""
			if !usingDefaultPath {
				pathArg = fmt.Sprintf(" --file %s", setModelJsonFilePath)
			}

			res := maybePromptAndOpenModelsFile(setModelJsonFilePath, pathArg, "set-model", defaultConfig, planConfig)
			if res.shouldReturn {
				return nil
			}
		}

		term.StartSpinner("")

		settings, err = lib.ApplyModelSettings(setModelJsonFilePath, originalSettings)

		if err != nil {
			term.OutputErrorAndExit("Error applying model settings: %v", err)
			return nil
		}

	} else {
		if nameArg == "" {
			var names []string
			var opts []string
			for _, ms := range builtInModelPacks {
				names = append(names, ms.Name)
				opts = append(opts, "Built-in | "+ms.Name)
			}
			for _, ms := range customModelPacks {
				names = append(names, ms.Name)
				opts = append(opts, "Custom | "+ms.Name)
			}

			term.StopSpinner()
			selection, err := term.SelectFromList("Select a model pack:", opts)
			if err != nil {
				if err.Error() == "interrupt" {
					return nil
				}
			}

			for i, opt := range opts {
				if opt == selection {
					nameArg = names[i]
					break
				}
			}
		}

		var modelPackName string
		compare := strings.ToLower(strings.TrimSpace(nameArg))
		if compare == "daily" {
			compare = "daily-driver"
		}
		if compare == "opus-4-planner" {
			compare = "opus-planner"
		}

		for _, ms := range builtInModelPacks {
			if strings.EqualFold(ms.Name, compare) {
				modelPackName = ms.Name
				break
			}
		}
		for _, ms := range customModelPacks {
			if strings.EqualFold(ms.Name, compare) {
				modelPackName = ms.Name
				break
			}
		}

		if modelPackName == "" {
			term.StopSpinner()
			term.OutputSimpleError("No model pack found with name '%s'", nameArg)
			fmt.Println()
			term.PrintCmds("", "model-packs")
			os.Exit(1)
			return nil
		}

		settings.SetModelPackByName(modelPackName)

		// clear the default settings file and hash file if they exist, ignoring errors
		os.Remove(defaultPath)
		os.Remove(defaultPath + ".hash")
	}

	term.StopSpinner()

	if originalSettings.Equals(settings) {
		fmt.Println("🤷‍♂️ No model settings were updated")
		return nil
	} else {
		return settings
	}
}
