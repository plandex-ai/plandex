package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"plandex/api"
	"plandex/fs"
	"plandex/term"
	"strings"
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

	anyOutdated, didUpdate := MustCheckOutdatedContext(true, nil)

	if anyOutdated && !didUpdate {
		term.StopSpinner()
		fmt.Println("Apply plan canceled")
		os.Exit(0)
	}

	currentPlanFiles := currentPlanState.CurrentPlanFiles
	isRepo := fs.ProjectRootIsGitRepo()

	toApply := currentPlanFiles.Files

	if len(toApply) == 0 {
		term.StopSpinner()
		fmt.Println("🤷‍♂️ No changes to apply")
		return
	}

	if !autoConfirm {
		term.StopSpinner()
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
		term.ResumeSpinner()
	}

	onErr := func(errMsg string, errArgs ...interface{}) {
		term.StopSpinner()
		term.OutputErrorAndExit(errMsg, errArgs...)
	}

	onGitErr := func(errMsg, unformattedErrMsg string) {
		term.StopSpinner()
		term.OutputSimpleError(errMsg, unformattedErrMsg)
	}

	apiErr = api.Client.ApplyPlan(planId, branch)

	if apiErr != nil {
		onErr("failed to set pending results applied: %s", apiErr.Msg)
		return
	}

	var updatedFiles []string
	for path, content := range toApply {
		// Compute destination path
		dstPath := filepath.Join(fs.ProjectRoot, path)

		content = strings.ReplaceAll(content, "\\`\\`\\`", "```")

		// Check if the file exists
		var exists bool
		_, err := os.Stat(dstPath)
		if err == nil {
			exists = true
		} else {
			if os.IsNotExist(err) {
				exists = false
			} else {
				onErr("failed to check if %s exists:", dstPath)
				return
			}
		}

		if exists {
			// read file content
			bytes, err := os.ReadFile(dstPath)

			if err != nil {
				onErr("failed to read %s:", dstPath)
				return
			}

			// Check if the file has changed
			if string(bytes) == content {
				// log.Println("File is unchanged, skipping")
				continue
			} else {
				updatedFiles = append(updatedFiles, path)
			}
		} else {
			updatedFiles = append(updatedFiles, path)

			// Create the directory if it doesn't exist
			err := os.MkdirAll(filepath.Dir(dstPath), 0755)
			if err != nil {
				onErr("failed to create directory %s:", filepath.Dir(dstPath))
				return
			}
		}

		// Write the file
		err = os.WriteFile(dstPath, []byte(content), 0644)
		if err != nil {
			onErr("failed to write %s:", dstPath)
			return
		}
	}

	term.StopSpinner()

	if len(updatedFiles) == 0 {
		fmt.Println("✅ Applied changes, but no files were updated")
		return
	} else {
		if isRepo {
			fmt.Println("✏️  Plandex can commit these updates with an automatically generated message.")
			fmt.Println()
			fmt.Println("ℹ️  Only the files that Plandex is updating will be included the commit. Any other changes, staged or unstaged, will remain exactly as they are.")
			fmt.Println()

			confirmed, err := term.ConfirmYesNo("Commit Plandex updates now?")

			if err != nil {
				onErr("failed to get confirmation user input: %s", err)
			}

			if confirmed {
				// Commit the changes
				msg := currentPlanState.PendingChangesSummaryForApply()

				// log.Println("Committing changes with message:")
				// log.Println(msg)

				// spew.Dump(currentPlanState)

				err := GitAddAndCommitPaths(fs.ProjectRoot, msg, updatedFiles, true)
				if err != nil {
					onGitErr("Failed to commit changes:", err.Error())
				}
			}
		}

		suffix := ""
		if len(updatedFiles) > 1 {
			suffix = "s"
		}
		fmt.Printf("✅ Applied changes, %d file%s updated\n", len(updatedFiles), suffix)
	}

}
