package term

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

var CmdDesc = map[string][2]string{
	"new":     {"", "start a new plan"},
	"current": {"cu", "show current plan"},
	"cd":      {"", "set current plan by name or index"},
	"load":    {"l", "load files, dirs, urls, notes or piped data into context"},
	"tell":    {"t", "describe a task, ask a question, or chat"},
	"changes": {"ch", "show plan changes"},
	// "diffs":       {"d", "show diffs between plan and project files"},
	// "preview":     {"pv", "preview the plan in a branch"},
	"apply":    {"ap", "apply plan changes to project files"},
	"continue": {"c", "continue the plan"},
	// "status":      {"s", "show status of the plan"},
	"rewind":        {"rw", "rewind to a previous state"},
	"ls":            {"", "list everything in context"},
	"rm":            {"", "remove context by name, index, or glob"},
	"clear":         {"", "remove all context"},
	"delete-plan":   {"dp", "delete plan by name or index"},
	"delete-branch": {"db", "delete a branch by name or index"},
	"plans":         {"pl", "list plans"},
	"update":        {"u", "update outdated context"},
	"log":           {"", "show log of plan updates"},
	"convo":         {"", "show plan conversation"},
	"branches":      {"br", "list plan branches"},
	"checkout":      {"co", "checkout or create a branch"},
	"build":         {"b", "build any pending changes"},
	"models":        {"", "show model settings"},
	"set-model":     {"", "update model settings"},
	"ps":            {"", "list active and recently finished plan streams"},
	"stop":          {"", "stop an active plan stream"},
	"connect":       {"conn", "connect to an active plan stream"},
}

func PrintCmds(prefix string, cmds ...string) {
	printCmds(prefix, []color.Attribute{color.Bold, color.FgHiWhite, color.BgCyan}, cmds...)
}

func PrintCmdsWithColors(prefix string, colors []color.Attribute, cmds ...string) {
	printCmds(prefix, colors, cmds...)
}

func printCmds(prefix string, colors []color.Attribute, cmds ...string) {
	for _, cmd := range cmds {
		config, ok := CmdDesc[cmd]
		if !ok {
			continue
		}

		alias := config[0]
		desc := config[1]
		if alias != "" {
			containsFull := strings.Contains(cmd, alias)

			if containsFull {
				cmd = strings.Replace(cmd, alias, fmt.Sprintf("(%s)", alias), 1)
			} else {
				cmd = fmt.Sprintf("%s (%s)", cmd, alias)
			}

			// desc += color.New(color.FgWhite).Sprintf(" • alias → %s", color.New(color.Bold).Sprint(alias))
		}
		styled := color.New(colors...).Sprintf(" plandex %s ", cmd)

		fmt.Printf("%s%s 👉 %s\n", prefix, styled, desc)
	}

}

func PrintCustomCmd(prefix, cmd, alias, desc string) {
	cmd = strings.Replace(cmd, alias, fmt.Sprintf("(%s)", alias), 1)
	// desc += color.New(color.FgWhite).Sprintf(" • alias → %s", color.New(color.Bold).Sprint(alias))
	styled := color.New(color.Bold, color.FgHiWhite, color.BgCyan).Sprintf(" plandex %s ", cmd)
	fmt.Printf("%s%s 👉 %s\n", prefix, styled, desc)
}

// PrintCustomHelp prints the custom help output for the Plandex CLI
func PrintCustomHelp() {
	color.New(color.Bold, color.BgGreen).Println(" Usage ")
	color.New(color.Bold).Println("  plandex [command] [flags]")
	color.New(color.Bold).Println("  pdx [command] [flags]")
	fmt.Println()

	color.New(color.Bold, color.BgGreen).Println(" Help ")
	color.New(color.Bold).Println("  plandex help")
	color.New(color.Bold).Println("  plandex [command] --help")
	fmt.Println()

	color.New(color.Bold, color.BgMagenta).Println(" Getting Started ")
	PrintCmdsWithColors(" ", []color.Attribute{color.Bold}, "new", "load", "tell", "changes", "apply")
	fmt.Println()

	color.New(color.Bold, color.BgBlue).Println(" Plans ")
	PrintCmdsWithColors(" ", []color.Attribute{color.Bold}, "new", "plans", "cd", "current", "delete-plan")
	fmt.Println()

	color.New(color.Bold, color.BgBlue).Println(" Changes ")
	PrintCmdsWithColors(" ", []color.Attribute{color.Bold}, "changes", "apply")
	fmt.Println()

	color.New(color.Bold, color.BgBlue).Println(" Context ")
	PrintCmdsWithColors(" ", []color.Attribute{color.Bold}, "load", "ls", "rm", "update", "clear")
	fmt.Println()

	color.New(color.Bold, color.BgBlue).Println(" Branches ")
	PrintCmdsWithColors(" ", []color.Attribute{color.Bold}, "branches", "checkout", "delete-branch")
	fmt.Println()

	color.New(color.Bold, color.BgBlue).Println(" History ")
	PrintCmdsWithColors(" ", []color.Attribute{color.Bold}, "convo", "log", "rewind")
	fmt.Println()

	color.New(color.Bold, color.BgBlue).Println(" Execution ")
	PrintCmdsWithColors(" ", []color.Attribute{color.Bold}, "tell", "continue", "build")
	fmt.Println()

	color.New(color.Bold, color.BgBlue).Println(" Active Plans ")
	PrintCmdsWithColors(" ", []color.Attribute{color.Bold}, "ps", "connect", "stop")
	fmt.Println()

	color.New(color.Bold, color.BgBlue).Println(" AI Models ")
	PrintCmdsWithColors(" ", []color.Attribute{color.Bold}, "models", "set-model")
}
