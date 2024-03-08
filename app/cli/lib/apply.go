package lib

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plandex/api"
	"plandex/fs"
	"plandex/term"
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
	var stashed bool
	var errMsg string
	var errArgs []interface{}
	var unformattedErrMsg string

	if len(toApply) == 0 {
		fmt.Println("🤷‍♂️ No changes to apply")
	} else {
		if !autoConfirm {
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
				os.Exit(0)
			}
		}

		defer func() {
			if aborted {
				// clear any partially applied changes before popping the stash
				err := GitClearUncommittedChanges()
				if err != nil {
					log.Printf("Failed to clear uncommitted changes: %v", err)
				}
			}

			if stashed {
				err := GitStashPop(true)
				if err != nil {
					log.Printf("Failed to pop git stash: %v", err)
				}
			}

			if errMsg != "" {
				if unformattedErrMsg == "" {
					term.OutputErrorAndExit(errMsg, errArgs...)
				} else {
					term.OutputSimpleError(errMsg, errArgs...)
					term.OutputUnformattedErrorAndExit(unformattedErrMsg)
				}
			}
		}()

		if isRepo && hasUncommittedChanges {
			// If there are uncommitted changes, first checkout any files that will be applied, then stash the changes

			// Checkout the files that will be applied
			// It's safe to do this with the confidence that no work will be lost because we just ensured the plan is using the latest state of all these files
			for path := range toApply {
				exists := true
				_, err := os.Stat(filepath.Join(fs.ProjectRoot, path))
				if err != nil {
					if os.IsNotExist(err) {
						exists = false
					} else {
						errMsg = "Error checking for file %s:"
						errArgs = append(errArgs, path)
						unformattedErrMsg = err.Error()
						aborted = true
						return
					}
				}

				if exists {
					hasChanges, err := GitFileHasUncommittedChanges(path)

					if err != nil {
						errMsg = "Error checking for uncommitted changes for file %s:"
						errArgs = append(errArgs, path)
						unformattedErrMsg = err.Error()
						aborted = true
						return
					}

					// log.Printf("File %s has uncommitted changes: %v", path, hasChanges)

					if hasChanges {
						err := os.Remove(filepath.Join(fs.ProjectRoot, path))
						if err != nil {
							errMsg = "Failed to remove file prior to update %s:"
							errArgs = append(errArgs, path)
							unformattedErrMsg = err.Error()
							aborted = true
							return
						}

						GitCheckoutFile(path) // ignore error to cover untracked files
					}
				}
			}

			err := GitStashCreate("Plandex auto-stash")
			if err != nil {
				errMsg = "Failed to create git stash:"
				unformattedErrMsg = err.Error()
				aborted = true
				return
			}
			stashed = true
		}

		for path, content := range toApply {
			// Compute destination path
			dstPath := filepath.Join(fs.ProjectRoot, path)
			// Create the directory if it doesn't exist
			err := os.MkdirAll(filepath.Dir(dstPath), 0755)
			if err != nil {
				aborted = true
				errMsg = "failed to create directory %s:"
				errArgs = append(errArgs, filepath.Dir(dstPath))
				return
			}

			// Write the file
			err = os.WriteFile(dstPath, []byte(content), 0644)
			if err != nil {
				aborted = true
				errMsg = "failed to write %s:"
				errArgs = append(errArgs, dstPath)
				return
			}
		}

		term.StartSpinner("")
		apiErr := api.Client.ApplyPlan(planId, branch)
		term.StopSpinner()

		if apiErr != nil {
			aborted = true
			errMsg = "failed to set pending results applied: %s"
			errArgs = append(errArgs, apiErr.Msg)
			return
		}

		if isRepo {
			// Commit the changes
			msg := currentPlanState.PendingChangesSummaryForApply()

			// log.Println("Committing changes with message:")
			// log.Println(msg)

			// spew.Dump(currentPlanState)

			err := GitAddAndCommit(fs.ProjectRoot, msg, true)
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
