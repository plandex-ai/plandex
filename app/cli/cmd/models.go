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
}

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Show model settings",
	Run:   models,
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
		modelSet = &shared.DefaultModelSet
	}

	color.New(color.Bold, term.ColorHiCyan).Println("🤖 Models")
	table := tablewriter.NewWriter(os.Stdout)
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
	addModelRow(string(shared.ModelRoleBuilder), modelSet.Builder.ModelRoleConfig)
	addModelRow(string(shared.ModelRoleName), modelSet.Namer.ModelRoleConfig)
	addModelRow(string(shared.ModelRoleCommitMsg), modelSet.CommitMsg.ModelRoleConfig)
	addModelRow(string(shared.ModelRoleExecStatus), modelSet.ExecStatus.ModelRoleConfig)
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

}
