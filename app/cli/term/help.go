package term

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

var CmdDesc = map[string][2]string{
	"new":          {"", "start a new plan"},
	"rename":       {"", "rename the current plan"},
	"current":      {"cu", "show current plan"},
	"cd":           {"", "set current plan by name or index"},
	"load":         {"l", "load files/dirs/urls/notes/images or pipe data into context"},
	"tell":         {"t", "describe a task to complete"},
	"chat":         {"", "ask a question or chat"},
	"changes":      {"ch", "review pending changes in a TUI"},
	"diff":         {"", "review pending changes in 'git diff' format"},
	"diff --plain": {"", "review pending changes in 'git diff' format with no color formatting"},
	"diff --ui":    {"", "review pending changes in a local browser UI"},
	"summary":      {"", "show the latest summary of the current plan"},
	// "preview":     {"pv", "preview the plan in a branch"},
	"apply":     {"ap", "apply pending changes to project files"},
	"reject":    {"rj", "reject pending changes to one or more project files"},
	"archive":   {"arc", "archive a plan"},
	"unarchive": {"unarc", "unarchive a plan"},
	"continue":  {"c", "continue the plan"},
	"debug":     {"db", "repeatedly run a command and auto-apply fixes until it succeeds"},
	// "status":      {"s", "show status of the plan"},
	"rewind":                    {"rw", "rewind to a previous state"},
	"ls":                        {"", "list everything in context"},
	"rm":                        {"", "remove context by index, range, name, or glob"},
	"clear":                     {"", "remove all context"},
	"delete-plan":               {"dp", "delete plan by name or index"},
	"delete-branch":             {"db", "delete a branch by name or index"},
	"plans":                     {"pl", "list plans"},
	"plans --archived":          {"", "list archived plans"},
	"update":                    {"u", "update outdated context"},
	"log":                       {"", "show log of plan updates"},
	"convo":                     {"", "show plan conversation"},
	"convo 1":                   {"", "show a specific message in the conversation"},
	"convo 2-5":                 {"", "show a range of messages in the conversation"},
	"convo --plain":             {"", "show conversation in plain text"},
	"branches":                  {"br", "list plan branches"},
	"checkout":                  {"co", "checkout or create a branch"},
	"build":                     {"b", "build any pending changes"},
	"models":                    {"", "show current plan model settings"},
	"models default":            {"", "show org-wide default model settings for new plans"},
	"models available":          {"", "show all available models"},
	"models available --custom": {"", "show available custom models only"},
	"models delete":             {"", "delete a custom model"},
	"models add":                {"", "add a custom model"},
	"model-packs":               {"", "show all available model packs"},
	"model-packs create":        {"", "create a new custom model pack"},
	"model-packs delete":        {"", "delete a custom model pack"},
	"model-packs --custom":      {"", "show custom model packs only"},
	"set-model":                 {"", "update current plan model settings"},
	"set-model default":         {"", "update org-wide default model settings for new plans"},
	"ps":                        {"", "list active and recently finished plan streams"},
	"stop":                      {"", "stop an active plan stream"},
	"connect":                   {"conn", "connect to an active plan stream"},
	"sign-in":                   {"", "sign in, accept an invite, or create an account"},
	"invite":                    {"", "invite a user to join your org"},
	"revoke":                    {"", "revoke an invite or remove a user from your org"},
	"users":                     {"", "list users and pending invites in your org"},
	"credits":                   {"", "show Plandex Cloud credits balance"},
	"credits log":               {"", "show Plandex Cloud credits transaction log"},
	"billing":                   {"", "show Plandex Cloud billing settings"},
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
func PrintCustomHelp(all bool) {
	builder := &strings.Builder{}

	color.New(color.Bold, color.BgGreen, color.FgHiWhite).Fprintln(builder, " Usage ")
	color.New(color.Bold).Fprintln(builder, "  plandex [command] [flags]")
	color.New(color.Bold).Fprintln(builder, "  pdx [command] [flags]")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgGreen, color.FgHiWhite).Fprintln(builder, " Help ")
	color.New(color.Bold).Fprintln(builder, "  plandex help # show basic usage")
	color.New(color.Bold).Fprintln(builder, "  plandex help --all # show all commands")
	color.New(color.Bold).Fprintln(builder, "  plandex [command] --help")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgMagenta, color.FgHiWhite).Fprintln(builder, " Getting Started ")
	fmt.Fprintln(builder)
	fmt.Fprintf(builder, "  1 - Create a new plan in your project's root directory with %s\n", color.New(color.Bold, color.BgCyan, color.FgHiWhite).Sprint(" plandex new "))
	fmt.Fprintln(builder)
	fmt.Fprintf(builder, "  2 - Load any relevant context with %s\n", color.New(color.Bold, color.BgCyan, color.FgHiWhite).Sprint(" plandex load [file-path-or-url] "))
	fmt.Fprintln(builder)
	fmt.Fprintf(builder, "  3 - Describe a task to complete with %s\n", color.New(color.Bold, color.BgCyan, color.FgHiWhite).Sprint(" plandex tell "))
	fmt.Fprintln(builder)

	if all {
		color.New(color.Bold, color.BgMagenta, color.FgHiWhite).Fprintln(builder, " Key Commands ")
		printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiMagenta}, "new", "load", "tell", "diff", "diff --ui", "apply", "reject", "debug", "chat")
		fmt.Fprintln(builder)

		color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Plans ")
		printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "new", "plans", "cd", "current", "delete-plan", "rename", "archive", "plans --archived", "unarchive")
		fmt.Fprintln(builder)

		color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Changes ")
		printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "diff", "diff --ui", "diff --plain", "changes", "apply", "reject")
		fmt.Fprintln(builder)

		color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Context ")
		printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "load", "ls", "rm", "update", "clear")
		fmt.Fprintln(builder)

		color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Branches ")
		printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "branches", "checkout", "delete-branch")
		fmt.Fprintln(builder)

		color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " History ")
		printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "log", "rewind", "convo", "convo 1", "convo 2-5", "convo --plain", "summary")
		fmt.Fprintln(builder)

		color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Control ")
		printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "tell", "continue", "build", "debug", "chat")
		fmt.Fprintln(builder)

		color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Streams ")
		printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "ps", "connect", "stop")
		fmt.Fprintln(builder)

		color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " AI Models ")
		printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "models", "models default", "models available", "set-model", "set-model default", "models available --custom", "models add", "models delete", "model-packs", "model-packs --custom", "model-packs create", "model-packs delete")
		fmt.Fprintln(builder)

		color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Accounts ")
		printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "sign-in", "invite", "revoke", "users")
		fmt.Fprintln(builder)

		color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Cloud ")
		printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "credits", "credits log", "billing")

	} else {

		// in the same style as 'getting started' section, output See All Commands

		color.New(color.Bold, color.BgHiBlue, color.FgHiWhite).Fprintln(builder, " Use 'plandex help --all' or 'plandex help -a' for a list of all commands ")

	}

	fmt.Print(builder.String())
}
