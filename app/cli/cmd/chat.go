package cmd

import (
	"fmt"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/plan_exec"
	"plandex-cli/types"

	shared "plandex-shared"

	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:     "chat [prompt]",
	Aliases: []string{"c"},
	Short:   "Chat without making changes",
	// Long:  ``,
	Args: cobra.RangeArgs(0, 1),
	Run:  doChat,
}

func init() {
	RootCmd.AddCommand(chatCmd)

	initExecFlags(chatCmd, initExecFlagsParams{
		omitNoBuild:      true,
		omitStop:         true,
		omitBg:           true,
		omitApply:        true,
		omitExec:         true,
		omitSmartContext: true,
		omitSkipMenu:     true,
	})

}

func doChat(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()
	mustSetPlanExecFlags(cmd, false)

	var apiKeys map[string]string
	if !auth.Current.IntegratedModelsMode {
		apiKeys = lib.MustVerifyAuthVars()
	}

	prompt := getTellPrompt(args)

	if prompt == "" {
		fmt.Println("🤷‍♂️ No prompt to send")
		return
	}

	plan_exec.TellPlan(plan_exec.ExecParams{
		CurrentPlanId: lib.CurrentPlanId,
		CurrentBranch: lib.CurrentBranch,
		AuthVars:      apiKeys,
		CheckOutdatedContext: func(maybeContexts []*shared.Context, projectPaths *types.ProjectPaths) (bool, bool, error) {
			auto := autoConfirm || tellAutoApply || tellAutoContext
			return lib.CheckOutdatedContextWithOutput(auto, auto, maybeContexts, projectPaths)
		},
	}, prompt, types.TellFlags{
		IsChatOnly:      true,
		AutoContext:     tellAutoContext,
		SkipChangesMenu: tellSkipMenu,
	})
}
