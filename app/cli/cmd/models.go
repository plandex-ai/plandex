package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"

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

	settings, err := api.Client.GetSettings(lib.CurrentPlanId, lib.CurrentBranch)

	if err != nil {
		fmt.Println("Error getting settings:", err)
		return
	}

	modelSet := settings.ModelSet
	if modelSet == nil {
		modelSet = &shared.DefaultModelSet
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Role", "Provider", "Model", "Max 🪙", "Temperature", "Top P", "Other"})

	addModelRow := func(role string, config shared.ModelRoleConfig, additionalConfig string) {
		table.Append([]string{
			role,
			string(config.BaseModelConfig.Provider),
			config.BaseModelConfig.ModelName,
			fmt.Sprintf("%d", config.BaseModelConfig.MaxTokens),
			fmt.Sprintf("%.1f", config.Temperature),
			fmt.Sprintf("%.1f", config.TopP),
			additionalConfig,
		})
	}

	addModelRow(string(shared.ModelRolePlannerRole), modelSet.Planner.ModelRoleConfig, fmt.Sprintf("max-convo-tokens → %d", modelSet.Planner.PlannerModelConfig.MaxConvoTokens))
	addModelRow(string(shared.ModelRolePlanSummaryRole), modelSet.PlanSummary, "")
	// Builder role
	builderAdditionalConfig := ""
	addModelRow(string(shared.ModelRoleBuilderRole), modelSet.Builder.ModelRoleConfig, builderAdditionalConfig)

	// Namer role
	namerAdditionalConfig := ""
	addModelRow(string(shared.ModelRoleNameRole), modelSet.Namer.ModelRoleConfig, namerAdditionalConfig)

	// CommitMsg role
	commitMsgAdditionalConfig := ""
	addModelRow(string(shared.ModelRoleCommitMsgRole), modelSet.CommitMsg.ModelRoleConfig, commitMsgAdditionalConfig)

	// ExecStatus role
	execStatusAdditionalConfig := ""
	addModelRow(string(shared.ModelRoleExecStatusRole), modelSet.ExecStatus.ModelRoleConfig, execStatusAdditionalConfig)

	table.Render()
}
