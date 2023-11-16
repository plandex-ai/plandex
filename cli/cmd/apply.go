package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"plandex/lib"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
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

	rootDir := filepath.Join(plandexDir, name)

	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return fmt.Errorf("error: plan with name '%+v' does not exist", name)
	}

	appliedAny := false
	currentPlanFiles, planResByPath, _, err := lib.GetCurrentPlanStateWithContext()

	if err != nil {
		return fmt.Errorf("error getting current plan files: %w", err)
	}

	// check if any files have been modified since the plan result was generated

	pathsRemoved := []string{}
	pathsOutdated := []string{}

	for path := range currentPlanFiles.Files {
		contextSha := currentPlanFiles.ContextShas[path]

		if contextSha == "" {
			// the path wasn't in context
			continue
		}

		// Compute destination path
		dstPath := filepath.Join(lib.ProjectRoot, path)

		// Check if the file has been removed
		_, err := os.Stat(dstPath)

		if os.IsNotExist(err) {
			pathsRemoved = append(pathsRemoved, path)
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

		if fileSha != contextSha {
			pathsOutdated = append(pathsOutdated, path)
		}
	}

	if len(pathsRemoved) > 0 || len(pathsOutdated) > 0 {
		fmt.Println("⚠️ Files in context have been modified since the plan was generated:")
		for _, path := range pathsRemoved {
			fmt.Println("  - " + path + " was removed")
		}
		for _, path := range pathsOutdated {
			fmt.Println("  - " + path + " was modified")
		}
		os.Exit(0)
	}

	isRepo := lib.CwdIsGitRepo()

	hasUncommittedChanges := false
	if isRepo {
		// Check if there are any uncommitted changes
		hasUncommittedChanges, err = lib.CheckUncommittedChanges()

		if err != nil {
			return fmt.Errorf("error checking for uncommitted changes: %w", err)
		}

		if hasUncommittedChanges {
			// If there are uncommitted changes, stash them
			err := lib.GitStashCreate("Plandex auto-stash")
			if err != nil {
				return fmt.Errorf("failed to create git stash: %w", err)
			}
		}
	}

	for path, content := range currentPlanFiles.Files {
		// Compute destination path
		dstPath := filepath.Join(lib.ProjectRoot, path)
		// Write the file
		err = os.WriteFile(dstPath, []byte(content), 0644)
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", dstPath, err)
		}
		appliedAny = true
	}

	if appliedAny {
		err := lib.SetPendingResultsApplied(planResByPath)
		if err != nil {
			return fmt.Errorf("failed to set pending results applied: %w", err)
		}

		if isRepo {
			desc, err := lib.GetLatestPlanDescription()
			if err != nil {
				return fmt.Errorf("failed to get latest plan description: %w", err)
			}

			// Commit the changes
			// After commit, try to pop the stash if one was created
			if hasUncommittedChanges {
				err := lib.GitStashPopNoConflict()
				if err != nil {
					return fmt.Errorf("failed to pop git stash: %w", err)
				}
			}
			err = lib.GitCommit(lib.ProjectRoot, "🤖 Plandex → "+desc.CommitMsg, true)
			if err != nil {
				return fmt.Errorf("failed to commit changes: %w", err)
			}
		}

		fmt.Println("✅ Applied changes")
	} else {
		return fmt.Errorf("this plan has no changes to apply")
	}

	return nil
}
