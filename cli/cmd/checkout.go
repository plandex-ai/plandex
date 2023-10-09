package cmd

import (
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
	Use:   "checkout [name]",
	Short: "Checkout to a new branch and apply a plan",
	Args:  cobra.ExactArgs(1),
	Run:   checkout,
}

func checkout(cmd *cobra.Command, args []string) {
	if !lib.IsCommandAvailable("git") {
		log.Fatalln("Error: git is required")
	}

	output, err := exec.Command("git", "rev-parse", "--is-inside-work-tree").CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "true" {
		log.Fatalln("Error: please make sure you're inside of a git repository")
	}

	branchName := "pdx_" + args[0]
	_, err = exec.Command("git", "checkout", "-b", branchName).CombinedOutput()

	if err != nil {
		log.Fatalln("Error: could not checkout to a new branch.")
	}

	err = apply(cmd, args)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error committing plan: ", err)
		return
	}
	fmt.Println("Plan applied and committed successfully to branch", branchName, "!")
}
