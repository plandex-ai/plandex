package plan_exec

import (
	"fmt"
	"log"
	"os"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/fs"
	"plandex-cli/lib"
	"plandex-cli/stream"
	streamtui "plandex-cli/stream_tui"
	"plandex-cli/term"

	"strings"

	shared "plandex-shared"

	"github.com/eiannone/keyboard"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

type hotkeyOption struct {
	key          string
	command      string
	description  string
	replOnly     bool
	terminalOnly bool
}

var allHotkeyOptions = []hotkeyOption{
	{"d", "diff ui", "Review diffs in browser UI", false, false},
	{"g", "git diff format", "Review diffs in git diff format", false, false},
	{"a", "apply", "Apply all pending changes", false, false},
	{"r", "reject", "Reject some or all pending changes", false, false},
	{"f", "follow up", "Iterate with a follow up prompt", true, false},
	// {"q", "quit", "Back to terminal", false, true},
	// {"q", "quit", "Back to REPL", true, false},
}

func TellPlan(
	params ExecParams,
	prompt string,
	flags TellFlags,
) {
	// showHotkeyMenu([]string{"a", "r"})
	// handleHotkey([]string{"a", "r"}, params)
	// return

	tellBg := flags.TellBg
	tellStop := flags.TellStop
	tellNoBuild := flags.TellNoBuild
	isUserContinue := flags.IsUserContinue
	isDebugCmd := flags.IsUserDebug
	isChatOnly := flags.IsChatOnly
	autoContext := flags.AutoContext
	execEnabled := flags.ExecEnabled

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

		apiErr := api.Client.TellPlan(params.CurrentPlanId, params.CurrentBranch, shared.TellPlanRequest{
			Prompt:         prompt,
			ConnectStream:  !tellBg,
			AutoContinue:   !tellStop,
			ProjectPaths:   paths.ActivePaths,
			BuildMode:      buildMode,
			IsUserContinue: isUserContinue,
			IsUserDebug:    isDebugCmd,
			IsChatOnly:     isChatOnly,
			AutoContext:    autoContext,
			ExecEnabled:    execEnabled,
			OsDetails:      osDetails,
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

				diffs, apiErr := getDiffs(params)
				numDiffs := len(diffs)
				if apiErr != nil {
					term.OutputErrorAndExit("Error getting plan diffs: %v", apiErr.Msg)
					return
				}
				hasDiffs := numDiffs > 0

				fmt.Println()

				if tellStop && !isChatOnly && hasDiffs {
					if hasDiffs {
						// term.PrintCmds("", "continue", "diff", "diff --ui", "apply", "reject", "log")
						showHotkeyMenu(diffs)
						handleHotkey(diffs, params)
					} else {
						term.PrintCmds("", "continue", "log")
					}
				} else if !isDebugCmd && !isChatOnly && hasDiffs {
					// term.PrintCmds("", "diff", "diff --ui", "apply", "reject", "log")
					showHotkeyMenu(diffs)
					handleHotkey(diffs, params)
				} else if isChatOnly {
					if !term.IsRepl {
						term.PrintCmds("", "tell", "convo", "summary", "log")
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

func showHotkeyMenu(diffs []string) {
	numDiffs := len(diffs)
	s := "files have"
	if numDiffs == 1 {
		s = "file has"
	}
	color.New(color.Bold, term.ColorHiGreen).Printf("🧐 %d %s pending changes\n", numDiffs, s)

	// for _, diff := range diffs {
	// 	fmt.Printf("• %s\n", diff)
	// }

	fmt.Println()

	var b strings.Builder
	table := tablewriter.NewWriter(&b)
	table.SetAutoWrapText(false)
	table.SetHeaderLine(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, opt := range allHotkeyOptions {
		if (opt.terminalOnly && term.IsRepl) || (opt.replOnly && !term.IsRepl) {
			continue
		}

		table.Append([]string{
			color.New(term.ColorHiGreen, color.Bold).Sprintf("(%s)", opt.key),
			opt.command,
			opt.description,
		})
	}

	table.Render()
	fmt.Print(b.String())
	color.New(term.ColorHiMagenta, color.Bold).Printf("Press a hotkey, %s %s, or %s %s> ", color.New(color.FgHiWhite, color.Bold).Sprintf("↓"), color.New(term.ColorHiMagenta, color.Bold).Sprintf("to select"), color.New(color.FgHiWhite, color.Bold).Sprintf("q"), color.New(term.ColorHiMagenta, color.Bold).Sprintf("to quit"))
}

func handleHotkey(diffs []string, params ExecParams) {
	char, key, err := term.GetUserKeyInput()
	if err != nil {
		fmt.Printf("\nError getting key: %v\n", err)
		showHotkeyMenu(diffs)
		handleHotkey(diffs, params)
	}

	if key == keyboard.KeyArrowDown {
		options := []string{}
		for _, opt := range allHotkeyOptions {
			if (opt.terminalOnly && term.IsRepl) || (opt.replOnly && !term.IsRepl) {
				continue
			}

			options = append(options, opt.description)
		}

		selected, err := term.SelectFromList(
			"Select an action",
			options,
		)
		if err != nil {
			fmt.Printf("\nError selecting action: %v\n", err)
			showHotkeyMenu(diffs)
			handleHotkey(diffs, params)
		}

		if selected != "" {
			var option hotkeyOption
			for _, opt := range allHotkeyOptions {
				if opt.description == selected {
					if (opt.terminalOnly && term.IsRepl) || (opt.replOnly && !term.IsRepl) {
						continue
					}

					option = opt
					break
				}
			}

			handleHotkeyOption(option, diffs, params)
		}
	}

	handleHotkeyOption(hotkeyOption{key: string(char)}, diffs, params)
}

func handleHotkeyOption(option hotkeyOption, diffs []string, params ExecParams) {
	exitUnlessDiffs := func() {
		diffs, apiErr := getDiffs(params)
		if apiErr != nil {
			fmt.Printf("\nError getting plan diffs: %v\n", apiErr.Msg)
			os.Exit(0)
		}

		if len(diffs) == 0 {
			os.Exit(0)
		}

		showHotkeyMenu(diffs)
		handleHotkey(diffs, params)
	}

	fmt.Println()

	switch option.key {
	case "d":
		_, err := lib.ExecPlandexCommandWithParams([]string{"diffs", "--ui"}, lib.ExecPlandexCommandParams{
			DisableSuggestions: true,
		})
		if err != nil {
			fmt.Printf("\nError showing diffs: %v\n", err)
		}

	case "g":
		_, err := lib.ExecPlandexCommandWithParams([]string{"diffs"}, lib.ExecPlandexCommandParams{
			DisableSuggestions: true,
		})
		if err != nil {
			fmt.Printf("\nError showing diffs: %v\n", err)
		}

	case "a":
		_, err := lib.ExecPlandexCommand([]string{"apply"})
		if err != nil {
			fmt.Printf("\nError applying changes: %v\n", err)
		}
		exitUnlessDiffs()

	case "r":
		_, err := lib.ExecPlandexCommand([]string{"reject"})
		if err != nil {
			fmt.Printf("\nError rejecting changes: %v\n", err)
		}
		exitUnlessDiffs()

	case "q":
		os.Exit(0)

	case "f":
		if term.IsRepl {
			color.New(color.Bold).Println("Write a prompt 👇")
			os.Exit(0)
		} else {
			term.PrintCmds("", "tell", "chat")
			os.Exit(0)
		}

	default:
		fmt.Println("\nInvalid command")
	}

	showHotkeyMenu(diffs)
	handleHotkey(diffs, params)
}

func getDiffs(params ExecParams) ([]string, *shared.ApiError) {
	currentPlan, apiErr := api.Client.GetCurrentPlanState(params.CurrentPlanId, params.CurrentBranch)
	if apiErr != nil {
		return nil, apiErr
	}

	return currentPlan.PlanResult.SortedPaths, nil
}
