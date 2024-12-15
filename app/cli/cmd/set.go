package cmd

import (
	"fmt"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"
	"strconv"
	"strings"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(setCmd)
	setCmd.AddCommand(defaultSetCmd)
}

var setCmd = &cobra.Command{
	Use:   "set [setting] [value]",
	Short: "Update current plan config settings",
	Run:   set,
	Args:  cobra.MaximumNArgs(2),
}

var defaultSetCmd = &cobra.Command{
	Use:   "default [setting] [value]",
	Short: "Update default plan config settings",
	Run:   defaultSet,
	Args:  cobra.MaximumNArgs(2),
}

func set(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	term.StartSpinner("")
	config, err := api.Client.GetPlanConfig(lib.CurrentPlanId)
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error getting current config: %v", err)
		return
	}

	if config == nil {
		config = &shared.PlanConfig{}
	}

	updatedConfig := updateConfig(args, config)
	if updatedConfig == nil {
		return
	}

	term.StartSpinner("")
	res, err := api.Client.UpdatePlanConfig(lib.CurrentPlanId, shared.UpdatePlanConfigRequest{
		Config: updatedConfig,
	})
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error updating config: %v", err)
		return
	}

	fmt.Println(res.Msg)
	fmt.Println()
	term.PrintCmds("", "settings", "set default", "models")
}

func defaultSet(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	config, err := api.Client.GetDefaultPlanConfig()
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error getting current config: %v", err)
		return
	}

	if config == nil {
		config = &shared.PlanConfig{}
	}

	updatedConfig := updateConfig(args, config)
	if updatedConfig == nil {
		return
	}

	term.StartSpinner("")
	res, err := api.Client.UpdateDefaultPlanConfig(shared.UpdateDefaultPlanConfigRequest{
		Config: updatedConfig,
	})
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error updating config: %v", err)
		return
	}

	fmt.Println(res.Msg)
	fmt.Println()
	term.PrintCmds("", "settings", "set", "models")
}

func updateConfig(args []string, originalConfig *shared.PlanConfig) *shared.PlanConfig {
	var setting, value string

	if len(args) > 0 {
		setting = strings.ToLower(strings.ReplaceAll(args[0], "-", ""))
	}

	if len(args) > 1 {
		value = args[1]
	}

	if setting == "" {
		opts := []string{
			"auto-apply → automatically apply changes when ready",
			"auto-commit → automatically commit changes after apply",
			"auto-context → automatically update context after changes",
			"no-exec → disable command execution",
			"auto-debug → automatically debug failed commands",
			"auto-debug-tries → number of auto-debug attempts",
		}

		selection, err := term.SelectFromList("Choose a setting to update:", opts)
		if err != nil {
			if err.Error() == "interrupt" {
				return nil
			}
			term.OutputErrorAndExit("Error selecting setting: %v", err)
			return nil
		}

		setting = strings.Split(selection, " →")[0]
		setting = strings.ToLower(strings.ReplaceAll(setting, "-", ""))
	}

	config := *originalConfig

	switch setting {
	case "autoapply":
		if value == "" {
			var err error
			value, err = term.GetRequiredUserStringInput("Set auto-apply (true/false)")
			if err != nil {
				if err.Error() == "interrupt" {
					return nil
				}
				term.OutputErrorAndExit("Error getting value: %v", err)
				return nil
			}
		}
		b, err := strconv.ParseBool(value)
		if err != nil {
			fmt.Println("Invalid value for auto-apply:", value)
			return nil
		}
		config.AutoApply = b

	case "autocommit":
		if value == "" {
			var err error
			value, err = term.GetRequiredUserStringInput("Set auto-commit (true/false)")
			if err != nil {
				if err.Error() == "interrupt" {
					return nil
				}
				term.OutputErrorAndExit("Error getting value: %v", err)
				return nil
			}
		}
		b, err := strconv.ParseBool(value)
		if err != nil {
			fmt.Println("Invalid value for auto-commit:", value)
			return nil
		}
		config.AutoCommit = b

	case "autocontext":
		if value == "" {
			var err error
			value, err = term.GetRequiredUserStringInput("Set auto-context (true/false)")
			if err != nil {
				if err.Error() == "interrupt" {
					return nil
				}
				term.OutputErrorAndExit("Error getting value: %v", err)
				return nil
			}
		}
		b, err := strconv.ParseBool(value)
		if err != nil {
			fmt.Println("Invalid value for auto-context:", value)
			return nil
		}
		config.AutoContext = b

	case "noexec":
		if value == "" {
			var err error
			value, err = term.GetRequiredUserStringInput("Set no-exec (true/false)")
			if err != nil {
				if err.Error() == "interrupt" {
					return nil
				}
				term.OutputErrorAndExit("Error getting value: %v", err)
				return nil
			}
		}
		b, err := strconv.ParseBool(value)
		if err != nil {
			fmt.Println("Invalid value for no-exec:", value)
			return nil
		}
		config.NoExec = b

	case "autodebug":
		if value == "" {
			var err error
			value, err = term.GetRequiredUserStringInput("Set auto-debug (true/false)")
			if err != nil {
				if err.Error() == "interrupt" {
					return nil
				}
				term.OutputErrorAndExit("Error getting value: %v", err)
				return nil
			}
		}
		b, err := strconv.ParseBool(value)
		if err != nil {
			fmt.Println("Invalid value for auto-debug:", value)
			return nil
		}
		config.AutoDebug = b

	case "autodebugtries":
		if value == "" {
			var err error
			value, err = term.GetRequiredUserStringInput("Set auto-debug-tries (number)")
			if err != nil {
				if err.Error() == "interrupt" {
					return nil
				}
				term.OutputErrorAndExit("Error getting value: %v", err)
				return nil
			}
		}
		n, err := strconv.Atoi(value)
		if err != nil {
			fmt.Println("Invalid value for auto-debug-tries:", value)
			return nil
		}
		config.AutoDebugTries = n

	default:
		fmt.Printf("Unknown setting: %s\n", setting)
		return nil
	}

	return &config
}
