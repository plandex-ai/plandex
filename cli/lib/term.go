package lib

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/term"
)

var cmdDesc = map[string][2]string{
	"new":     {"", "start a new plan"},
	"load":    {"l", "load files/urls or pipe data into context"},
	"tell":    {"t", "describe a task, ask a question, or chat"},
	"diffs":   {"d", "show diffs between plan and project files"},
	"preview": {"p", "preview the plan in a branch"},
	"apply":   {"a", "apply the plan to your project files"},
	"next":    {"n", "continue to next step of the plan"},
	"status":  {"s", "show status of the plan"},
	"rewind":  {"r", "rewind the plan to a previous step or state"},
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

func clearScreen() {
	fmt.Print("\x1b[2J")
}

func moveCursorToTopLeft() {
	fmt.Print("\x1b[H")
}

func clearCurrentLine() {
	fmt.Print("\033[2K")
}

func moveUpLines(numLines int) {
	fmt.Printf("\033[%dA", numLines)
}

func backToMain() {
	// Switch back to main screen and show the cursor on exit
	fmt.Print("\x1b[?1049l\x1b[?25h")
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

func displayHotkeys() string {
	// Get the terminal width
	terminalWidth, err := getTerminalWidth()
	if err != nil {
		fmt.Println("Error fetching terminal size:", err)
		terminalWidth = 50 // default width if unable to fetch width
	}
	// Creating the terminal width long division line
	divisionLine := strings.Repeat("─", terminalWidth)

	return divisionLine + "\n" +
		"  \x1b[1m(s)\x1b[0m" + `top  
` + divisionLine
}
