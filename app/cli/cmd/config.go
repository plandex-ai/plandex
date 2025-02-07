package cmd

import (
	"fmt"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/term"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(configCmd)
	configCmd.AddCommand(defaultConfigCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show plan config",
	Run:   config,
}

var defaultConfigCmd = &cobra.Command{
	Use:   "default",
	Short: "Show default config for new plans",
	Run:   defaultConfig,
}

func config(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	term.StartSpinner("")

	config, apiErr := api.Client.GetPlanConfig(lib.CurrentPlanId)
	if apiErr != nil {
		term.StopSpinner()
		term.OutputErrorAndExit("Error getting config: %v", apiErr.Msg)
		return
	}

	term.StopSpinner()

	color.New(color.Bold, term.ColorHiCyan).Println("⚙️  Plan Config")
	lib.ShowPlanConfig(config, "")
	fmt.Println()

	term.PrintCmds("", "set-config", "config default", "set-config default")
}

func defaultConfig(cmd *cobra.Command, args []string) {
	auth.MustResolveAuth(false)

	term.StartSpinner("")
	config, err := api.Client.GetDefaultPlanConfig()
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error getting default config: %v", err)
		return
	}

	color.New(color.Bold, term.ColorHiCyan).Println("⚙️  Default Config")
	lib.ShowPlanConfig(config, "")
	fmt.Println()

	term.PrintCmds("", "set-config default", "config", "set-config")
}
