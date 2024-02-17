package cmd

import (
	"fmt"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"

	"github.com/davecgh/go-spew/spew"
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
	table.SetHeader([]string{"Role", "Provider", "Model Name", "Max Tokens", "Temperature", "Top P", "Additional Config"})

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

	addModelRow("Planner", modelSet.Planner.ModelRoleConfig, fmt.Sprintf("MaxConvoTokens: %d", modelSet.Planner.PlannerModelConfig.MaxConvoTokens))
	addModelRow("PlanSummary", modelSet.PlanSummary, "")
	// Builder role
	builderAdditionalConfig := ""
	if modelSet.Builder.TaskModelConfig.OpenAIResponseFormat != nil {
		builderAdditionalConfig = fmt.Sprintf("OpenAIResponseFormat: %s", spew.Sdump(modelSet.Builder.TaskModelConfig.OpenAIResponseFormat.Type))
	}
	addModelRow("Builder", modelSet.Builder.ModelRoleConfig, builderAdditionalConfig)

	// Namer role
	namerAdditionalConfig := ""
	if modelSet.Namer.TaskModelConfig.OpenAIResponseFormat != nil {
		namerAdditionalConfig = fmt.Sprintf("OpenAIResponseFormat: %s", spew.Sdump(modelSet.Namer.TaskModelConfig.OpenAIResponseFormat.Type))
	}
	addModelRow("Namer", modelSet.Namer.ModelRoleConfig, namerAdditionalConfig)

	// CommitMsg role
	commitMsgAdditionalConfig := ""
	if modelSet.CommitMsg.TaskModelConfig.OpenAIResponseFormat != nil {
		commitMsgAdditionalConfig = fmt.Sprintf("OpenAIResponseFormat: %s", spew.Sdump(modelSet.CommitMsg.TaskModelConfig.OpenAIResponseFormat.Type))
	}
	addModelRow("CommitMsg", modelSet.CommitMsg.ModelRoleConfig, commitMsgAdditionalConfig)

	// ExecStatus role
	execStatusAdditionalConfig := ""
	if modelSet.ExecStatus.TaskModelConfig.OpenAIResponseFormat != nil {
		execStatusAdditionalConfig = fmt.Sprintf("OpenAIResponseFormat: %s", spew.Sdump(modelSet.ExecStatus.TaskModelConfig.OpenAIResponseFormat.Type))
	}
	addModelRow("ExecStatus", modelSet.ExecStatus.ModelRoleConfig, execStatusAdditionalConfig)

	table.Render()
}
