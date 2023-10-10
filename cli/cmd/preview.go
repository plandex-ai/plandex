package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"plandex/lib"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(checkoutCmd)
}

var checkoutCmd = &cobra.Command{
	Use:     "preview [name]",
	Aliases: []string{"pv"},
	Short:   "Preview changes in a new branch",
	Args:    cobra.MaximumNArgs(1),
	Run:     checkout,
}

func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func checkout(cmd *cobra.Command, args []string) {
	if !lib.IsCommandAvailable("git") {
		log.Fatalln("Error: git is required")
	}

	var name string

	if len(args) > 0 {
		name = args[0]
		name = strings.TrimSpace(name)
	}

	output, err := exec.Command("git", "rev-parse", "--is-inside-work-tree").CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "true" {
		log.Fatalln("Error: please make sure you're inside of a git repository")
	}

	if name == "" || name == "current" {
		name = lib.CurrentPlanName
	}

	branchName := "pdx-" + name

	currentBranch, err := getCurrentBranch()
	if err != nil {
		log.Fatalf("Error: could not retrieve current branch: %v\n", err)
	}

	if currentBranch == branchName {
		fmt.Printf("Already on branch %s\n", branchName)
	} else {
		_, err = exec.Command("git", "checkout", "-b", branchName).CombinedOutput()
		fmt.Printf("✅ Checked out branch %s\n", branchName)
		if err != nil {
			log.Fatalf("Error: could not checkout to a new branch: %v\n", err)
		}
	}

	err = apply(cmd, args)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error committing plan: ", err)
		return
	}
	fmt.Println("✅ Applied changes")
}
