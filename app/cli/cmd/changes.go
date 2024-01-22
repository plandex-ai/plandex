package cmd

import (
	"fmt"
	"os"
	"plandex/auth"
	"plandex/changes_tui"
	"plandex/lib"

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
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	err := changes_tui.StartChangesUI()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting changes UI: %v\n", err)
	}

}
