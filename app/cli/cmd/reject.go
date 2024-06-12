package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"
	"sort"

	"github.com/spf13/cobra"
)

var rejectAll bool

func init() {
	RootCmd.AddCommand(rejectCmd)

	rejectCmd.Flags().BoolVarP(&rejectAll, "all", "a", false, "Reject all pending changes")
}

var rejectCmd = &cobra.Command{
	Use:     "reject [files...]",
	Aliases: []string{"rj"},
	Short:   "Reject pending changes",
	Run:     reject,
}

func reject(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	term.StartSpinner("")

	currentPlanState, apiErr := api.Client.GetCurrentPlanState(lib.CurrentPlanId, lib.CurrentBranch)

	if apiErr != nil {
		term.StopSpinner()
		term.OutputErrorAndExit("Error getting current plan state: %v", apiErr)
	}

	currentFiles := currentPlanState.CurrentPlanFiles.Files

	if len(currentFiles) == 0 {
		term.StopSpinner()
		term.OutputErrorAndExit("No pending changes to reject")
	}

	if rejectAll || len(args) == 0 {
		numToReject := len(currentFiles)
		suffix := ""
		if numToReject > 1 {
			suffix = "s"
		}

		if !rejectAll {
			term.StopSpinner()

			// output list of pending files
			fmt.Println("Files with pending changes:")

			pathsToSort := make([]string, 0, len(currentFiles))

			for path := range currentFiles {
				pathsToSort = append(pathsToSort, path)
			}

			sort.Strings(pathsToSort)

			for _, path := range pathsToSort {
				fmt.Println(" • ", path)
			}
			fmt.Println()

			shouldContinue, err := term.ConfirmYesNo("Reject changes to %d file%s?", numToReject, suffix)

			if err != nil {
				term.OutputErrorAndExit("failed to get confirmation user input: %s", err)
			}

			if !shouldContinue {
				os.Exit(0)
			}
			term.ResumeSpinner()
		}

		apiErr := api.Client.RejectAllChanges(lib.CurrentPlanId, lib.CurrentBranch)

		if apiErr != nil {
			term.StopSpinner()
			term.OutputErrorAndExit("Error rejecting all changes: %v", apiErr)
		}

		term.StopSpinner()
		fmt.Printf("✅ Rejected changes to %d file%s\n", numToReject, suffix)

		return
	}

	if len(args) > 0 {
		for _, path := range args {
			if _, ok := currentFiles[path]; !ok {
				term.StopSpinner()
				term.OutputErrorAndExit("File %s not found in plan or has no changes to reject", path)
			}
		}
	}

	numToReject := len(args)
	suffix := ""
	if numToReject > 1 {
		suffix = "s"
	}

	apiErr = api.Client.RejectFiles(lib.CurrentPlanId, lib.CurrentBranch, args)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error rejecting changes: %v", apiErr)
	}

	fmt.Printf("✅ Rejected changes to %d file%s\n", numToReject, suffix)
}
