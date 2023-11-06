package cmd

import (
	"plandex/lib"

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
	updateRes := lib.MustUpdateContextWithOuput()
	table := lib.TableForContextUpdateRes(updateRes)
	table.Render()
}
