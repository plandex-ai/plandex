package cmd

import (
	"fmt"
	"log"
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
const continuePrompt = "Continue the plan."

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
	if os.Getenv("OPENAI_API_KEY") == "" {
		term.OutputNoApiKeyMsg()
		os.Exit(1)
	}

	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()
	lib.MustCheckOutdatedContextWithOutput()

	promptingTrialExceeded := false
	if !continueBg {
		go func() {
			err := streamtui.StartStreamUI()

			if err != nil {
				fmt.Fprintln(os.Stderr, "Error starting stream UI:", err)
				os.Exit(1)
			}

			if !promptingTrialExceeded {
				os.Exit(0)
			}
		}()
	}

	var fn func() bool
	fn = func() bool {
		apiErr := api.Client.TellPlan(lib.CurrentPlanId, lib.CurrentBranch, shared.TellPlanRequest{
			Prompt:        continuePrompt,
			ConnectStream: !continueBg,
			ApiKey:        os.Getenv("OPENAI_API_KEY"),
		}, lib.OnStreamPlan)
		if apiErr != nil {
			log.Println("Error telling plan:", apiErr)
			if apiErr.Type == shared.ApiErrorTypeTrialMessagesExceeded {
				promptingTrialExceeded = true
				streamtui.Quit()
				promptingTrialExceeded = false

				fmt.Fprintf(os.Stderr, "🚨 You've reached the free trial limit of %d messages per plan\n", apiErr.TrialMessagesExceededError.MaxReplies)

				res, err := term.ConfirmYesNo("Upgrade now?")

				if err != nil {
					fmt.Fprintln(os.Stderr, "Error prompting upgrade trial:", err)
					return false
				}

				if res {
					err := auth.ConvertTrial()
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error converting trial:", err)
						return false
					}
					// retry action after converting trial
					return fn()
				}

				return false
			}

			fmt.Fprintln(os.Stderr, "Prompt error:", apiErr.Msg)
			return false
		}
		return true
	}

	shouldContinue := fn()
	if !shouldContinue {
		return
	}

	if tellBg {
		fmt.Println("✅ Prompt sent")
	} else {
		// Wait for stream UI to quit
		select {}
	}

}
