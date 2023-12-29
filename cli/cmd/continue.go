package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/lib"
	streamtui "plandex/stream_tui"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

// Variables to be used in the nextCmd
const continuePrompt = "continue the plan"

var continueBg bool

// nextCmd represents the prompt command
var nextCmd = &cobra.Command{
	Use:     "continue",
	Aliases: []string{"c"},
	Short:   "Continue the plan.",
	Run:     next,
}

func init() {
	RootCmd.AddCommand(nextCmd)

	nextCmd.Flags().BoolVar(&continueBg, "bg", false, "Execute autonomously in the background")
}

func next(cmd *cobra.Command, args []string) {
	lib.MustResolveProject()
	lib.MustCheckOutdatedContextWithOutput()

	err := api.Client.TellPlan(lib.CurrentPlanId, shared.TellPlanRequest{
		Prompt:        continuePrompt,
		ConnectStream: !continueBg,
	}, lib.OnStreamPlan)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Prompt error:", err)
		return
	}

	if !continueBg {
		err = streamtui.StartStreamUI()

		if err != nil {
			fmt.Fprintln(os.Stderr, "Error starting stream UI:", err)
			return
		}
	}
}
