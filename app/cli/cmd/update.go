package cmd

import (
	"fmt"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:     "update ",
	Aliases: []string{"u"},
	Short:   "Update outdated context",
	Args:    cobra.MaximumNArgs(1),
	Run:     update,
}

func init() {
	RootCmd.AddCommand(updateCmd)

}

func update(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	term.StartSpinner("")
	outdated, err := lib.CheckOutdatedContext(nil)

	if err != nil {
		term.StopSpinner()
		term.OutputErrorAndExit("failed to check outdated context: %s", err)
	}

	if len(outdated.UpdatedContexts) == 0 {
		term.StopSpinner()
		fmt.Println("✅ Context is up to date")
		return
	}

	lib.MustUpdateContext(nil)
}
