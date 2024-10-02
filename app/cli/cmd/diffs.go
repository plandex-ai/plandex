package cmd

import (
	"fmt"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"

	"github.com/spf13/cobra"
)

// var plainTextOutput bool

var diffsCmd = &cobra.Command{
	Use:     "diff",
	Aliases: []string{"diffs"},
	Short:   "Show diffs for the pending changes in git diff format",
	Run:     diffs,
}

func init() {
	RootCmd.AddCommand(diffsCmd)

	diffsCmd.Flags().BoolVarP(&plainTextOutput, "plain", "p", false, "Output diffs in plain text with no ANSI codes")

}

func diffs(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MaybeResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	diffs, err := api.Client.GetPlanDiffs(lib.CurrentPlanId, lib.CurrentBranch, plainTextOutput)

	if err != nil {
		term.OutputErrorAndExit("Error getting plan diffs: %v", err)
		return
	}

	if plainTextOutput {
		fmt.Println(diffs)
	} else {
		term.PageOutput(diffs)
	}
}
