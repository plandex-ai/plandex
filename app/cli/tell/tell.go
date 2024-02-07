package tell

import (
	"fmt"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/fs"
	"plandex/lib"
	"plandex/stream"
	streamtui "plandex/stream_tui"
	"plandex/term"

	"github.com/plandex/plandex/shared"
)

func TellPlan(prompt string, tellBg, tellStep bool) {
	projectPaths, _, err := fs.GetProjectPaths()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting project paths:", err)
		return
	}

	promptingTrialExceeded := false
	var fn func() bool
	fn = func() bool {
		if !tellBg {
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

		apiErr := api.Client.TellPlan(lib.CurrentPlanId, lib.CurrentBranch, shared.TellPlanRequest{
			Prompt:        prompt,
			ConnectStream: !tellBg,
			AutoContinue:  !tellStep,
			ProjectPaths:  projectPaths,
			ApiKey:        os.Getenv("OPENAI_API_KEY"),
		}, stream.OnStreamPlan)
		if apiErr != nil {
			if apiErr.Type == shared.ApiErrorTypeTrialMessagesExceeded {
				promptingTrialExceeded = true
				streamtui.Quit()
				promptingTrialExceeded = false

				fmt.Fprintf(os.Stderr, "\n🚨 You've reached the free trial limit of %d messages per plan\n", apiErr.TrialMessagesExceededError.MaxReplies)

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
