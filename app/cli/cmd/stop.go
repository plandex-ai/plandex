package cmd

import (
	"fmt"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/term"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [stream-id-or-plan] [branch]",
	Short: "Connect to an active stream",
	// Long:  ``,
	Args: cobra.MaximumNArgs(2),
	Run:  stop,
}

func init() {
	RootCmd.AddCommand(stopCmd)
}

func stop(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	planId, branch, shouldContinue := lib.SelectActiveStream(args)

	if !shouldContinue {
		return
	}

	term.StartSpinner("")
	apiErr := api.Client.StopPlan(planId, branch)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error stopping stream: %v", apiErr.Msg)
	}

	fmt.Println("✅ Plan stream stopped")

	fmt.Println()
	term.PrintCmds("", "convo", "log")

}
