package cmd

import (
	"fmt"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/term"

	shared "plandex-shared"

	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:     "current",
	Aliases: []string{"cu"},
	Short:   "Get the current plan",
	Run:     current,
}

func init() {
	RootCmd.AddCommand(currentCmd)
}

func current(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MaybeResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	term.StartSpinner("")
	plan, err := api.Client.GetPlan(lib.CurrentPlanId)
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error getting plan: %v", err)
		return
	}

	currentBranchesByPlanId, err := api.Client.GetCurrentBranchByPlanId(lib.CurrentProjectId, shared.GetCurrentBranchByPlanIdRequest{
		CurrentBranchByPlanId: map[string]string{
			lib.CurrentPlanId: lib.CurrentBranch,
		},
	})

	if err != nil {
		term.OutputErrorAndExit("Error getting current branches: %v", err)
	}

	table := lib.GetCurrentPlanTable(plan, currentBranchesByPlanId, nil)
	fmt.Println(table)

	term.PrintCmds("", "tell", "ls", "plans")

}
