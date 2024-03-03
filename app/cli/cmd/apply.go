package cmd

import (
	"fmt"
	"plandex/auth"
	"plandex/lib"

	"github.com/spf13/cobra"
)

var autoConfirm bool

func init() {
	applyCmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "Automatically confirm unless plan is outdated")

	RootCmd.AddCommand(applyCmd)
}

var applyCmd = &cobra.Command{
	Use:     "apply [name-or-index]",
	Aliases: []string{"ap"},
	Short:   "Apply a plan to the project",
	Run:     apply,
}

func apply(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		fmt.Println("🤷‍♂️ No current plan")
		return
	}

	lib.MustApplyPlan(lib.CurrentPlanId, lib.CurrentBranch, autoConfirm)
}
