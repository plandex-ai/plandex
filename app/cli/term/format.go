package term

import (
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glow/utils"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

func GetMarkdown(input string) (string, error) {
	width, _, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}

	inputBytes := utils.RemoveFrontmatter([]byte(input))

	r, _ := glamour.NewTermRenderer(
		// detect background color and pick either the default dark or light theme
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(min(width, 80)),
	)

	out, err := r.RenderBytes(inputBytes)
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func GetPlain(input string) (string, error) {
	width, _, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}

	s := wordwrap.String(input, min(width-2, 80))

	// add padding
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = "  " + lines[i]
	}
	s = strings.Join(lines, "\n")

	c := "234"
	if termenv.HasDarkBackground() {
		c = "251"
	}

	return termenv.String(s).Foreground(termenv.ANSI256.Color(c)).String(), nil
}
