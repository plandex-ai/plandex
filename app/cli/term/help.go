package term

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

type CmdConfig struct {
	Cmd   string
	Alias string
	Desc  string
	Repl  bool
}

var CliCommands = []CmdConfig{
	{"", "", "start the Plandex REPL", false},

	// {"--full", "", fmt.Sprintf("start the Plandex REPL with auto-mode %s", "'full'"), false},
	// {"--semi", "", fmt.Sprintf("start the Plandex REPL with auto-mode %s", "'semi'"), false},
	// {"--plus", "", fmt.Sprintf("start the Plandex REPL with auto-mode %s", "'plus'"), false},
	// {"--basic", "", fmt.Sprintf("start the Plandex REPL with auto-mode %s", "'basic'"), false},
	// {"--none", "", fmt.Sprintf("start the Plandex REPL with auto-mode %s", "'none'"), false},

	// {"--daily", "", fmt.Sprintf("start the Plandex REPL with %s model pack", "'daily-driver'"), false},
	// {"--strong", "", fmt.Sprintf("start the Plandex REPL with %s model pack", "'strong'"), false},
	// {"--cheap", "", fmt.Sprintf("start the Plandex REPL with %s model pack", "'cheap'"), false},
	// {"--oss", "", fmt.Sprintf("start the Plandex REPL with %s model pack", "'oss'"), false},

	{"new", "", "start a new plan", true},

	{"new --full", "", fmt.Sprintf("start a new plan with auto-mode %s", "'full'"), true},
	{"new --semi", "", fmt.Sprintf("start a new plan with auto-mode %s", "'semi'"), true},
	{"new --plus", "", fmt.Sprintf("start a new plan with auto-mode %s", "'plus'"), true},
	{"new --basic", "", fmt.Sprintf("start a new plan with auto-mode %s", "'basic'"), true},
	{"new --none", "", fmt.Sprintf("start a new plan with auto-mode %s", "'none'"), true},

	{"new --daily", "", fmt.Sprintf("start a new plan with %s model pack", "'daily-driver'"), true},
	{"new --reasoning", "", fmt.Sprintf("start a new plan with %s model pack", "'reasoning'"), true},
	{"new --strong", "", fmt.Sprintf("start a new plan with %s model pack", "'strong'"), true},
	{"new --cheap", "", fmt.Sprintf("start a new plan with %s model pack", "'cheap'"), true},
	{"new --oss", "", fmt.Sprintf("start a new plan with %s model pack", "'oss'"), true},
	{"new --gemini-preview", "", fmt.Sprintf("start a new plan with %s model pack", "'gemini-preview'"), true},
	// {"new --crazy", "", fmt.Sprintf("start a new plan with %s model pack", "'crazy'"), true},
	{"plans", "pl", "list plans", true},
	{"cd", "", "set current plan by name or index", true},
	{"current", "cu", "show current plan", true},
	{"rename", "", "rename the current plan", true},
	{"delete-plan", "dp", "delete plan by name or index", true},

	{"config", "", "show current plan config", true},
	{"set-config", "", "update current plan config", true},
	{"config default", "", "show the default config for new plans", true},
	{"set-config default", "", "update the default config for new plans", true},

	{"set-auto", "", "update auto-mode (autonomy level) for current plan", true},
	{"set-auto none", "", fmt.Sprintf("set auto-mode to %s", "'none'"), true},
	{"set-auto basic", "", fmt.Sprintf("set auto-mode to %s", "'basic'"), true},
	{"set-auto plus", "", fmt.Sprintf("set auto-mode to %s", "'plus'"), true},
	{"set-auto semi", "", fmt.Sprintf("set auto-mode to %s", "'semi'"), true},
	{"set-auto full", "", fmt.Sprintf("set auto-mode to %s", "'full'"), true},

	{"set-auto default", "", "set the default auto-mode for new plans", true},

	{"tell", "t", "describe a task to complete", false},
	{"chat", "ch", "ask a question or chat", false},

	{"load", "l", "load files/dirs/urls/notes/images or pipe data into context", true},
	{"ls", "", "list everything in context", true},
	{"rm", "", "remove context by index, range, name, or glob", true},
	{"clear", "", "remove all context", true},
	{"update", "u", "update outdated context", true},
	{"show", "", "show current context by name or index", true},

	{"diff --ui", "", "review pending changes in a browser UI", true},
	{"diff", "", "review pending changes in 'git diff' format", true},
	{"diff --plain", "", "review pending changes in 'git diff' format with no color formatting", false},
	{"summary", "", "show the latest summary of the current plan", true},

	{"apply", "ap", "apply pending changes to project files", true},
	{"reject", "rj", "reject pending changes to one or more project files", true},

	{"log", "", "show log of plan updates", true},
	{"rewind", "rw", "rewind to a previous state", true},

	{"continue", "c", "continue the plan", true},
	{"debug", "db", "repeatedly run a command and auto-apply fixes until it succeeds", true},
	{"build", "b", "build any pending changes", true},

	{"convo", "", "show plan conversation", true},
	{"convo 1", "", "show a specific message in the conversation", false},
	{"convo 2-5", "", "show a range of messages in the conversation", false},
	{"convo --plain", "", "show conversation in plain text", false},

	{"branches", "br", "list plan branches", true},
	{"checkout", "co", "checkout or create a branch", true},
	{"delete-branch", "dlb", "delete a branch by name or index", true},

	{"plans --archived", "", "list archived plans", true},
	{"archive", "arc", "archive a plan", true},
	{"unarchive", "unarc", "unarchive a plan", true},

	{"models", "", "show current plan model settings", true},
	{"models default", "", "show the default model settings for new plans", true},
	{"models available", "", "show all available models", true},
	{"models available --custom", "", "show available custom models only", true},
	{"models delete", "", "delete a custom model", true},
	{"models add", "", "add a custom model", true},
	{"model-packs", "", "show all available model packs", true},
	{"model-packs create", "", "create a new custom model pack", true},
	{"model-packs delete", "", "delete a custom model pack", true},
	{"model-packs --custom", "", "show custom model packs only", true},
	{"model-packs update", "", "update a custom model pack", true},
	{"model-packs show", "", "show a built-in or custom model pack's settings", true},
	{"set-model", "", "update current plan model settings", true},
	{"set-model default", "", "update the default model settings for new plans", true},

	{"set-model daily", "", fmt.Sprintf("Use %s model pack", "'daily-driver'"), true},
	{"set-model reasoning", "", fmt.Sprintf("Use %s model pack", "'reasoning'"), true},
	{"set-model strong", "", fmt.Sprintf("Use %s model pack", "'strong'"), true},
	{"set-model cheap", "", fmt.Sprintf("Use %s model pack", "'cheap'"), true},
	{"set-model oss", "", fmt.Sprintf("Use %s model pack", "'oss'"), true},
	{"set-model gemini-preview", "", fmt.Sprintf("Use %s model pack", "'gemini-preview'"), true},

	{"ps", "", "list active and recently finished plan streams", true},
	{"stop", "", "stop an active plan stream", true},
	{"connect", "conn", "connect to an active plan stream", true},

	{"sign-in", "", "sign in, accept an invite, or create an account", true},
	{"invite", "", "invite a user to join your org", true},
	{"revoke", "", "revoke an invite or remove a user from your org", true},
	{"users", "", "list users and pending invites in your org", true},

	{"usage", "", "show Plandex Cloud current balance and usage report", true},
	{"usage --today", "", "show Plandex Cloud usage for the day so far", true},
	{"usage --month", "", "show Plandex Cloud usage for the current billing month", true},
	{"usage --plan", "", "show Plandex Cloud usage for the current plan", true},

	{"usage --log", "", "show Plandex Cloud transaction log", true},

	{"billing", "", "show Plandex Cloud billing settings", true},
}

