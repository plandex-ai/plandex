package cmd

import (
	"plandex/auth"
	"plandex/lib"
	"plandex/plan_exec"
	"plandex/term"
	"strconv"

	"github.com/spf13/cobra"
)

var autoCommit, noCommit, autoExec bool

func init() {
	applyCmd.Flags().BoolVarP(&autoCommit, "commit", "c", false, "Commit changes to git")
	applyCmd.Flags().BoolVar(&noCommit, "no-commit", false, "Do not commit changes to git")
	applyCmd.Flags().BoolVar(&noExec, "no-exec", false, "Disable _apply.sh execution")
	applyCmd.Flags().BoolVar(&autoExec, "auto-exec", false, "Automatically execute commands without confirmation")
	applyCmd.Flags().Var(newAutoDebugValue(&autoDebug), "debug", "Automatically execute and debug failing commands (optionally specify number of tries—default is 5)")
	applyCmd.Flag("debug").NoOptDefVal = strconv.Itoa(defaultAutoDebugTries)
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

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	flags := lib.ApplyFlags{
		AutoConfirm: true,
		AutoCommit:  autoCommit,
		NoCommit:    noCommit,
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
