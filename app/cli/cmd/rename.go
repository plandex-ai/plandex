package cmd

import (
	"fmt"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename [new-name]",
	Short: "Rename the current plan",
	Args:  cobra.MaximumNArgs(1),
	Run:   rename,
}

func init() {
	RootCmd.AddCommand(renameCmd)
}

func rename(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	var newName string
	if len(args) > 0 {
		newName = args[0]
	} else {
		var err error
		newName, err = term.GetRequiredUserStringInput("New name:")
		if err != nil {
			term.OutputErrorAndExit("Error reading new name: %v", err)
		}
	}

	if newName == "" {
		fmt.Println("🤷‍♂️ No new name provided")
		return
	}

	term.StartSpinner("")
	err := api.Client.RenamePlan(lib.CurrentPlanId, newName)
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error renaming plan: %v", err)
	}

	fmt.Printf("✅ Plan renamed to %s\n", color.New(color.Bold, term.ColorHiGreen).Sprint(newName))
}
