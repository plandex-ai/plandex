package cmd

import (
	"plandex/lib"
	"strings"

	"github.com/spf13/cobra"
)

var autoConfirm bool

func init() {
	applyCmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "Automatically confirm unless plan is outdated")
	RootCmd.AddCommand(applyCmd)
}

var applyCmd = &cobra.Command{
	Use:     "apply [name]",
	Aliases: []string{"ap"},
	Short:   "Apply a plan to the project",
	Args:    cobra.MaximumNArgs(1),
	RunE:    apply,
}

func apply(cmd *cobra.Command, args []string) error {
	var name string

	if len(args) > 0 {
		name = args[0]
		name = strings.TrimSpace(name)
	}

	if name == "" || name == "current" {
		name = lib.CurrentPlanName
	}

	return lib.ApplyPlanWithOutput(name, autoConfirm)
}
