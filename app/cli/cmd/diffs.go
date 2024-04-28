package cmd

import (
	"fmt"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"

	"github.com/spf13/cobra"
)

var diffsCmd = &cobra.Command{
	Use:     "diff",
	Aliases: []string{"diffs"},
	Short:   "Show diffs for the pending changes in git diff format",
	Run:     diffs,
}

func init() {
	RootCmd.AddCommand(diffsCmd)
}

func diffs(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MaybeResolveProject()

	if lib.CurrentPlanId == "" {
		fmt.Println("🤷‍♂️ No current plan")
		return
	}

	diffs, err := api.Client.GetPlanDiffs(lib.CurrentPlanId, lib.CurrentBranch)

	if err != nil {
		term.OutputErrorAndExit("Error getting plan diffs: %v", err)
		return
	}

	term.PageOutput(diffs)
}
