package cmd

import (
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/plan_exec"

	shared "plandex-shared"

	"github.com/spf13/cobra"
)

var continueCmd = &cobra.Command{
	Use:     "continue",
	Aliases: []string{"c"},
	Short:   "Continue the plan",
	Run:     doContinue,
}

func init() {
	RootCmd.AddCommand(continueCmd)

	initExecFlags(continueCmd, initExecFlagsParams{
		omitFile:   true,
		omitEditor: true,
	})
}

func doContinue(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()
	mustSetPlanExecFlags(cmd)

	var apiKeys map[string]string
	if !auth.Current.IntegratedModelsMode {
		apiKeys = lib.MustVerifyApiKeys()
	}

	plan_exec.TellPlan(plan_exec.ExecParams{
		CurrentPlanId: lib.CurrentPlanId,
		CurrentBranch: lib.CurrentBranch,
		ApiKeys:       apiKeys,
		CheckOutdatedContext: func(maybeContexts []*shared.Context) (bool, bool, error) {
			auto := autoConfirm || tellAutoApply || tellAutoContext

			return lib.CheckOutdatedContextWithOutput(auto, auto, maybeContexts)
		},
	}, "", plan_exec.TellFlags{
		TellBg:         tellBg,
		TellStop:       tellStop,
		TellNoBuild:    tellNoBuild,
		IsUserContinue: true,
		ExecEnabled:    !noExec,
		AutoContext:    tellAutoContext,
	})

	if tellAutoApply {
		flags := lib.ApplyFlags{
			AutoConfirm: true,
			AutoCommit:  autoCommit,
			NoCommit:    !autoCommit,
			AutoExec:    autoExec,
			NoExec:      noExec,
			AutoDebug:   autoDebug,
		}

		lib.MustApplyPlan(
			lib.CurrentPlanId,
			lib.CurrentBranch,
			flags,
			plan_exec.GetOnApplyExecFail(flags),
		)
	}
}
