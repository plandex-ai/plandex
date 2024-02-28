package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/term"

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
		fmt.Fprintln(os.Stderr, "No current plan")
		return
	}

	planId, branch, shouldContinue := lib.SelectActiveStream(args)

	if !shouldContinue {
		return
	}

	apiErr := api.Client.StopPlan(planId, branch)

	if apiErr != nil {
		fmt.Fprintln(os.Stderr, "Error stopping stream:", apiErr.Msg)
		return
	}

	fmt.Println("✅ Plan stream stopped")

	fmt.Println()
	term.PrintCmds("", "convo", "log")

}
