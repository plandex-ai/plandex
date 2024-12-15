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
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(settingsCmd)
	settingsCmd.AddCommand(defaultSettingsCmd)
}

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Show plan settings",
	Run:   settings,
}

var defaultSettingsCmd = &cobra.Command{
	Use:   "default",
	Short: "Show default settings for new plans",
	Run:   defaultSettings,
}

func settings(cmd *cobra.Command, args []string) {
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

	config, err := api.Client.GetPlanConfig(lib.CurrentPlanId)
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error getting settings: %v", err)
		return
	}

	title := fmt.Sprintf("%s Settings", color.New(color.Bold, term.ColorHiGreen).Sprint(plan.Name))

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.Append([]string{title})
	table.Render()
	fmt.Println()

	renderConfig(config)

	term.PrintCmds("", "set", "settings default", "models")
}

func defaultSettings(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	config, err := api.Client.GetDefaultPlanConfig()
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error getting default settings: %v", err)
		return
	}

	title := fmt.Sprintf("%s Settings", color.New(color.Bold, term.ColorHiGreen).Sprint("Default"))
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.Append([]string{title})
	table.Render()
	fmt.Println()

	renderConfig(config)

	term.PrintCmds("", "set default", "settings", "models")
}

func renderConfig(config *shared.PlanConfig) {
	color.New(color.Bold, term.ColorHiCyan).Println("⚙️  Settings")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Name", "Value"})

	table.Append([]string{"Auto Apply", fmt.Sprintf("%t", config.AutoApply)})
	table.Append([]string{"Auto Commit", fmt.Sprintf("%t", config.AutoCommit)})
	table.Append([]string{"Auto Context", fmt.Sprintf("%t", config.AutoContext)})
	table.Append([]string{"No Exec", fmt.Sprintf("%t", config.NoExec)})
	table.Append([]string{"Auto Debug", fmt.Sprintf("%t", config.AutoDebug)})
	table.Append([]string{"Auto Debug Tries", fmt.Sprintf("%d", config.AutoDebugTries)})

	table.Render()
	fmt.Println()
}
