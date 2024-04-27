package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"

	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:     "rename [new-name]",
	Aliases: []string{"rn"},
	Short:   "Rename the current plan",
	Args:    cobra.MaximumNArgs(1),
	Run:     rename,
}

func init() {
	RootCmd.AddCommand(renameCmd)
}

func rename(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	var newName string
	if len(args) > 0 {
		newName = args[0]
	} else {
		var err error
		newName, err = term.Prompt("Enter new name for the current plan:")
		if err != nil {
			term.OutputErrorAndExit("Error reading new name: %v", err)
		}
	}

	if newName == "" {
		fmt.Println("No new name provided.")
		return
	}

	term.StartSpinner("Renaming plan")
	err := api.Client.RenamePlan(lib.CurrentPlanId, newName)
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error renaming plan: %v", err)
	}

	fmt.Printf("✅ Plan renamed to %s\n", newName)
}
