package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"plandex/lib"
	"plandex/types"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var autoConfirm bool

func init() {
	applyCmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "Automatically confirm unless plan is outdated")
	RootCmd.AddCommand(applyCmd)
}

var applyCmd = &cobra.Command{
	Use:     "apply [name]",
	Aliases: []string{"ap"},
	Short:   "Apply a plan to the project",
	Args:    cobra.MaximumNArgs(1),
	RunE:    apply,
}

func apply(cmd *cobra.Command, args []string) error {
	plandexDir, _, err := lib.FindOrCreatePlandex()
	var name string

	if len(args) > 0 {
		name = args[0]
		name = strings.TrimSpace(name)
	}

	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	if name == "" || name == "current" {
		name = lib.CurrentPlanName
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

	currentPlanFiles, planResByPath, _, err := lib.GetCurrentPlanStateWithContext()

	if err != nil {
		return fmt.Errorf("error getting current plan files: %w", err)
	}

	aborted := false

	// check if any files have been modified since the plan result was generated
	pathsRemoved := []string{}
	pathsOutdated := []string{}
	pathsUnmodified := []string{}
	pathsNew := []string{}
	pathsRemovedOrOutdatedSet := map[string]bool{}

	for path := range currentPlanFiles.Files {
		contextSha := currentPlanFiles.ContextShas[path]

		if contextSha == "" {
			// the path wasn't in context
			pathsNew = append(pathsNew, path)
			continue
		}

		// Compute destination path
		dstPath := filepath.Join(lib.ProjectRoot, path)

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
			table.Rich([]string{path, "🟡 Modified"}, []tablewriter.Colors{
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

			options = append(options,
				types.PlanOutdatedStrategyRebuild,
				types.PlanOutdatedStrategyCancel,
			)

			choice, err := lib.SelectFromList("🤔 How do you want to handle it?", options)

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

	case types.PlanOutdatedStrategyRebuild:
		fmt.Println("⚡️ Rebuilding the plan with updated context")

	case types.PlanOutdatedStrategyCancel:
		aborted = true
		return nil
	}

	isRepo := lib.CwdIsGitRepo()

	hasUncommittedChanges := false
	if isRepo {
		// Check if there are any uncommitted changes
		hasUncommittedChanges, err = lib.CheckUncommittedChanges()

		if err != nil {
			return fmt.Errorf("error checking for uncommitted changes: %w", err)
		}
	}

	toApply := map[string]string{}
	for path, content := range currentPlanFiles.Files {
		outdated := pathsRemovedOrOutdatedSet[path]
		if outdated {
			if conflictStrategy == types.PlanOutdatedStrategyOverwrite {
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
			shouldContinue, err := lib.ConfirmYesNo("Apply changes to %d file%s?", numToApply, suffix)

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
			err := lib.GitStashCreate("Plandex auto-stash")
			if err != nil {
				return fmt.Errorf("failed to create git stash: %w", err)
			}

			defer func() {
				if aborted {
					// clear any partially applied changes before popping the stash
					err := lib.GitClearUncommittedChanges()
					if err != nil {
						fmt.Printf("failed to clear uncommitted changes: %v\n", err)
					}
				}

				err := lib.GitStashPop(conflictStrategy)
				if err != nil {
					fmt.Printf("failed to pop git stash: %v\n", err)
				}
			}()
		}

		for path, content := range toApply {
			// Compute destination path
			dstPath := filepath.Join(lib.ProjectRoot, path)
			// Write the file
			err = os.WriteFile(dstPath, []byte(content), 0644)
			if err != nil {
				aborted = true
				return fmt.Errorf("failed to write %s: %w", dstPath, err)
			}
		}

		err := lib.SetPendingResultsApplied(planResByPath)
		if err != nil {
			aborted = true
			return fmt.Errorf("failed to set pending results applied: %w", err)
		}

		if isRepo {
			desc, err := lib.GetLatestPlanDescription()
			if err != nil {
				aborted = true
				return fmt.Errorf("failed to get latest plan description: %w", err)
			}

			// Commit the changes
			err = lib.GitAddAndCommit(lib.ProjectRoot, color.New(color.BgBlue, color.FgHiWhite, color.Bold).Sprintln(" 🤖 Plandex ")+desc.CommitMsg, true)
			if err != nil {
				aborted = true
				return fmt.Errorf("failed to commit changes: %w", err)
			}
		}

		fmt.Println("✅ Applied changes")
	}

	return nil
}
