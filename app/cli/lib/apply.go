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

func ApplyPlan(planId, branch string, autoConfirm bool) error {
	term.StartSpinner("🔬 Checking plan state...")

	currentPlanState, apiErr := api.Client.GetCurrentPlanState(planId, branch)

	if apiErr != nil {
		term.StopSpinner()
		return fmt.Errorf("error getting current plan state: %s", apiErr.Msg)
	}

	if currentPlanState.HasPendingBuilds() {
		plansRunningRes, apiErr := api.Client.ListPlansRunning([]string{CurrentProjectId}, false)

		if apiErr != nil {
			term.StopSpinner()
			return fmt.Errorf("error getting running plans: %s", apiErr.Msg)
		}

		for _, b := range plansRunningRes.Branches {
			if b.PlanId == planId && b.Name == branch {
				fmt.Println("This plan is currently active. Please wait for it to finish before applying.")
				fmt.Println()
				term.PrintCmds("", "ps", "connect")
				return nil
			}
		}

		term.StopSpinner()

		fmt.Println("This plan has changes that need to be built before applying")
		fmt.Println()

		shouldBuild, err := term.ConfirmYesNo("Build changes now?")

		if err != nil {
			return fmt.Errorf("failed to get confirmation user input: %s", err)
		}

		if !shouldBuild {
			fmt.Println("Apply plan canceled")
			os.Exit(0)
		}

		_, err = buildPlanInlineFn(nil)

		if err != nil {
			return fmt.Errorf("failed to build plan: %w", err)
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
		return nil
	}

	isRepo := fs.ProjectRootIsGitRepo()

	hasUncommittedChanges := false
	if isRepo {
		// Check if there are any uncommitted changes
		var err error
		hasUncommittedChanges, err = CheckUncommittedChanges()

		if err != nil {
			return fmt.Errorf("error checking for uncommitted changes: %w", err)
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
				return fmt.Errorf("failed to get confirmation user input: %s", err)
			}

			if !shouldContinue {
				aborted = true
				return nil
			}
		}

		if isRepo && hasUncommittedChanges {
			// If there are uncommitted changes, stash them
			err := GitStashCreate("Plandex auto-stash")
			if err != nil {
				return fmt.Errorf("failed to create git stash: %w", err)
			}

			defer func() {
				if aborted {
					// clear any partially applied changes before popping the stash
					err := GitClearUncommittedChanges()
					if err != nil {
						term.OutputErrorAndExit("Failed to clear uncommitted changes: %v", err)
					}
				}

				err := GitStashPop(true)
				if err != nil {
					term.OutputErrorAndExit("Failed to pop git stash: %v", err)
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
				return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(dstPath), err)
			}

			// Write the file
			err = os.WriteFile(dstPath, []byte(content), 0644)
			if err != nil {
				aborted = true
				return fmt.Errorf("failed to write %s: %w", dstPath, err)
			}
		}

		apiErr := api.Client.ApplyPlan(planId, branch)

		if apiErr != nil {
			aborted = true
			return fmt.Errorf("failed to set pending results applied: %s", apiErr.Msg)
		}

		if isRepo {
			// Commit the changes
			err := GitAddAndCommit(fs.ProjectRoot, color.New(color.BgBlue, color.FgHiWhite, color.Bold).Sprintln(" 🤖 Plandex ")+currentPlanState.PendingChangesSummary(), true)
			if err != nil {
				aborted = true
				return fmt.Errorf("failed to commit changes: %w", err)
			}
		}

		fmt.Println("✅ Applied changes")
	}

	return nil

}
