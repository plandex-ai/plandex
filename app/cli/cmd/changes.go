package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/changes_tui"
	"plandex/lib"
	"plandex/plan_exec"
	"plandex/term"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(changesCmd)
}

var changesCmd = &cobra.Command{
	Use:     "changes",
	Aliases: []string{"ch"},
	Short:   "View, copy, or manage changes for the current plan",
	Run:     changes,
}

func changes(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	term.StartSpinner("🔬 Checking plan state...")

	currentPlanState, apiErr := api.Client.GetCurrentPlanState(lib.CurrentPlanId, lib.CurrentBranch)

	if apiErr != nil {
		term.StopSpinner()
		fmt.Fprintf(os.Stderr, "Error getting current plan state: %s\n", apiErr.Msg)
		return
	}

	if currentPlanState.HasPendingBuilds() {
		plansRunningRes, apiErr := api.Client.ListPlansRunning([]string{lib.CurrentProjectId}, false)

		if apiErr != nil {
			// return fmt.Errorf("error getting running plans: %s", apiErr.Msg)
			term.StopSpinner()
			fmt.Fprintf(os.Stderr, "Error getting running plans: %s\n", apiErr.Msg)
		}

		viewIncomplete := false

		for _, b := range plansRunningRes.Branches {
			if b.PlanId == lib.CurrentPlanId && b.Name == lib.CurrentBranch {
				fmt.Println("This plan is currently active.")

				res, err := term.ConfirmYesNo("View potentially incomplete changes anyway?")

				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting confirmation user input: %s\n", err)
					return
				}

				if res {
					viewIncomplete = true
					break
				} else {
					fmt.Println()
					term.PrintCmds("", "ps", "connect")
					return
				}
			}
		}

		term.StopSpinner()

		if !viewIncomplete {
			fmt.Println("This plan has unbuilt changes. Building now.")

			didBuild, err := plan_exec.Build(plan_exec.ExecParams{
				CurrentPlanId: lib.CurrentPlanId,
				CurrentBranch: lib.CurrentBranch,
				CheckOutdatedContext: func(cancelOpt bool, maybeContexts []*shared.Context) (bool, bool, bool) {
					return lib.MustCheckOutdatedContext(cancelOpt, true, maybeContexts)
				},
			}, false)

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error building plan: %v\n", err)
				return
			}

			if !didBuild {
				fmt.Println("Build canceled")
				fmt.Println()
				term.PrintCmds("", "build", "log", "rewind")
				return
			}
		}
	}

	term.StopSpinner()

	err := changes_tui.StartChangesUI()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting changes UI: %v\n", err)
	}

}
