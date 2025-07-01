package cmd

import (
	"fmt"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/term"
	"slices"
	"strconv"
	"strings"

	shared "plandex-shared"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(setConfigCmd)
	setConfigCmd.AddCommand(defaultSetConfigCmd)
	RootCmd.AddCommand(setAutoCmd)
	setAutoCmd.AddCommand(setAutoDefaultCmd)
}

var setConfigCmd = &cobra.Command{
	Use:   "set-config [setting] [value]",
	Short: "Update current plan config",
	Run:   setConfig,
	Args:  cobra.MaximumNArgs(2),
}

var defaultSetConfigCmd = &cobra.Command{
	Use:   "default [setting] [value]",
	Short: "Update default plan config",
	Run:   defaultSetConfig,
	Args:  cobra.MaximumNArgs(2),
}

var setAutoCmd = &cobra.Command{
	Use:   "set-auto [value]",
	Short: "Update config auto-mode",
	Run:   setAuto,
	Args:  cobra.MaximumNArgs(1),
}

var setAutoDefaultCmd = &cobra.Command{
	Use:   "default [value]",
	Short: "Update default config auto-mode",
	Run:   setAutoDefault,
	Args:  cobra.MaximumNArgs(1),
}

func setAuto(cmd *cobra.Command, args []string) {
	setConfig(cmd, append([]string{"auto-mode"}, args...))
}

func setAutoDefault(cmd *cobra.Command, args []string) {
	defaultSetConfig(cmd, append([]string{"auto-mode"}, args...))
}

func setConfig(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	term.StartSpinner("")
	config, apiErr := api.Client.GetPlanConfig(lib.CurrentPlanId)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting current config: %v", apiErr)
		return
	}

	if config == nil {
		config = &shared.PlanConfig{}
	}

	key, updatedConfig := updateConfig(args, config)
	if updatedConfig == nil {
		return
	}

	term.StartSpinner("")
	apiErr = api.Client.UpdatePlanConfig(lib.CurrentPlanId, shared.UpdatePlanConfigRequest{
		Config: updatedConfig,
	})
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error updating config: %v", apiErr)
		return
	}

	fmt.Println("✅ Config updated")
	lib.ShowPlanConfig(updatedConfig, key)
	fmt.Println()

	loadMapIfNeeded(config, updatedConfig)
	removeMapIfNeeded(config, updatedConfig)

	if !(config.AutoApply && config.AutoExec) && updatedConfig.AutoApply && updatedConfig.AutoExec {
		color.New(term.ColorHiYellow, color.Bold).Println("⚠️  You enabled automatic apply and execution.")

		fmt.Println()
	} else if !config.AutoApply && updatedConfig.AutoApply {
		color.New(term.ColorHiYellow, color.Bold).Println("⚠️  You enabled automatic apply.")
		fmt.Println()
	} else if !config.AutoExec && updatedConfig.AutoExec {
		color.New(term.ColorHiYellow, color.Bold).Println("⚠️  You enabled automatic execution.")
		fmt.Println()
	}

	term.StopSpinner()

	term.PrintCmds("", "config", "config default", "set-config default")
}

func defaultSetConfig(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	config, apiErr := api.Client.GetDefaultPlanConfig()
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting current config: %v", apiErr)
		return
	}

	if config == nil {
		config = &shared.PlanConfig{}
	}

	key, updatedConfig := updateConfig(args, config)
	if updatedConfig == nil {
		return
	}

	term.StartSpinner("")
	apiErr = api.Client.UpdateDefaultPlanConfig(shared.UpdateDefaultPlanConfigRequest{
		Config: updatedConfig,
	})
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error updating config: %v", apiErr)
		return
	}

	fmt.Println("✅ Default config updated")
	lib.ShowPlanConfig(updatedConfig, key)
	fmt.Println()
	term.PrintCmds("", "config default", "config", "set-config")
}

type sortableSetting struct {
	sortKey string
	cfg     shared.ConfigSetting
}