var CmdDesc = map[string]CmdConfig{}

func init() {
	for _, cmd := range CliCommands {
		CmdDesc[cmd.Cmd] = cmd
	}
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

		if IsRepl && !config.Repl {
			continue
		}

		alias := config.Alias
		desc := config.Desc

		if alias != "" {
			if IsRepl {
				cmd = fmt.Sprintf("%s (\\%s)", cmd, alias)
			} else {
				containsFull := strings.Contains(cmd, alias)

				if containsFull {
					cmd = strings.Replace(cmd, alias, fmt.Sprintf("(%s)", alias), 1)
				} else {
					cmd = fmt.Sprintf("%s (%s)", cmd, alias)
				}
			}

			// desc += color.New(color.FgWhite).Sprintf(" • alias → %s", fmt.Sprint(a'lias'))
		}

		var styled string
		if IsRepl {
			styled = color.New(colors...).Sprintf(" \\%s ", cmd)
		} else if cmd == "" { // special case for the repl
			styled = color.New(colors...).Sprintf(" plandex ")
		} else {
			styled = color.New(colors...).Sprintf(" plandex %s ", cmd)
		}

		fmt.Fprintf(w, "%s%s 👉 %s\n", prefix, styled, desc)
	}

}

func PrintCustomCmd(prefix, cmd, alias, desc string) {
	cmd = strings.Replace(cmd, alias, fmt.Sprintf("(%s)", alias), 1)
	// desc += color.New(color.FgWhite).Sprintf(" • alias → %s", fmt.Sprint(a'lias'))
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
	fmt.Fprintf(builder, " 🚀 Start the Plandex REPL in a project directory with %s or %s\n", color.New(color.Bold, color.BgCyan, color.FgHiWhite).Sprint(" plandex "), color.New(color.Bold, color.BgCyan, color.FgHiWhite).Sprint(" pdx "))
	fmt.Fprintln(builder)
	fmt.Fprintf(builder, " 💻 You can also use any command outside the REPL with %s or %s\n", color.New(color.Bold, color.BgCyan, color.FgHiWhite).Sprint(" plandex [command] "), color.New(color.Bold, color.BgCyan, color.FgHiWhite).Sprint(" pdx [command] "))
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgMagenta, color.FgHiWhite).Fprintln(builder, " REPL Options ")
	fmt.Fprintln(builder)
	// Add REPL startup flags
	fmt.Fprintln(builder, color.New(color.Bold, color.FgHiBlue).Sprint("  Mode "))
	fmt.Fprintln(builder, "    --chat, -c     Start in chat mode (for conversation without making changes)")
	fmt.Fprintln(builder, "    --tell, -t     Start in tell mode (for implementation)")
	fmt.Fprintln(builder)

	fmt.Fprintln(builder, color.New(color.Bold, color.FgHiBlue).Sprint("  Autonomy "))
	fmt.Fprintln(builder, "    --no-auto      None → step-by-step, no automation")
	fmt.Fprintln(builder, "    --basic        Basic → auto-continue plans")
	fmt.Fprintln(builder, "    --plus         Plus → auto-update context, smart context, auto-commit changes")
	fmt.Fprintln(builder, "    --semi         Semi-Auto → auto-load context")
	fmt.Fprintln(builder, "    --full         Full-Auto → auto-apply, auto-exec, auto-debug")
	fmt.Fprintln(builder)

	fmt.Fprintln(builder, color.New(color.Bold, color.FgHiBlue).Sprint("  Models "))
	fmt.Fprintln(builder, "    --daily        Daily driver pack")
	fmt.Fprintln(builder, "    --reasoning    Reasoning pack")
	fmt.Fprintln(builder, "    --strong       Strong pack")
	fmt.Fprintln(builder, "    --cheap        Cheap pack")
	fmt.Fprintln(builder, "    --oss          Open source pack")

	fmt.Fprintln(builder)

	if all {
		fmt.Print(builder.String())
		PrintHelpAllCommands()
	} else {
		fmt.Print(builder.String())
		// in the same style as 'getting started' section, output See All Commands

		color.New(color.Bold, color.BgHiBlue, color.FgHiWhite).Fprintln(builder, " Use 'plandex help --all' or 'plandex help -a' for a list of all commands ")
		fmt.Fprintln(builder)

		fmt.Print(builder.String())
	}

}

