package cmd

import (
	"fmt"
	"os"
	"plandex/changes_tui"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(changesCmd)
}

var changesCmd = &cobra.Command{
	Use:     "changes",
	Aliases: []string{"ch"},
	Short:   "View, copy, or manage changes for the current plan",
	Run:     changes,
}

func changes(cmd *cobra.Command, args []string) {

	err := changes_tui.StartChangesUI()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting changes UI: %v\n", err)
	}

}
