package cmd

import (
	"fmt"
	"plandex/lib"

	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:     "current",
	Aliases: []string{"c"},
	Short:   "Get the current plan",
	Run:     current,
}

func init() {
	RootCmd.AddCommand(currentCmd)
}

func current(cmd *cobra.Command, args []string) {
	if lib.CurrentPlanName == "" {
		fmt.Println("No current plan.")
		return
	}
	fmt.Println(lib.CurrentPlanName)
}
