package cmd

import (
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/plan_exec"
	"plandex-cli/term"

	"github.com/spf13/cobra"
)

var autoCommit, skipCommit, autoExec bool

func init() {
	initApplyFlags(applyCmd)
	initExecScriptFlags(applyCmd)
	RootCmd.AddCommand(applyCmd)
}

var applyCmd = &cobra.Command{
	Use:     "apply",
	Aliases: []string{"ap"},
	Short:   "Apply a plan to the project",
	Run:     apply,
}

func apply(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()
	mustSetPlanExecFlags(cmd)

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	flags := lib.ApplyFlags{
		AutoConfirm: true,
		AutoCommit:  autoCommit,
		NoCommit:    skipCommit,
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
