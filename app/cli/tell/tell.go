package tell

import (
	"fmt"
	"log"
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

func TellPlan(prompt string, tellBg, tellStop, tellNoBuild bool) {
	projectPaths, _, err := fs.GetProjectPaths()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting project paths:", err)
		return
	}

	log.Println("tellNoBuild:", tellNoBuild)

	var fn func() bool
	fn = func() bool {

		var buildMode shared.BuildMode
		if tellNoBuild {
			buildMode = shared.BuildModeNone
		} else {
			buildMode = shared.BuildModeAuto
		}

		apiErr := api.Client.TellPlan(lib.CurrentPlanId, lib.CurrentBranch, shared.TellPlanRequest{
			Prompt:        prompt,
			ConnectStream: !tellBg,
			AutoContinue:  !tellStop,
			ProjectPaths:  projectPaths,
			BuildMode:     buildMode,
			ApiKey:        os.Getenv("OPENAI_API_KEY"),
		}, stream.OnStreamPlan)
		if apiErr != nil {
			if apiErr.Type == shared.ApiErrorTypeTrialMessagesExceeded {
				streamtui.Quit()

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

		if !tellBg {
			go func() {
				err := streamtui.StartStreamUI(prompt, false)

				if err != nil {
					log.Printf("Error starting stream UI: %v\n", err)
					os.Exit(1)
				}

				fmt.Println()

				if tellStop {
					term.PrintCmds("", "continue", "convo", "changes", "log", "rewind")
				} else if tellNoBuild {
					term.PrintCmds("", "build", "convo", "log", "rewind")
				} else {
					term.PrintCmds("", "changes", "log", "rewind")
				}
				os.Exit(0)
			}()
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
