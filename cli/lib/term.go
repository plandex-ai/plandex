package lib

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/eiannone/keyboard"
	"github.com/fatih/color"
	"golang.org/x/term"
)

var cmdDesc = map[string][2]string{
	"new":         {"", "start a new plan"},
	"current":     {"c", "show current plan"},
	"cd":          {"", "set current plan by name or index"},
	"load":        {"l", "load files/directories/urls/strings or pipe data into context"},
	"tell":        {"t", "describe a task, ask a question, or chat"},
	"diffs":       {"d", "show diffs between plan and project files"},
	"preview":     {"p", "preview the plan in a branch"},
	"apply":       {"ap", "apply the plan to your project files"},
	"next":        {"n", "continue to next step of the plan"},
	"status":      {"s", "show status of the plan"},
	"rewind":      {"rw", "rewind the plan to a previous step or state"},
	"ls":          {"", "list everything in context"},
	"rm":          {"", "remove context by name, index, or glob"},
	"clear":       {"", "remove all context"},
	"delete-plan": {"del", "delete plan by name or index"},
	"plans":       {"", "list plans"},
	"update":      {"u", "update outdated context"},
}

func PrintCmds(prefix string, cmds ...string) {
	for _, cmd := range cmds {
		config, ok := cmdDesc[cmd]
		if !ok {
			continue
		}

		alias := config[0]
		desc := config[1]
		if alias != "" {
			cmd = strings.Replace(cmd, alias, fmt.Sprintf("(%s)", alias), 1)
		}
		styled := color.New(color.Bold, color.FgHiWhite, color.BgCyan).Sprintf(" plandex %s ", cmd)

		fmt.Printf("%s%s 👉 %s\n", prefix, styled, desc)
	}
}

func PrintCustomCmd(prefix, cmd, alias, desc string) {
	cmd = strings.Replace(cmd, alias, fmt.Sprintf("(%s)", alias), 1)
	styled := color.New(color.Bold, color.FgHiWhite, color.BgCyan).Sprintf(" plandex %s ", cmd)
	fmt.Printf("%s%s 👉 %s\n", prefix, styled, desc)
}

func alternateScreen() {
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

// Function for 'a' key action
func handleAbortKey(proposalId string) error {
	return Abort(proposalId)
}

// // Function for 'r' key action
// func handleReviseKey(proposalId string) error {
// 	// Terminate current operation
// 	err := Api.Abort(proposalId)
// 	if err != nil {
// 		return err
// 	}

// 	// Prompt the user for new message
// 	fmt.Println(">\"")
// 	reader := bufio.NewReader(os.Stdin)
// 	newMessage, _ := reader.ReadString('"')

// 	// Propose the new message
// 	err = Propose(newMessage)
// 	if err != nil {
// 		return err
// 	}

// 	fmt.Println("Revision proposed.")
// 	return nil
// }

func handleKeyPress(input rune, proposalId string) error {
	switch input {
	case 's':
		return handleAbortKey(proposalId)
	// case 'r':
	// 	return handleReviseKey(proposalId)
	default:
		return fmt.Errorf("invalid key pressed: %s", string(input))
	}
}

func getTerminalWidth() (int, error) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0, err
	}
	return width, nil
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

func displayHotkeys() string {
	divisionLine := GetDivisionLine()

	return divisionLine + "\n" +
		"  \x1b[1m(s)\x1b[0m" + `top  
` + divisionLine
}

func getUserInput() (rune, error) {
	if err := keyboard.Open(); err != nil {
		return 0, fmt.Errorf("failed to open keyboard: %s\n", err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	char, _, err := keyboard.GetKey()
	if err != nil {
		return 0, fmt.Errorf("failed to read keypress: %s\n", err)
	}

	return char, nil
}
