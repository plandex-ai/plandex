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

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
)

func TellPlan(
	params ExecParams,
	prompt string,
	tellBg,
	tellStop,
	tellNoBuild,
	isUserContinue,
	isDebugCmd,
	isChatOnly bool,
) {
	done := make(chan struct{})

	outputPromptIfTell := func() {
		if isUserContinue || prompt == "" {
			return
		}

		term.StopSpinner()
		// print prompt so it isn't lost
		color.New(term.ColorHiCyan, color.Bold).Println("\nYour prompt 👇")
		fmt.Println()
		fmt.Println(prompt)
		fmt.Println()
	}

	term.StartSpinner("")
	contexts, apiErr := api.Client.ListContext(params.CurrentPlanId, params.CurrentBranch)

	if apiErr != nil {
		outputPromptIfTell()
		term.OutputErrorAndExit("Error getting context: %v", apiErr)
	}

	anyOutdated, didUpdate, err := params.CheckOutdatedContext(contexts)

	if err != nil {
		outputPromptIfTell()
		term.OutputErrorAndExit("Error checking outdated context: %v", err)
	}

	if anyOutdated && !didUpdate {
		term.StopSpinner()
		if isUserContinue {
			log.Println("Plan won't continue")
		} else {
			log.Println("Prompt not sent")
		}

		outputPromptIfTell()
		color.New(term.ColorHiRed, color.Bold).Println("🛑 Plan won't continue due to outdated context")

		os.Exit(0)
	}

	term.StartSpinner("")
	paths, err := fs.GetProjectPaths(fs.GetBaseDirForContexts(contexts))
	term.StopSpinner()

	if err != nil {
		outputPromptIfTell()
		term.OutputErrorAndExit("Error getting project paths: %v", err)
	}

	var fn func() bool
	fn = func() bool {

		var buildMode shared.BuildMode
		if tellNoBuild || isChatOnly {
			buildMode = shared.BuildModeNone
		} else {
			buildMode = shared.BuildModeAuto
		}

		if isUserContinue {
			term.StartSpinner("⚡️ Continuing plan...")
		} else {
			term.StartSpinner("💬 Sending prompt...")
		}

		var legacyApiKey, openAIBase, openAIOrgId string

		if params.ApiKeys["OPENAI_API_KEY"] != "" {
			openAIBase = os.Getenv("OPENAI_API_BASE")
			if openAIBase == "" {
				openAIBase = os.Getenv("OPENAI_ENDPOINT")
			}

			legacyApiKey = params.ApiKeys["OPENAI_API_KEY"]
			openAIOrgId = params.ApiKeys["OPENAI_ORG_ID"]
		}

		apiErr := api.Client.TellPlan(params.CurrentPlanId, params.CurrentBranch, shared.TellPlanRequest{
			Prompt:         prompt,
			ConnectStream:  !tellBg,
			AutoContinue:   !tellStop,
			ProjectPaths:   paths.ActivePaths,
			BuildMode:      buildMode,
			IsUserContinue: isUserContinue,
			IsUserDebug:    isDebugCmd,
			IsChatOnly:     isChatOnly,
			ApiKey:         legacyApiKey, // deprecated
			Endpoint:       openAIBase,   // deprecated
			ApiKeys:        params.ApiKeys,
			OpenAIBase:     openAIBase,
			OpenAIOrgId:    openAIOrgId,
		}, stream.OnStreamPlan)

		term.StopSpinner()

		if apiErr != nil {
			if apiErr.Type == shared.ApiErrorTypeTrialMessagesExceeded {
				fmt.Fprintf(os.Stderr, "\n🚨 You've reached the Plandex Cloud trial limit of %d messages per plan\n", apiErr.TrialMessagesExceededError.MaxReplies)

				res, err := term.ConfirmYesNo("Upgrade now?")

				if err != nil {
					outputPromptIfTell()
					term.OutputErrorAndExit("Error prompting upgrade trial: %v", err)
				}

				if res {
					auth.ConvertTrial()
					// retry action after converting trial
					return fn()
				}

				outputPromptIfTell()
				return false
			}

			outputPromptIfTell()
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
					outputPromptIfTell()
					term.OutputErrorAndExit("Error starting stream UI: %v", err)
				}

				fmt.Println()

				if tellStop && !isChatOnly {
					term.PrintCmds("", "continue", "diff", "diff --ui", "apply", "log")
				} else if !isDebugCmd && !isChatOnly {
					term.PrintCmds("", "diff", "diff --ui", "apply", "log")
				} else if isChatOnly {
					term.PrintCmds("", "tell", "convo", "summary")
				}
				close(done)
			}()
		}

		return true
	}

	shouldContinue := fn()
	if !shouldContinue {
		return
	}

	if tellBg {
		outputPromptIfTell()
		fmt.Println("✅ Plan is active in the background")
		fmt.Println()
		term.PrintCmds("", "ps", "connect", "stop")
	} else {
		<-done
	}
}
