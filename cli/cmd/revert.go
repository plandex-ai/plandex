package cmd

import (
	"github.com/spf13/cobra"
)

var revertCmd = &cobra.Command{
	Use:   "revert [commit-sha]",
	Short: "Revert plan to a previous commit",
	Long:  `Revert plan to a previous commit`,
	Args:  cobra.ExactArgs(1),
	Run:   revert,
}

func init() {
	RootCmd.AddCommand(revertCmd)
}

func revert(cmd *cobra.Command, args []string) {

}
