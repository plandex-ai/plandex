package cmd

import (
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/plan_exec"
	"plandex-cli/types"

	shared "plandex-shared"

	"github.com/spf13/cobra"
)

var (
	chatOnly bool
)

var continueCmd = &cobra.Command{
	Use:     "continue",
	Aliases: []string{"c"},
	Short:   "Continue the plan",
	Run:     doContinue,
}

func init() {
	RootCmd.AddCommand(continueCmd)

	continueCmd.Flags().BoolVar(&chatOnly, "chat", false, "Continue in chat mode (no file changes)")

	initExecFlags(continueCmd, initExecFlagsParams{
		omitFile:   true,
		omitEditor: true,
	})
}

func doContinue(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()
	mustSetPlanExecFlags(cmd, false)

	tellFlags := types.TellFlags{
		TellBg:          tellBg,
		TellStop:        tellStop,
		TellNoBuild:     tellNoBuild,
		IsUserContinue:  true,
		ExecEnabled:     !noExec,
		AutoContext:     tellAutoContext,
		SmartContext:    tellSmartContext,
		AutoApply:       tellAutoApply,
		IsChatOnly:      chatOnly,
		SkipChangesMenu: tellSkipMenu,
	}

	plan_exec.TellPlan(plan_exec.ExecParams{
		CurrentPlanId: lib.CurrentPlanId,
		CurrentBranch: lib.CurrentBranch,
		AuthVars:      lib.MustVerifyAuthVars(auth.Current.IntegratedModelsMode),
		CheckOutdatedContext: func(maybeContexts []*shared.Context, projectPaths *types.ProjectPaths) (bool, bool, error) {
			auto := autoConfirm || tellAutoApply || tellAutoContext

			return lib.CheckOutdatedContextWithOutput(auto, auto, maybeContexts, projectPaths)
		},
	}, "", tellFlags)

	if tellAutoApply {
		applyFlags := types.ApplyFlags{
			AutoConfirm: true,
			AutoCommit:  autoCommit,
			NoCommit:    !autoCommit,
			AutoExec:    autoExec,
			NoExec:      noExec,
			AutoDebug:   autoDebug,
		}

		lib.MustApplyPlan(lib.ApplyPlanParams{
			PlanId:     lib.CurrentPlanId,
			Branch:     lib.CurrentBranch,
			ApplyFlags: applyFlags,
			TellFlags:  tellFlags,
			OnExecFail: plan_exec.GetOnApplyExecFail(applyFlags, tellFlags),
		})
	}
}
