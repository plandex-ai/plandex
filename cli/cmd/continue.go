package cmd

import (
	"fmt"
	"os"
	"plandex/lib"

	"github.com/spf13/cobra"
)

// Variables to be used in the nextCmd
const continuePrompt = "continue the plan"

// nextCmd represents the prompt command
var nextCmd = &cobra.Command{
	Use:     "continue",
	Aliases: []string{"c"},
	Short:   "Continue the plan.",
	Run:     next,
}

func init() {
	RootCmd.AddCommand(nextCmd)
}

func next(cmd *cobra.Command, args []string) {
	lib.MustResolveProject()

	err := lib.Propose(continuePrompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Prompt error:", err)
		return
	}
}
