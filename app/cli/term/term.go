package term

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/eiannone/keyboard"
	"github.com/fatih/color"
	"golang.org/x/term"

	"github.com/cqroot/prompt"
	"github.com/cqroot/prompt/input"
)

var CmdDesc = map[string][2]string{
	"new":     {"", "start a new plan"},
	"current": {"cu", "show current plan"},
	"cd":      {"", "set current plan by name or index"},
	"load":    {"l", "load files, dirs, urls, notes or piped data into context"},
	"tell":    {"t", "describe a task, ask a question, or chat"},
	// "diffs":       {"d", "show diffs between plan and project files"},
	// "preview":     {"pv", "preview the plan in a branch"},
	"apply":       {"ap", "apply the plan to your project files"},
	"continue":    {"c", "continue the plan"},
	"status":      {"s", "show status of the plan"},
	"rewind":      {"rw", "rewind to a previous step or state"},
	"ls":          {"", "list everything in context"},
	"rm":          {"", "remove context by name, index, or glob"},
	"clear":       {"", "remove all context"},
	"delete-plan": {"del", "delete plan by name or index"},
	"plans":       {"pl", "list plans"},
	"update":      {"u", "update outdated context"},
	"log":         {"", "show log of plan updates"},
	"branches":    {"br", "list plan branches"},
	// "checkout":    {"co", "checkout or create a branch"}, need to implement non-contiguous aliases
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
			cmd = strings.Replace(cmd, alias, fmt.Sprintf("(%s)", alias), 1)
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
		fmt.Fprintln(os.Stderr, "Failed to page output:", err)
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
		fmt.Fprintln(os.Stderr, "Failed to page output with colors:", err)
	}
}

func GetUserStringInput(msg string) (string, error) {
	return prompt.New().Ask(msg).Input("")
}

func GetUserPasswordInput(msg string) (string, error) {
	return prompt.New().Ask(msg).Input("", input.WithEchoMode(input.EchoPassword))
}

func GetUserKeyInput() (rune, error) {
	if err := keyboard.Open(); err != nil {
		return 0, fmt.Errorf("failed to open keyboard: %s", err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	char, _, err := keyboard.GetKey()
	if err != nil {
		return 0, fmt.Errorf("failed to read keypress: %s", err)
	}

	return char, nil
}

func ConfirmYesNo(fmtStr string, fmtArgs ...interface{}) (bool, error) {
	color.New(color.FgHiMagenta, color.Bold).Printf(fmtStr+" (y)es | (n)o", fmtArgs...)
	color.New(color.FgHiMagenta, color.Bold).Print("> ")

	char, err := GetUserKeyInput()
	if err != nil {
		return false, fmt.Errorf("failed to get user input: %s", err)
	}

	fmt.Println(string(char))
	if char == 'y' || char == 'Y' {
		return true, nil
	} else if char == 'n' || char == 'N' {
		return false, nil
	} else {
		fmt.Println()
		color.New(color.FgHiRed, color.Bold).Print("Invalid input.\nEnter 'y' for yes or 'n' for no.\n\n")
		return ConfirmYesNo(fmtStr, fmtArgs...)
	}
}

func ConfirmYesNoCancel(fmtStr string, fmtArgs ...interface{}) (bool, bool, error) {
	color.New(color.FgHiMagenta, color.Bold).Printf(fmtStr+" (y)es | (n)o | (c)ancel", fmtArgs...)
	color.New(color.FgHiMagenta, color.Bold).Print("> ")

	char, err := GetUserKeyInput()
	if err != nil {
		return false, false, fmt.Errorf("failed to get user input: %s", err)
	}

	fmt.Println(string(char))
	if char == 'y' || char == 'Y' {
		return true, false, nil
	} else if char == 'n' || char == 'N' {
		return false, false, nil
	} else if char == 'c' || char == 'C' {
		return false, true, nil
	} else {
		fmt.Println()
		color.New(color.FgHiRed, color.Bold).Print("Invalid input.\nEnter 'y' for yes, 'n' for no, or 'c' for cancel.\n\n")
		return ConfirmYesNoCancel(fmtStr, fmtArgs...)
	}
}

func GetDivisionLine() string {
	// Get the terminal width
	terminalWidth, err := getTerminalWidth()
	if err != nil {
		fmt.Println("Error fetching terminal size:", err)
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
