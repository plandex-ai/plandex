package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plandex/lib"

	"github.com/spf13/cobra"
)

const tempBranchName = "pdx_temp"

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

	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting current directory: %v", err)
	}

	// Create a temporary directory
	tempDir, err := os.MkdirTemp(os.TempDir(), "plandex-diffs-*")
	if err != nil {
		return fmt.Errorf("error creating temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Cleanup temporary directory

	// Copy relevant project files to the temporary directory
	paths, err := lib.GetCurrentPlanFilePaths()
	ch := make(chan error)
	for _, filePath := range paths {
		go func(filePath string) {
			ch <- lib.CopyFile(filepath.Join(lib.ProjectRoot, filePath), filepath.Join(tempDir, filePath))
		}(filePath)
	}

	for range paths {
		select {
		case err := <-ch:
			if err != nil {
				if !os.IsNotExist(err) {
					return fmt.Errorf("error copying file: %v", err)
				}
			}
		}
	}

	// Navigate to the temporary directory
	err = os.Chdir(tempDir)
	if err != nil {
		return fmt.Errorf("error changing to temp directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Initialize a new git repo in the temporary directory
	cmd := exec.Command("git", "init")
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error initializing git repo in temp directory: %v", err)
	}

	// Add and commit changes in the temporary directory
	cmd = exec.Command("git", "add", ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("error adding changes in temp directory: %v", err)
		fmt.Println(err)
		fmt.Println(string(output))
	}

	cmd = exec.Command("git", "commit", "-m", "temp commit", "--allow-empty")
	output, err = cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("error committing changes in temp directory: %v", err)
		fmt.Println(err)
		fmt.Println(string(output))
	}

	err = lib.CopyDir(lib.DraftFilesDir, tempDir)
	if err != nil {
		return fmt.Errorf("error copying files to temp directory: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	output, err = cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("error adding changes in temp directory: %v", err)
		fmt.Println(err)
		fmt.Println(string(output))
	}

	// Show the diff
	cmd = exec.Command("git", "diff", "--cached")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error showing diffs: %v", err)
	}

	return nil
}
