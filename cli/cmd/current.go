package cmd

import (
	"fmt"
	"plandex/lib"

	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Get the current plan",
	Run:   current,
}

func init() {
	RootCmd.AddCommand(currentCmd)
}

func current(cmd *cobra.Command, args []string) {
	fmt.Println(lib.CurrentPlanName)
}
