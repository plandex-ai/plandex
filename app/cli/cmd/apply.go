package cmd

import (
	"plandex/auth"
	"plandex/lib"
	"plandex/term"

	"github.com/spf13/cobra"
)

var autoCommit, noCommit bool

func init() {
	applyCmd.Flags().BoolVarP(&autoCommit, "commit", "c", false, "Commit changes to git")
	applyCmd.Flags().BoolVarP(&noCommit, "no-commit", "n", false, "Do not commit changes to git")
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

	lib.MustApplyPlan(lib.CurrentPlanId, lib.CurrentBranch, true, autoCommit, noCommit)
}
