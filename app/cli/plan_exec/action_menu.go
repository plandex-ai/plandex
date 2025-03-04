package plan_exec

import (
	"fmt"
	"os"
	"plandex-cli/api"
	"plandex-cli/lib"
	"plandex-cli/term"
	shared "plandex-shared"
	"strings"

	"github.com/eiannone/keyboard"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

type hotkeyOption struct {
	char            string
	key             keyboard.Key
	command         string
	description     string
	replOnly        bool
	terminalOnly    bool
	dropdownOnly    bool
	applyScriptOnly bool
}

var allHotkeyOptions = []hotkeyOption{
	{
		char:         "d",
		command:      "diff ui",
		description:  "Review diffs in browser UI",
		replOnly:     false,
		terminalOnly: false,
	},
	{
		char:         "g",
		command:      "git diff format",
		description:  "Review diffs in git diff format",
		replOnly:     false,
		terminalOnly: false,
	},
	{
		char:         "r",
		command:      "reject",
		description:  "Reject some or all pending changes",
		replOnly:     false,
		terminalOnly: false,
	},

	{
		char:         "a",
		command:      "apply",
		description:  "Apply all pending changes",
		replOnly:     false,
		terminalOnly: false,
	},

	{
		char:            "f",
		command:         "full auto",
		description:     "Apply and debug in full auto mode",
		replOnly:        true,
		terminalOnly:    false,
		applyScriptOnly: true,
	},

	{
		char:         "q",
		key:          keyboard.KeyEnter,
		command:      "exit menu",
		description:  "Exit menu",
		dropdownOnly: true,
	},
}

func showHotkeyMenu(diffs []string) {
	hasApplyScript := false
	for _, diff := range diffs {
		if diff == "_apply.sh" {
			hasApplyScript = true
			break
		}
	}

	numDiffs := len(diffs)
	s := "files have"
	if numDiffs == 1 {
		s = "file has"
	}
	color.New(color.Bold, term.ColorHiGreen).Printf("🧐 %d %s pending changes\n", numDiffs, s)

	for _, diff := range diffs {
		if diff == "_apply.sh" {
			continue
		}

		fmt.Printf("• 📄 %s\n", diff)
	}
	fmt.Println()

	if hasApplyScript {
		color.New(color.Bold, term.ColorHiYellow).Println("🚀 Commands pending")
		fmt.Println()
	}

	var b strings.Builder
	table := tablewriter.NewWriter(&b)
	table.SetAutoWrapText(false)
	table.SetHeaderLine(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, opt := range allHotkeyOptions {
		if (opt.terminalOnly && term.IsRepl) || (opt.replOnly && !term.IsRepl) || opt.dropdownOnly || (opt.applyScriptOnly && !hasApplyScript) {
			continue
		}

		c := color.New(term.ColorHiCyan, color.Bold)
		if opt.command == "apply" {
			c = color.New(term.ColorHiGreen, color.Bold)
		} else if opt.command == "reject" {
			c = color.New(term.ColorHiRed, color.Bold)
		} else if opt.command == "full auto" {
			c = color.New(term.ColorHiYellow, color.Bold)
		}

		table.Append([]string{
			c.Sprintf("(%s)", opt.char),
			opt.command,
			opt.description,
		})
	}

	table.Render()
	fmt.Print(b.String())

	fmt.Printf("%s %s %s %s %s",
		color.New(term.ColorHiMagenta, color.Bold).Sprint("Press a hotkey,"),
		color.New(color.FgHiWhite, color.Bold).Sprintf("↓"),
		color.New(term.ColorHiMagenta, color.Bold).Sprintf("to select, or"),
		color.New(color.FgHiWhite, color.Bold).Sprintf("enter"),
		color.New(term.ColorHiMagenta, color.Bold).Sprintf("to exit menu/keep iterating>"),
	)
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

	handleHotkeyOption(hotkeyOption{char: string(char), key: key}, diffs, params)
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

	if option.char == "d" {
		fmt.Println()
		_, err := lib.ExecPlandexCommandWithParams([]string{"diffs", "--ui", "--from-tell-menu"}, lib.ExecPlandexCommandParams{
			DisableSuggestions: true,
		})
		if err != nil {
			fmt.Printf("\nError showing diffs: %v\n", err)
		}
		fmt.Println()
	} else if option.char == "g" {
		fmt.Println()
		_, err := lib.ExecPlandexCommandWithParams([]string{"diffs", "--git"}, lib.ExecPlandexCommandParams{
			DisableSuggestions: true,
		})
		if err != nil {
			fmt.Printf("\nError showing diffs: %v\n", err)
		}
		fmt.Println()
	} else if option.char == "a" {
		fmt.Print("(a)")
		fmt.Println()
		_, err := lib.ExecPlandexCommand([]string{"apply"})
		if err != nil {
			fmt.Printf("\nError applying changes: %v\n", err)
		}
		fmt.Println()
		os.Exit(0)
	} else if option.char == "r" {
		fmt.Println()
		_, err := lib.ExecPlandexCommand([]string{"reject"})
		if err != nil {
			fmt.Printf("\nError rejecting changes: %v\n", err)
		}
		fmt.Println()
		exitUnlessDiffs()
	} else if option.char == "f" {
		fmt.Print("(f)")
		fmt.Println()

		color.New(term.ColorHiYellow, color.Bold).Println("⚠️  Full auto mode allows automatic apply, execution, and multiple rounds of debugging without review.")
		fmt.Println()

		_, err := lib.ExecPlandexCommand([]string{"apply", "--full"})
		if err != nil {
			fmt.Printf("\nError applying changes: %v\n", err)
		}
		fmt.Println()
		os.Exit(0)
	} else if option.char == "q" || option.key == keyboard.KeyEnter {
		os.Exit(0)
	} else {
		fmt.Println("\nInvalid hotkey")
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
