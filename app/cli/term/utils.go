package term

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/term"
)

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
	terminalWidth := getTerminalWidth()
	return strings.Repeat("─", terminalWidth)
}

func getTerminalWidth() int {
	// Try to get terminal size
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		return w
	}

	// Use tput to get terminal width
	if w, err := getWidthFromTput(); err == nil {
		return w
	}

	// Try to get width from environment variable
	if w, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil {
		return w
	}

	// Fallback to default width
	return 50
}

func getWidthFromTput() (int, error) {
	cmd := exec.Command("tput", "cols")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	width, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, err
	}
	return width, nil
}
