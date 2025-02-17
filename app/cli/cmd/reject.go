package cmd

import (
	"fmt"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/term"
	"sort"

	"github.com/plandex-ai/survey/v2"
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

	if rejectAll {
		numToReject := len(currentFiles)
		suffix := ""
		if numToReject > 1 {
			suffix = "s"
		}

		apiErr := api.Client.RejectAllChanges(lib.CurrentPlanId, lib.CurrentBranch)

		if apiErr != nil {
			term.StopSpinner()
			term.OutputErrorAndExit("Error rejecting all changes: %v", apiErr)
		}

		term.StopSpinner()
		fmt.Printf("✅ Rejected changes to %d file%s\n", numToReject, suffix)

		sortedFiles := make([]string, 0, len(currentFiles))
		for file := range currentFiles {
			sortedFiles = append(sortedFiles, file)
		}
		sort.Strings(sortedFiles)

		for _, file := range sortedFiles {
			fmt.Printf("• 📄 %s\n", file)
		}

		return
	}

	if len(args) > 0 {
		for _, path := range args {
			if _, ok := currentFiles[path]; !ok {
				term.StopSpinner()
				term.OutputErrorAndExit("File %s not found in plan or has no changes to reject", path)
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

		sortedFiles := append([]string{}, args...)
		sort.Strings(sortedFiles)

		for _, file := range sortedFiles {
			fmt.Printf("• 📄 %s\n", file)
		}

		return
	}

	// No args provided - use survey multiselect
	term.StopSpinner()

	pathsToSort := make([]string, 0, len(currentFiles))
	for path := range currentFiles {
		pathsToSort = append(pathsToSort, path)
	}
	sort.Strings(pathsToSort)

	var selectedFiles []string
	prompt := &survey.MultiSelect{
		Message: "Select files to reject:",
		Options: pathsToSort,
	}

	err := survey.AskOne(prompt, &selectedFiles)
	if err != nil {
		term.OutputErrorAndExit("Error getting file selection: %v", err)
	}

	if len(selectedFiles) == 0 {
		fmt.Println("No files selected")
		return
	}

	term.StartSpinner("")
	apiErr = api.Client.RejectFiles(lib.CurrentPlanId, lib.CurrentBranch, selectedFiles)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error rejecting changes: %v", apiErr)
	}

	suffix := ""
	if len(selectedFiles) > 1 {
		suffix = "s"
	}
	fmt.Printf("✅ Rejected changes to %d file%s\n", len(selectedFiles), suffix)

	sortedFiles := append([]string{}, selectedFiles...)
	sort.Strings(sortedFiles)

	for _, file := range sortedFiles {
		fmt.Printf("• 📄 %s\n", file)
	}
}
