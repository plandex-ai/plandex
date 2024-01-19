package cmd

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	streamtui "plandex/stream_tui"
	"plandex/term"

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
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()
	lib.MustCheckOutdatedContextWithOutput()

	apiErr := api.Client.TellPlan(lib.CurrentPlanId, shared.TellPlanRequest{
		Prompt:        continuePrompt,
		ConnectStream: !continueBg,
	}, lib.OnStreamPlan)
	if apiErr != nil {
		if apiErr.Type == shared.ApiErrorTypeTrialMessagesExceeded {
			fmt.Fprintf(os.Stderr, "🚨 You've reached the free trial limit of %d messages per plan\n", apiErr.TrialMessagesExceededError.MaxMessages)

			res, err := term.ConfirmYesNo("Upgrade now?")

			if err != nil {
				fmt.Fprintln(os.Stderr, "Error prompting upgrade trial:", err)
				return
			}

			if res {
				err := auth.ConvertTrial()
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error converting trial:", err)
					return
				}
			}

			return
		}

		fmt.Fprintln(os.Stderr, "Prompt error:", apiErr.Msg)
		return
	}

	if !continueBg {
		err := streamtui.StartStreamUI()

		if err != nil {
			fmt.Fprintln(os.Stderr, "Error starting stream UI:", err)
			return
		}
	}
}
