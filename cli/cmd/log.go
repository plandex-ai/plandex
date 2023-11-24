package cmd

import (
	"fmt"
	"os"
	"plandex/lib"
	"plandex/term"

	"github.com/spf13/cobra"
)

// logCmd represents the log command
var logCmd = &cobra.Command{
	Use:     "log [scope]",
	Aliases: []string{"history"},
	Short:   "Show plan history",
	Long:    `Show plan history. Pass an optional scope to log only a specific part of the plan. Valid scopes are "convo", "context", and "draft". If no scope is passed, the history of the entire plan will be output.`,
	Args:    cobra.MaximumNArgs(1),
	Run:     runLog,
}

func init() {
	// Add log command
	RootCmd.AddCommand(logCmd)
}

func runLog(cmd *cobra.Command, args []string) {
	var dir string
	switch {
	case len(args) == 0:
		dir = lib.CurrentPlanDir
	case args[0] == "convo":
		dir = lib.ConversationSubdir
	case args[0] == "context":
		dir = lib.ContextSubdir
	case args[0] == "draft":
		dir = lib.ResultsSubdir
	default:
		fmt.Fprint(os.Stderr, "Invalid scope. Valid scopes are 'convo', 'context', and 'draft'")
		os.Exit(1)
	}

	history, err := lib.GetGitCommitHistory(dir)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	term.PageOutput(history)
}
