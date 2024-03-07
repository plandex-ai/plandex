package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"plandex/api"
	"plandex/fs"
	"plandex/term"

	"github.com/fatih/color"
)

func MustApplyPlan(planId, branch string, autoConfirm bool) {
	term.StartSpinner("")

	currentPlanState, apiErr := api.Client.GetCurrentPlanState(planId, branch)

	if apiErr != nil {
		term.StopSpinner()
		term.OutputErrorAndExit("Error getting current plan state: %v", apiErr)
	}

	if currentPlanState.HasPendingBuilds() {
		plansRunningRes, apiErr := api.Client.ListPlansRunning([]string{CurrentProjectId}, false)

		if apiErr != nil {
			term.StopSpinner()
			term.OutputErrorAndExit("Error getting running plans: %v", apiErr)
		}

		for _, b := range plansRunningRes.Branches {
			if b.PlanId == planId && b.Name == branch {
				fmt.Println("This plan is currently active. Please wait for it to finish before applying.")
				fmt.Println()
				term.PrintCmds("", "ps", "connect")
				return
			}
		}

		term.StopSpinner()

		fmt.Println("This plan has changes that need to be built before applying")
		fmt.Println()

		shouldBuild, err := term.ConfirmYesNo("Build changes now?")

		if err != nil {
			term.OutputErrorAndExit("failed to get confirmation user input: %s", err)
		}

		if !shouldBuild {
			fmt.Println("Apply plan canceled")
			os.Exit(0)
		}

		_, err = buildPlanInlineFn(nil)

		if err != nil {
			term.OutputErrorAndExit("failed to build plan: %v", err)
		}
	}

	anyOutdated, didUpdate, _ := MustCheckOutdatedContext(false, true, nil)

	if anyOutdated && !didUpdate {
		term.StopSpinner()
		fmt.Println("Apply plan canceled")
		os.Exit(0)
	}

	term.StopSpinner()

	currentPlanFiles := currentPlanState.CurrentPlanFiles

	if len(currentPlanFiles.Files) == 0 {
		term.StopSpinner()
		fmt.Println("🤷‍♂️ No changes to apply")
		return
	}

	isRepo := fs.ProjectRootIsGitRepo()

	hasUncommittedChanges := false
	if isRepo {
		// Check if there are any uncommitted changes
		var err error
		hasUncommittedChanges, err = CheckUncommittedChanges()

		if err != nil {
			term.OutputSimpleError("Error checking for uncommitted changes:")
			term.OutputUnformattedErrorAndExit(err.Error())
		}
	}

	toApply := currentPlanFiles.Files

	var aborted bool

	if len(toApply) == 0 {
		fmt.Println("🤷‍♂️ No changes to apply")
	} else {
		if !autoConfirm {
			fmt.Println()
			numToApply := len(toApply)
			suffix := ""
			if numToApply > 1 {
				suffix = "s"
			}
			shouldContinue, err := term.ConfirmYesNo("Apply changes to %d file%s?", numToApply, suffix)

			if err != nil {
				term.OutputErrorAndExit("failed to get confirmation user input: %s", err)
			}

			if !shouldContinue {
				aborted = true
				return
			}
		}

		if isRepo && hasUncommittedChanges {
			// If there are uncommitted changes, first checkout any files that will be applied, then stash the changes

			// Checkout the files that will be applied
			// It's safe to do this with the confidence that no work will be lost because we just ensured the plan is using the latest state of all these files
			for path := range toApply {
				err := GitCheckoutFile(path)
				if err != nil {
					term.OutputSimpleError("Failed to reset file %s: %v", path, err)
					term.OutputUnformattedErrorAndExit(err.Error())
				}
			}

			err := GitStashCreate("Plandex auto-stash")
			if err != nil {
				term.OutputSimpleError("Failed to create git stash:")
				term.OutputUnformattedErrorAndExit(err.Error())
			}

			defer func() {
				if aborted {
					// clear any partially applied changes before popping the stash
					err := GitClearUncommittedChanges()
					if err != nil {
						term.OutputSimpleError("Failed to clear uncommitted changes:")
						term.OutputUnformattedErrorAndExit(err.Error())
					}
				}

				err := GitStashPop(true)
				if err != nil {
					term.OutputSimpleError("Failed to pop git stash:")
					term.OutputUnformattedErrorAndExit(err.Error())
				}
			}()
		}

		for path, content := range toApply {
			// Compute destination path
			dstPath := filepath.Join(fs.ProjectRoot, path)
			// Create the directory if it doesn't exist
			err := os.MkdirAll(filepath.Dir(dstPath), 0755)
			if err != nil {
				aborted = true
				term.OutputErrorAndExit("failed to create directory %s: %v", filepath.Dir(dstPath), err)
			}

			// Write the file
			err = os.WriteFile(dstPath, []byte(content), 0644)
			if err != nil {
				aborted = true
				term.OutputErrorAndExit("failed to write %s: %v", dstPath, err)
			}
		}

		term.StartSpinner("")
		apiErr := api.Client.ApplyPlan(planId, branch)
		term.StopSpinner()

		if apiErr != nil {
			aborted = true
			term.OutputErrorAndExit("failed to set pending results applied: %s", apiErr.Msg)
		}

		if isRepo {
			// Commit the changes
			err := GitAddAndCommit(fs.ProjectRoot, color.New(color.BgBlue, color.FgHiWhite, color.Bold).Sprintln(" 🤖 Plandex ")+currentPlanState.PendingChangesSummary(), true)
			if err != nil {
				aborted = true
				// return fmt.Errorf("failed to commit changes: %w", err)
				term.OutputSimpleError("Failed to commit changes:")
				term.OutputUnformattedErrorAndExit(err.Error())
			}
		}

		fmt.Println("✅ Applied changes")
	}

}