func PrintHelpAllCommands() {
	builder := &strings.Builder{}

	color.New(color.Bold, color.BgMagenta, color.FgHiWhite).Fprintln(builder, " Key Commands ")
	printCmds(builder, " ", []color.Attribute{color.Bold, color.FgHiMagenta}, "new", "load", "tell", "diff", "diff --ui", "apply", "reject", "debug", "chat")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Plans ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "new", "plans", "cd", "current", "delete-plan", "rename", "archive", "plans --archived", "unarchive")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Changes ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "diff", "diff --ui", "diff --plain", "apply", "reject")
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

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Config ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "config", "set-config", "config default", "set-config default")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Autonomy ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "set-auto", "set-auto default", "set-auto full", "set-auto semi", "set-auto plus", "set-auto basic", "set-auto none")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " AI Models ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "models", "models default", "model-packs", "set-model", "set-model daily", "set-model reasoning", "set-model strong", "set-model cheap", "set-model oss", "set-model default")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Custom Models ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "models available", "models available --custom", "models add", "models delete", "model-packs --custom", "model-packs create", "model-packs show", "model-packs update", "model-packs delete")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Accounts ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "sign-in", "invite", "revoke", "users")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " Cloud ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "usage", "usage --today", "usage --month", "usage --plan", "usage --log", "billing")
	fmt.Fprintln(builder)

	color.New(color.Bold, color.BgCyan, color.FgHiWhite).Fprintln(builder, " New Plan Shortcuts ")
	printCmds(builder, " ", []color.Attribute{color.Bold, ColorHiCyan}, "new --full", "new --semi", "new --plus", "new --basic", "new --none", "new --daily", "new --reasoning", "new --strong", "new --cheap", "new --oss", "new --gemini-preview" /*"new --crazy"*/)
	fmt.Fprintln(builder)

	fmt.Print(builder.String())
}
