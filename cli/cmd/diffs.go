package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plandex/lib"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(diffsCmd)
}

var diffsCmd = &cobra.Command{
	Use:     "diffs",
	Aliases: []string{"d"},
	Short:   "Show differences for each individual file between plan files and project files",
	Run:     diffs,
}

func diffs(cmd *cobra.Command, args []string) {
	err := showDiffs()
	if err != nil {
		fmt.Println("An error occurred while generating diffs:", err)
	}
}

func showDiffs() error {
	// Check if git is installed
	if !lib.IsCommandAvailable("git") {
		return fmt.Errorf("git is required to generate diffs")
	}

	// Save current branch name
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	currentBranch, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error getting current branch: %v", err)
	}

	// Create and checkout a temporary branch
	cmd = exec.Command("git", "checkout", "-b", "temp_diff_branch")
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error creating temp branch: %v", err)
	}

	defer func() {
		// Switch back to the original branch and delete the temporary branch
		cmd = exec.Command("git", "checkout", string(currentBranch))
		err = cmd.Run()
		if err != nil {
			fmt.Println(fmt.Errorf("error switching back to original branch: %v", err))
		}

		cmd = exec.Command("git", "branch", "-D", "temp_diff_branch")
		err = cmd.Run()
		if err != nil {
			fmt.Println(fmt.Errorf("error deleting temp branch: %v", err))
		}

	}()

	// Commit any outstanding changes
	cmd = exec.Command("git", "add", ".")
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error adding changes: %v", err)
	}
	cmd = exec.Command("git", "commit", "-m", "temp commit")
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error committing changes: %v", err)
	}

	// Overwrite project files with plan files
	err = copyDirectory(lib.ProjectRoot, lib.PlanFilesDir)

	if err != nil {
		return fmt.Errorf("error copying files: %v", err)
	}

	// Add changes to the staging area
	cmd = exec.Command("git", "add", ".")
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error adding changes: %v", err)
	}

	// Show the diff
	cmd = exec.Command("git", "diff", "--cached")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error generating diff: %v", err)
	}

	return nil
}

func copyDirectory(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		if entry.IsDir() {
			err = copyDirectory(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			err = os.WriteFile(dstPath, data, 0644)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
