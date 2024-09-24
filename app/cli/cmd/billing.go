package cmd

import (
	"plandex/auth"
	"plandex/term"
	"plandex/ui"

	"github.com/spf13/cobra"
)

var billingCmd = &cobra.Command{
	Use:   "billing",
	Short: "Open the billing page in the browser",
	Run:   billing,
}

func init() {
	RootCmd.AddCommand(billingCmd)
}

func billing(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	if !auth.Current.IsCloud {
		term.OutputErrorAndExit("This command is only available for Plandex Cloud accounts.")
	}

	ui.OpenAuthenticatedURL("Opening billing page in your default browser...", "/settings/billing")
}