func updateConfig(args []string, originalConfig *shared.PlanConfig) (string, *shared.PlanConfig) {
	var setting, value string

	if len(args) > 0 {
		setting = strings.ToLower(strings.ReplaceAll(args[0], "-", ""))
	}

	if len(args) > 1 {
		value = args[1]
	}

	if setting == "" {
		var sorted []sortableSetting

		for key, cfg := range shared.ConfigSettingsByKey {
			var sortKey string
			if cfg.SortKey != "" {
				sortKey = cfg.SortKey
			} else {
				sortKey = key
			}
			sorted = append(sorted, sortableSetting{sortKey, cfg})
		}

		slices.SortFunc(sorted, func(a, b sortableSetting) int {
			return strings.Compare(a.sortKey, b.sortKey)
		})

		var opts []string
		for _, opt := range sorted {
			opts = append(opts, fmt.Sprintf("%s → %s", opt.cfg.Name, opt.cfg.Desc))
		}

		selection, err := term.SelectFromList("Choose a setting to update:", opts)
		if err != nil {
			if err.Error() == "interrupt" {
				return "", nil
			}
			term.OutputErrorAndExit("Error selecting setting: %v", err)
			return "", nil
		}

		setting = strings.Split(selection, " →")[0]
		setting = strings.ToLower(strings.ReplaceAll(setting, "-", ""))
	}

	config := *originalConfig
	cfgSetting, exists := shared.ConfigSettingsByKey[setting]
	if !exists {
		term.OutputErrorAndExit("Unknown setting: %s\n", setting)
		return "", nil
	}

	if value == "" {
		if cfgSetting.BoolSetter != nil {
			options := []string{"Enabled", "Disabled"}
			selection, err := term.SelectFromList(fmt.Sprintf("Set %s:", cfgSetting.Name), options)
			if err != nil {
				if err.Error() == "interrupt" {
					return "", nil
				}
				term.OutputErrorAndExit("Error selecting value: %v", err)
				return "", nil
			}
			cfgSetting.BoolSetter(&config, selection == "Enabled")
		} else if cfgSetting.IntSetter != nil {
			value, err := term.GetRequiredUserStringInput(fmt.Sprintf("Set %s (number)", cfgSetting.Name))
			if err != nil {
				if err.Error() == "interrupt" {
					return "", nil
				}
				term.OutputErrorAndExit("Error getting value: %v", err)
				return "", nil
			}
			n, err := strconv.Atoi(value)
			if err != nil {
				term.OutputErrorAndExit("Invalid number value for %s (%s)", cfgSetting.Name, value)
				return "", nil
			}
			cfgSetting.IntSetter(&config, n)
		} else if cfgSetting.StringSetter != nil {
			var selection string
			var err error
			choices := *cfgSetting.Choices
			if len(choices) > 0 {
				if cfgSetting.HasCustomChoice {
					choices = append(choices, "Other")
				}
				selection, err = term.SelectFromList(fmt.Sprintf("Set %s:", cfgSetting.Name), choices)
				if err != nil {
					if err.Error() == "interrupt" {
						return "", nil
					}
					term.OutputErrorAndExit("Error selecting value: %v", err)
					return "", nil
				}
				if selection == "Other" {
					selection, err = term.GetRequiredUserStringInput(fmt.Sprintf("Enter value for %s", cfgSetting.Name))
					if err != nil {
						if err.Error() == "interrupt" {
							return "", nil
						}
						term.OutputErrorAndExit("Error getting value: %v", err)
						return "", nil
					}
				} else if cfgSetting.ChoiceToKey != nil {
					selection = cfgSetting.ChoiceToKey(selection)
				}
			} else {
				selection, err = term.GetRequiredUserStringInput(fmt.Sprintf("Set %s", cfgSetting.Name))
				if err != nil {
					if err.Error() == "interrupt" {
						return "", nil
					}
					term.OutputErrorAndExit("Error getting value: %v", err)
					return "", nil
				}
			}
			cfgSetting.StringSetter(&config, selection)
		} else if cfgSetting.EditorSetter != nil {
			editor := lib.SelectEditor(false)
			cfgSetting.EditorSetter(&config, editor.Name, editor.Cmd, editor.Args)
		}
	} else {
		if cfgSetting.BoolSetter != nil {
			b, err := parseBooleanArg(value)
			if err != nil {
				term.OutputErrorAndExit("Invalid value for %s (%s)", cfgSetting.Name, value)
				return "", nil
			}
			cfgSetting.BoolSetter(&config, b)
		} else if cfgSetting.IntSetter != nil {
			n, err := strconv.Atoi(value)
			if err != nil {
				term.OutputErrorAndExit("Invalid number value for %s (%s)", cfgSetting.Name, value)
				return "", nil
			}
			cfgSetting.IntSetter(&config, n)
		} else if cfgSetting.StringSetter != nil {
			cfgSetting.StringSetter(&config, value)
		} else if cfgSetting.EditorSetter != nil {
			fields := strings.Fields(value)
			cmd := fields[0]
			var cmdArgs []string
			if len(fields) > 1 {
				cmdArgs = fields[1:]
			}
			cfgSetting.EditorSetter(&config, value, cmd, cmdArgs)
		}
	}

	return setting, &config
}

func parseBooleanArg(value string) (bool, error) {
	switch value {
	case "enabled", "true", "t", "yes", "y", "1":
		return true, nil
	case "disabled", "false", "f", "no", "n", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid value: %s", value)
	}

}

func loadMapIfNeeded(originalConfig, updatedConfig *shared.PlanConfig) {
	if updatedConfig.AutoLoadContext && !originalConfig.AutoLoadContext {
		hasMap := false

		term.StartSpinner("")
		context, err := api.Client.ListContext(lib.CurrentPlanId, lib.CurrentBranch)

		if err == nil {
			for _, c := range context {
				if c.ContextType == shared.ContextMapType {
					hasMap = true
					break
				}
			}

			if !hasMap {
				lib.MustLoadAutoContextMap()
				fmt.Println()
			}
		}
	}
}

func removeMapIfNeeded(originalConfig, updatedConfig *shared.PlanConfig) {
	if originalConfig.AutoLoadContext && !updatedConfig.AutoLoadContext {
		term.StartSpinner("")
		context, err := api.Client.ListContext(lib.CurrentPlanId, lib.CurrentBranch)

		if err == nil {
			for _, c := range context {
				if c.ContextType == shared.ContextMapType && (c.AutoLoaded || c.FilePath == ".") {
					res, err := api.Client.DeleteContext(lib.CurrentPlanId, lib.CurrentBranch, shared.DeleteContextRequest{
						Ids: map[string]bool{c.Id: true},
					})
					term.StopSpinner()
					if err != nil {
						term.OutputErrorAndExit("Error deleting context: %v", err)
						return
					}
					fmt.Println("✅ " + res.Msg)
					break
				}
			}
		}

	}
}
