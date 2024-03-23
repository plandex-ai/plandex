package plan_exec

import (
	"fmt"
	"log"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/fs"
	"plandex/stream"
	streamtui "plandex/stream_tui"
	"plandex/term"

	"github.com/plandex/plandex/shared"
)

func TellPlan(
	params ExecParams,
	prompt string,
	tellBg,
	tellStop,
	tellNoBuild,
	isUserContinue bool,
) {
	term.StartSpinner("")
	contexts, apiErr := api.Client.ListContext(params.CurrentPlanId, params.CurrentBranch)

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting context: %v", apiErr)
	}

	anyOutdated, didUpdate := params.CheckOutdatedContext(contexts)

	if anyOutdated && !didUpdate {
		term.StopSpinner()
		if isUserContinue {
			log.Println("Plan won't continue")
		} else {
			log.Println("Prompt not sent")
		}
		os.Exit(0)
	}

	paths, err := fs.GetProjectPaths(fs.GetBaseDirForContexts(contexts))

	if err != nil {
		term.OutputErrorAndExit("Error getting project paths: %v", err)
	}

	var fn func() bool
	fn = func() bool {

		var buildMode shared.BuildMode
		if tellNoBuild {
			buildMode = shared.BuildModeNone
		} else {
			buildMode = shared.BuildModeAuto
		}

		if isUserContinue {
			term.StartSpinner("⚡️ Continuing plan...")
		} else {
			term.StartSpinner("💬 Sending prompt...")
		}

		apiErr := api.Client.TellPlan(params.CurrentPlanId, params.CurrentBranch, shared.TellPlanRequest{
			Prompt:         prompt,
			ConnectStream:  !tellBg,
			AutoContinue:   !tellStop,
			ProjectPaths:   paths.ActivePaths,
			BuildMode:      buildMode,
			IsUserContinue: isUserContinue,
			ApiKey:         os.Getenv("OPENAI_API_KEY"),
		}, stream.OnStreamPlan)

		term.StopSpinner()

		if apiErr != nil {
			if apiErr.Type == shared.ApiErrorTypeTrialMessagesExceeded {
				fmt.Fprintf(os.Stderr, "\n🚨 You've reached the Plandex Cloud anonymous trial limit of %d messages per plan\n", apiErr.TrialMessagesExceededError.MaxReplies)

				res, err := term.ConfirmYesNo("Upgrade to an unlimited free account?")

				if err != nil {
					term.OutputErrorAndExit("Error prompting upgrade trial: %v", err)
				}

				if res {
					err := auth.ConvertTrial()
					if err != nil {
						term.OutputErrorAndExit("Error converting trial: %v", err)
					}
					// retry action after converting trial
					return fn()
				}
				return false
			}

			term.OutputErrorAndExit("Prompt error: %v", apiErr.Msg)
		} else if apiErr != nil && isUserContinue && apiErr.Type == shared.ApiErrorTypeContinueNoMessages {
			fmt.Println("🤷‍♂️ There's no plan yet to continue")
			fmt.Println()
			term.PrintCmds("", "tell")
			os.Exit(0)
		}

		if !tellBg {
			go func() {
				err := streamtui.StartStreamUI(prompt, false)

				if err != nil {
					term.OutputErrorAndExit("Error starting stream UI: %v", err)
				}

				fmt.Println()

				if tellStop {
					term.PrintCmds("", "continue", "changes", "apply", "log", "rewind")
				} else {
					term.PrintCmds("", "changes", "apply", "log", "rewind")
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
		fmt.Println("✅ Plan is active in the background")
		fmt.Println()
		term.PrintCmds("", "ps", "connect", "stop")
	} else {
		// Wait for stream UI to quit
		select {}
	}
}
