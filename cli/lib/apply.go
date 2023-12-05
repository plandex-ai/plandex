package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"plandex/term"
	"plandex/types"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/plandex/plandex/shared"
)

func ApplyPlanWithOutput(name string, autoConfirm bool) error {
	plandexDir, _, err := FindOrCreatePlandex()
	if err != nil {
		return fmt.Errorf("error finding or creating plandex dir: %w", err)
	}

	if name == "" {
		fmt.Println("🤷‍♂️ No plan specified and no current plan")
		return nil
	}

	rootDir := filepath.Join(plandexDir, name)

	_, err = os.Stat(rootDir)

	if os.IsNotExist(err) {
		fmt.Printf("🤷‍♂️ Plan with name '%s' doesn't exist\n", name)
		return nil
	} else if err != nil {
		return fmt.Errorf("error checking if plan exists: %w", err)
	}

	res, err := GetCurrentPlanState()

	if err != nil {
		return fmt.Errorf("error getting current plan files: %w", err)
	}

	currentPlanFiles := res.CurrentPlanFiles
	planResByPath := res.PlanResByPath

	if len(currentPlanFiles.Files) == 0 {
		fmt.Println("🤷‍♂️ No changes to apply")
		return nil
	}

	aborted := false

	// check if any files have been modified since the plan result was generated
	pathsRemoved := []string{}
	pathsOutdated := []string{}
	pathsUnmodified := []string{}
	pathsNew := []string{}
	pathsRemovedOrOutdatedSet := map[string]bool{}
	outdatedCanMergeSet := map[string]bool{}

	canMergeAllOutdated := true

	for path := range currentPlanFiles.Files {
		contextSha := currentPlanFiles.ContextShas[path]

		if contextSha == "" {
			// the path wasn't in context
			pathsNew = append(pathsNew, path)
			continue
		}

		// Compute destination path
		dstPath := filepath.Join(ProjectRoot, path)

		// Check if the file has been removed
		_, err := os.Stat(dstPath)

		if os.IsNotExist(err) {
			pathsRemoved = append(pathsRemoved, path)
			pathsRemovedOrOutdatedSet[path] = true
		} else if err != nil {
			return fmt.Errorf("failed to check existence of %s: %w", dstPath, err)
		}

		// Read the file
		bytes, err := os.ReadFile(dstPath)

		if err != nil {
			return fmt.Errorf("failed to read %s: %w", dstPath, err)
		}

		// Get the sha of the file
		hash := sha256.Sum256(bytes)
		fileSha := hex.EncodeToString(hash[:])

		if fileSha == contextSha {
			pathsUnmodified = append(pathsUnmodified, path)
		} else {
			pathsOutdated = append(pathsOutdated, path)
			pathsRemovedOrOutdatedSet[path] = true

			planRes := planResByPath[path]

			updated := string(bytes)
			allSucceeded := true
			for _, res := range planRes {
				var succeeded bool
				updated, succeeded = shared.ApplyReplacements(updated, res.Replacements, false)
				if !succeeded {
					allSucceeded = false
					canMergeAllOutdated = false
					break
				}
			}
			outdatedCanMergeSet[path] = allSucceeded
		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Path", "Status"})
	table.SetAutoWrapText(false)

	numConflicts := len(pathsRemoved) + len(pathsOutdated)
	hasConflicts := numConflicts > 0

	if hasConflicts {
		suffix := ""
		verb := "has"
		if numConflicts > 1 {
			suffix = "s"
			verb = "have"
		}

		color.New(color.Bold).Printf("⚠️  %d file%s in context %s been modified since the plan was generated\n\n", numConflicts, suffix, verb)

		for _, path := range pathsRemoved {
			table.Rich([]string{path, "🔴 Removed"}, []tablewriter.Colors{
				{tablewriter.FgHiRedColor, tablewriter.Bold},
				{tablewriter.FgHiWhiteColor},
			})
		}

		for _, path := range pathsOutdated {
			s := "🟡 Modified"
			if outdatedCanMergeSet[path] {
				s += "  " + color.New(color.BgGreen).Sprint(" Can still apply changes ")
			} else {
				s += "  " + color.New(color.BgRed).Sprint(" Can't apply changes ")
			}

			table.Rich([]string{path, s}, []tablewriter.Colors{
				{tablewriter.FgHiYellowColor, tablewriter.Bold},
				{tablewriter.FgHiWhiteColor},
			})
		}
	} else {
		numUnmodified := len(pathsUnmodified) + len(pathsNew)
		suffix := ""

		if numUnmodified > 1 {
			suffix = "s"
		}

		color.New(color.Bold).Printf("⚡️ %d file%s will be updated\n\n", numUnmodified, suffix)
	}

	for _, path := range pathsUnmodified {
		table.Rich([]string{path, "🟢 Ready for update"}, []tablewriter.Colors{
			{tablewriter.FgHiGreenColor, tablewriter.Bold},
			{tablewriter.FgHiWhiteColor},
		})
	}

	for _, path := range pathsNew {
		table.Rich([]string{path, "🔵 New file"}, []tablewriter.Colors{
			{tablewriter.FgHiCyanColor, tablewriter.Bold},
			{tablewriter.FgHiWhiteColor},
		})
	}

	table.Render()

	var conflictStrategy string

	if hasConflicts {
		fmt.Println()

		if conflictStrategy == "" {
			options := []string{
				types.PlanOutdatedStrategyOverwrite,
			}

			if len(currentPlanFiles.Files) > (len(pathsRemoved) + len(pathsOutdated)) {
				options = append(options, types.PlanOutdatedStrategyApplyUnmodified)
			}

			if canMergeAllOutdated {
				options = append(options, types.PlanOutdatedStrategyApplyNoConflicts)
			}

			options = append(options,
				// types.PlanOutdatedStrategyRebuild, // TODO: implement rebuild
				types.PlanOutdatedStrategyCancel,
			)

			choice, err := term.SelectFromList("🤔 How do you want to handle it?", options)

			if err != nil {
				return fmt.Errorf("failed to get user input: %w", err)
			}

			conflictStrategy = choice
		}
	}

	switch conflictStrategy {
	case types.PlanOutdatedStrategyOverwrite:
		fmt.Println("⚡️ Overwriting modifications")

	case types.PlanOutdatedStrategyApplyUnmodified:
		fmt.Println("⚡️ Applying only new and unmodified files")

	case types.PlanOutdatedStrategyApplyNoConflicts:
		// nothing to do

	case types.PlanOutdatedStrategyRebuild:
		fmt.Println("⚡️ Rebuilding the plan with updated context")
		// TODO: rebuild the plan
		return nil

	case types.PlanOutdatedStrategyCancel:
		aborted = true
		return nil
	}

	isRepo := CwdIsGitRepo()

	hasUncommittedChanges := false
	if isRepo {
		// Check if there are any uncommitted changes
		hasUncommittedChanges, err = CheckUncommittedChanges()

		if err != nil {
			return fmt.Errorf("error checking for uncommitted changes: %w", err)
		}
	}

	toApply := map[string]string{}
	for path, content := range currentPlanFiles.Files {
		outdated := pathsRemovedOrOutdatedSet[path]
		if outdated {
			if conflictStrategy == types.PlanOutdatedStrategyOverwrite || conflictStrategy == types.PlanOutdatedStrategyApplyNoConflicts {
				toApply[path] = content
			}
		} else {
			toApply[path] = content
		}
	}

	if len(toApply) == 0 {
		fmt.Println("🤷‍♂️ No changes to apply")
	} else {
		if !autoConfirm && !hasConflicts {
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
						fmt.Printf("failed to clear uncommitted changes: %v\n", err)
					}
				}

				err := GitStashPop(conflictStrategy)
				if err != nil {
					fmt.Printf("failed to pop git stash: %v\n", err)
				}
			}()
		}

		for path, content := range toApply {
			// Compute destination path
			dstPath := filepath.Join(ProjectRoot, path)
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

		err := SetPendingResultsApplied(planResByPath)
		if err != nil {
			aborted = true
			return fmt.Errorf("failed to set pending results applied: %w", err)
		}

		if isRepo {
			desc, err := GetLatestPlanDescription()
			if err != nil {
				aborted = true
				return fmt.Errorf("failed to get latest plan description: %w", err)
			}

			// Commit the changes
			err = GitAddAndCommit(ProjectRoot, color.New(color.BgBlue, color.FgHiWhite, color.Bold).Sprintln(" 🤖 Plandex ")+desc.CommitMsg, true)
			if err != nil {
				aborted = true
				return fmt.Errorf("failed to commit changes: %w", err)
			}
		}

		fmt.Println("✅ Applied changes")
	}

	return nil

}
