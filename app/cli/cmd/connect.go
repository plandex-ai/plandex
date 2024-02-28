package cmd

import (
	"fmt"
	"log"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/stream"
	streamtui "plandex/stream_tui"
	"plandex/term"

	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:     "connect [stream-id-or-plan] [branch]",
	Aliases: []string{"conn"},
	Short:   "Connect to an active stream",
	// Long:  ``,
	Args: cobra.MaximumNArgs(2),
	Run:  connect,
}

func init() {
	RootCmd.AddCommand(connectCmd)

}

func connect(cmd *cobra.Command, args []string) {
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

	apiErr := api.Client.ConnectPlan(planId, branch, stream.OnStreamPlan)

	if apiErr != nil {
		fmt.Fprintln(os.Stderr, "Error connecting to stream:", apiErr.Msg)
		return
	}

	go func() {
		err := streamtui.StartStreamUI("", false)

		if err != nil {
			log.Printf("Error starting stream UI: %v\n", err)
			os.Exit(1)
		}

		fmt.Println()
		term.PrintCmds("", "changes", "apply", "log")

		os.Exit(0)
	}()

	// Wait for the stream to finish
	select {}
}
