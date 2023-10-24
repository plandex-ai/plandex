package cmd

import (
	"github.com/spf13/cobra"
)

// rewindCmd represents the rewind command
var rewindCmd = &cobra.Command{
	Use:     "rewind [step]",
	Aliases: []string{"rw"},
	Short:   "Rewind the plan to an earlier state",
	Long:    `By default it rollback the conversation by one message. It can accept 'step' to rollback by more steps. It can also accept --sha to rollback to a specific version in the plan git history.`,
	Args:    cobra.MaximumNArgs(1),
	Run:     rewind,
}

var sha string

func init() {
	// Add rewind command
	RootCmd.AddCommand(rewindCmd)

	// Add sha flag
	rewindCmd.Flags().StringVar(&sha, "sha", "", "Specify a commit sha to rewind to")
}

func rewind(cmd *cobra.Command, args []string) {
	// Argument parsing and rewind logic will be implemented

}
