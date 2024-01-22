package cmd

import (
	"plandex/auth"
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
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	lib.MustUpdateContextWithOuput()
}
