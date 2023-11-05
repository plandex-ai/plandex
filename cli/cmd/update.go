package cmd

import (
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:     "update ",
	Aliases: []string{"u"},
	Short:   "Update outdated context",
	Args:    cobra.MaximumNArgs(1),
	Run:     update,
}

func init() {
	// Add update command
	RootCmd.AddCommand(updateCmd)

}

func update(cmd *cobra.Command, args []string) {
	// Argument parsing and update logic will be implemented

}
