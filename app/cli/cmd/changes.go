package cmd

import (
	"fmt"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/changes_tui"
	"plandex-cli/lib"
	"plandex-cli/plan_exec"
	"plandex-cli/term"
	"plandex-cli/types"

	shared "plandex-shared"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(changesCmd)
}

var changesCmd = &cobra.Command{
	Use:   "changes",
	Short: "View, copy, or manage changes for the current plan",
	Run:   changes,
}

func changes(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	term.StartSpinner("")

	currentPlanState, apiErr := api.Client.GetCurrentPlanState(lib.CurrentPlanId, lib.CurrentBranch)

	if apiErr != nil {
		term.StopSpinner()
		term.OutputErrorAndExit("Error getting current plan state: %s", apiErr.Msg)
	}

	// log.Println(spew.Sdump(currentPlanState))

	for currentPlanState.HasPendingBuilds() {
		plansRunningRes, apiErr := api.Client.ListPlansRunning([]string{lib.CurrentProjectId}, false)

		if apiErr != nil {
			term.StopSpinner()
			term.OutputErrorAndExit("Error getting running plans: %s", apiErr.Msg)
		}

		viewIncomplete := false

		for _, b := range plansRunningRes.Branches {
			if b.PlanId == lib.CurrentPlanId && b.Name == lib.CurrentBranch {
				fmt.Println("This plan is currently active.")

				res, err := term.ConfirmYesNo("View potentially incomplete changes anyway?")

				if err != nil {
					term.OutputErrorAndExit("Error getting confirmation user input: %v", err)
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

			var apiKeys map[string]string
			if !auth.Current.IntegratedModelsMode {
				apiKeys = lib.MustVerifyApiKeys()
			}

			didBuild, err := plan_exec.Build(plan_exec.ExecParams{
				CurrentPlanId: lib.CurrentPlanId,
				CurrentBranch: lib.CurrentBranch,
				ApiKeys:       apiKeys,
				CheckOutdatedContext: func(maybeContexts []*shared.Context) (bool, bool, error) {
					return lib.CheckOutdatedContextWithOutput(true, false, maybeContexts)
				},
			}, types.BuildFlags{})

			if err != nil {
				term.OutputErrorAndExit("Error building plan: %v\n", err)
			}

			if !didBuild {
				fmt.Println("Build canceled")
				fmt.Println()
				term.PrintCmds("", "build", "log", "rewind")
				return
			}

			term.ResumeSpinner()
			currentPlanState, apiErr = api.Client.GetCurrentPlanState(lib.CurrentPlanId, lib.CurrentBranch)

			if apiErr != nil {
				term.StopSpinner()
				term.OutputErrorAndExit("Error getting current plan state: %s", apiErr.Msg)
			}
		}
	}

	term.StopSpinner()

	err := changes_tui.StartChangesUI(currentPlanState)

	if err != nil {
		term.OutputErrorAndExit("Error starting changes UI: %v\n", err)
	}

}
