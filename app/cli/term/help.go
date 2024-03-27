package term

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

var CmdDesc = map[string][2]string{
	"new":     {"", "start a new plan"},
	"current": {"cu", "show current plan"},
	"cd":      {"", "set current plan by name or index"},
	"load":    {"l", "load files, dirs, urls, notes or piped data into context"},
	"tell":    {"t", "describe a task, ask a question, or chat"},
	"changes": {"ch", "review plan changes"},
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
	"sign-in":       {"", "sign in, accept an invite, or create an account"},
}

func PrintCmds(prefix string, cmds ...string) {
	printCmds(os.Stderr, prefix, []color.Attribute{color.Bold, color.FgHiWhite, color.BgCyan, color.FgHiWhite}, cmds...)
}

func PrintCmdsWithColors(prefix string, colors []color.Attribute, cmds ...string) {
	printCmds(os.Stderr, prefix, colors, cmds...)
}

func printCmds(w io.Writer, prefix string, colors []color.Attribute, cmds ...string) {
	if os.Getenv("PLANDEX_DISABLE_SUGGESTIONS") != "" {
		return
	}
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

		fmt.Fprintf(w, "%s%s 👉 %s\n", prefix, styled, desc)
	}

}

func PrintCustomCmd(prefix, cmd, alias, desc string) {
	cmd = strings.Replace(cmd, alias, fmt.Sprintf("(%s)", alias), 1)
	// desc += color.New(color.FgWhite).Sprintf(" • alias → %s", color.New(color.Bold).Sprint(alias))
	styled := color.New(color.Bold, color.FgHiWhite, color.BgCyan, color.FgHiWhite).Sprintf(" plandex %s ", cmd)
	fmt.Printf("%s%s 👉 %s\n", prefix, styled, desc)
}

// PrintCustomHelp prints the custom help output for the Plandex CLI
func PrintCustomHelp() {
	builder := &strings.Builder{}

	color.New(color.Bold, color.BgGreen, color.FgHiWhite).Fprintln(builder, " Usage ")
	color.New(color.Bold).Fprintln(builder, "  plandex [command] [flags]")
	color.New(color.Bold).Fprintln(builder, "  pdx [command] [flags]")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgGreen, color.FgHiWhite).Fprintln(builder, " Help ")
	color.New(color.Bold).Fprintln(builder, "  plandex help")
	color.New(color.Bold).Fprintln(builder, "  plandex [command] --help")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgMagenta, color.FgHiWhite).Fprintln(builder, " Getting Started ")
	fmt.Fprintf(builder, "  Create a new plan in your project's root directory with %s\n\n", color.New(color.Bold, color.BgCyan, color.FgHiWhite).Sprint(" plandex new "))

	color.New(color.Bold, color.BgMagenta, color.FgHiWhite).Fprintln(builder, " Key Commands ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiMagenta}, "new", "load", "tell", "changes", "apply")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Plans ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "new", "plans", "cd", "current", "delete-plan")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Changes ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "changes", "apply")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Context ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "load", "ls", "rm", "update", "clear")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Branches ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "branches", "checkout", "delete-branch")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " History ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "convo", "log", "rewind")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Control ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "tell", "continue", "build")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Streams ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "ps", "connect", "stop")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " AI Models ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "models", "set-model")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Accounts ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "sign-in")
	fmt.Fprintln(builder)

	fmt.Print(builder.String())
}
