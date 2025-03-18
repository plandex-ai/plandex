package plan_exec

import (
	"fmt"
	"log"
	"os"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/fs"
	"plandex-cli/stream"
	streamtui "plandex-cli/stream_tui"
	"plandex-cli/term"
	"plandex-cli/types"

	shared "plandex-shared"

	"github.com/fatih/color"
)

func TellPlan(
	params ExecParams,
	prompt string,
	flags types.TellFlags,
) {

	tellBg := flags.TellBg
	tellStop := flags.TellStop
	tellNoBuild := flags.TellNoBuild
	isUserContinue := flags.IsUserContinue
	isDebugCmd := flags.IsUserDebug
	isChatOnly := flags.IsChatOnly
	autoContext := flags.AutoContext
	smartContext := flags.SmartContext
	execEnabled := flags.ExecEnabled
	autoApply := flags.AutoApply
	isApplyDebug := flags.IsApplyDebug
	isImplementationOfChat := flags.IsImplementationOfChat
	done := make(chan struct{})

	if prompt == "" && isImplementationOfChat {
		prompt = "Go ahead with the plan based on what we've discussed so far."
	}

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

	paths, err := fs.GetProjectPaths(fs.GetBaseDirForContexts(contexts))

	if err != nil {
		outputPromptIfTell()
		term.OutputErrorAndExit("Error getting project paths: %v", err)
	}

	anyOutdated, didUpdate, err := params.CheckOutdatedContext(contexts, paths)

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

	var fn func() bool
	fn = func() bool {

		var buildMode shared.BuildMode
		if tellNoBuild || isChatOnly {
			buildMode = shared.BuildModeNone
		} else {
			buildMode = shared.BuildModeAuto
		}

		// if isUserContinue {
		// 	term.StartSpinner("⚡️ Continuing plan...")
		// } else {
		// 	term.StartSpinner("💬 Sending prompt...")
		// }

		term.StartSpinner("")

		var legacyApiKey, openAIBase, openAIOrgId string

		if params.ApiKeys["OPENAI_API_KEY"] != "" {
			openAIBase = os.Getenv("OPENAI_API_BASE")
			if openAIBase == "" {
				openAIBase = os.Getenv("OPENAI_ENDPOINT")
			}

			legacyApiKey = params.ApiKeys["OPENAI_API_KEY"]
			openAIOrgId = params.ApiKeys["OPENAI_ORG_ID"]
		}

		var osDetails string
		if execEnabled {
			osDetails = term.GetOsDetails()
		}

		isGitRepo := fs.ProjectRootIsGitRepo()

		apiErr := api.Client.TellPlan(params.CurrentPlanId, params.CurrentBranch, shared.TellPlanRequest{
			Prompt:                 prompt,
			ConnectStream:          !tellBg,
			AutoContinue:           !tellStop,
			ProjectPaths:           paths.ActivePaths,
			BuildMode:              buildMode,
			IsUserContinue:         isUserContinue,
			IsUserDebug:            isDebugCmd,
			IsChatOnly:             isChatOnly,
			AutoContext:            autoContext,
			SmartContext:           smartContext,
			ExecEnabled:            execEnabled,
			OsDetails:              osDetails,
			ApiKey:                 legacyApiKey, // deprecated
			Endpoint:               openAIBase,   // deprecated
			ApiKeys:                params.ApiKeys,
			OpenAIBase:             openAIBase,
			OpenAIOrgId:            openAIOrgId,
			IsImplementationOfChat: isImplementationOfChat,
			IsGitRepo:              isGitRepo,
			SessionId:              os.Getenv("PLANDEX_REPL_SESSION_ID"),
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
				err := streamtui.StartStreamUI(
					prompt,
					false,
					!(autoApply || autoContext || isApplyDebug || isDebugCmd),
				)

				if err != nil {
					outputPromptIfTell()
					term.OutputErrorAndExit("Error starting stream UI: %v", err)
				}

				if isChatOnly {
					if !term.IsRepl {
						term.PrintCmds("", "tell", "convo", "summary", "log")
					}
				} else if autoApply || isDebugCmd || isApplyDebug {
					// do nothing, allow auto apply to run
				} else {
					term.StartSpinner("")
					diffs, apiErr := getDiffs(params)
					term.StopSpinner()
					if apiErr != nil {
						term.OutputErrorAndExit("Error getting plan diffs: %v", apiErr.Msg)
						return
					}
					numDiffs := len(diffs)
					hasDiffs := numDiffs > 0

					fmt.Println()

					if tellStop && hasDiffs {
						if hasDiffs {
							// term.PrintCmds("", "continue", "diff", "diff --ui", "apply", "reject", "log")
							showHotkeyMenu(diffs)
							handleHotkey(diffs, params)
						} else {
							term.PrintCmds("", "continue", "log")
						}
					} else if hasDiffs {
						// term.PrintCmds("", "diff", "diff --ui", "apply", "reject", "log")
						showHotkeyMenu(diffs)
						handleHotkey(diffs, params)
					}
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
