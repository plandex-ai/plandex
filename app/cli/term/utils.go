package term

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/muesli/termenv"
	"golang.org/x/term"
)

func init() {
	// pre-cache terminal settings
	IsTerminal()
	GetTerminalWidth()
	GetStreamForegroundColor()
	HasDarkBackground()
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
	terminalWidth := GetTerminalWidth()
	return strings.Repeat("─", terminalWidth)
}

var envReplCols int
var envDefaultCols int

func GetTerminalWidth() int {
	if envReplCols != 0 {
		return envReplCols
	}

	if os.Getenv("PLANDEX_COLUMNS") != "" {
		w, err := strconv.Atoi(os.Getenv("PLANDEX_COLUMNS"))
		if err == nil {
			envReplCols = w
			return w
		}
	}

	if IsTerminal() {
		// Try to get terminal size
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
			return w
		}
	}

	if envDefaultCols != 0 {
		return envDefaultCols
	}

	// Not running in a TTY or GetSize failed; use a default.
	// Try to get width from environment variable
	if w, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil {
		envDefaultCols = w
		return w
	}

	// Fallback to default width
	return 80
}

var envStreamForegroundColor termenv.Color

func GetStreamForegroundColor() termenv.Color {
	if envStreamForegroundColor != nil {
		return envStreamForegroundColor
	}

	if os.Getenv("PLANDEX_STREAM_FOREGROUND_COLOR") != "" {
		envStreamForegroundColor = termenv.ANSI256.Color(os.Getenv("PLANDEX_STREAM_FOREGROUND_COLOR"))
		return envStreamForegroundColor
	}

	c := "234"
	if HasDarkBackground() {
		c = "251"
	}
	envStreamForegroundColor = termenv.ANSI256.Color(c)
	return envStreamForegroundColor
}

var envHasDarkBackground bool
var cachedHasDarkBackground bool

func HasDarkBackground() bool {
	if cachedHasDarkBackground {
		return envHasDarkBackground
	}
	envHasDarkBackground = termenv.HasDarkBackground()
	cachedHasDarkBackground = true
	return envHasDarkBackground
}

var envIsTerminal bool
var cachedIsTerminal bool

func IsTerminal() bool {
	if cachedIsTerminal {
		return envIsTerminal
	}
	envIsTerminal = term.IsTerminal(int(os.Stdout.Fd()))
	cachedIsTerminal = true
	return envIsTerminal
}
