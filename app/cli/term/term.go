package term

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/term"
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
	"apply":    {"ap", "apply the plan to your project files"},
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
		styled := color.New(color.Bold, color.FgHiWhite, color.BgCyan).Sprintf(" plandex %s ", cmd)

		fmt.Printf("%s%s 👉 %s\n", prefix, styled, desc)
	}
}

func PrintCustomCmd(prefix, cmd, alias, desc string) {
	cmd = strings.Replace(cmd, alias, fmt.Sprintf("(%s)", alias), 1)
	// desc += color.New(color.FgWhite).Sprintf(" • alias → %s", color.New(color.Bold).Sprint(alias))
	styled := color.New(color.Bold, color.FgHiWhite, color.BgCyan).Sprintf(" plandex %s ", cmd)
	fmt.Printf("%s%s 👉 %s\n", prefix, styled, desc)
}

func AlternateScreen() {
	// Switch to alternate screen and hide the cursor
	fmt.Print("\x1b[?1049h\x1b[?25l")
}

func ClearScreen() {
	fmt.Print("\x1b[2J")
}

func MoveCursorToTopLeft() {
	fmt.Print("\x1b[H")
}

func ClearCurrentLine() {
	fmt.Print("\033[2K")
}

func MoveUpLines(numLines int) {
	fmt.Printf("\033[%dA", numLines)
}

func BackToMain() {
	// Switch back to main screen and show the cursor on exit
	fmt.Print("\x1b[?1049l\x1b[?25h")
}

func PageOutput(output string) {
	cmd := exec.Command("less", "-R")
	cmd.Env = append(os.Environ(), "LESS=FRX", "LESSCHARSET=utf-8")
	cmd.Stdin = strings.NewReader(output)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		OutputErrorAndExit("Failed to page output: %v", err)
	}
}

func PageOutputReverse(output string) {
	cmd := exec.Command("less", "-RX", "+G")
	cmd.Stdin = strings.NewReader(output)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set the environment variables specifically for the less command
	cmd.Env = append(os.Environ(), "LESS=FRX", "LESSCHARSET=utf-8")

	if err := cmd.Run(); err != nil {
		OutputErrorAndExit("Failed to page output: %v", err)
	}
}

func GetDivisionLine() string {
	// Get the terminal width
	terminalWidth, err := getTerminalWidth()
	if err != nil {
		log.Println("Error fetching terminal size:", err)
		terminalWidth = 50 // default width if unable to fetch width
	}
	return strings.Repeat("─", terminalWidth)
}

func getTerminalWidth() (int, error) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0, err
	}
	return width, nil
}
