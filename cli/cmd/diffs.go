package cmd

import (
	"bytes"
	"fmt"
	"os/exec"
	"plandex/lib"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(diffsCmd)
}

var diffsCmd = &cobra.Command{
	Use:     "diffs",
	Aliases: []string{"d"},
	Short:   "Show differences between plan and project files",
	Run:     diffs,
}

func diffs(cmd *cobra.Command, args []string) {
	err := showDiffs()
	if err != nil {
		fmt.Println("An error occurred while generating diffs:", err)
	}
}

// Show differences in 'git diff' style between plan files and project files.
func showDiffs() error {
	// Check git installed
	if !lib.IsCommandAvailable("git") {
		return fmt.Errorf("git is required to generate diffs")
	}

	// Run git diff command
	cmd := exec.Command("git", "diff", lib.PlanFilesDir, lib.ProjectRoot)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	if err != nil {
		return fmt.Errorf("error generating diff: %v", err)
	}

	// Print output
	fmt.Println(out.String())

	return nil
}