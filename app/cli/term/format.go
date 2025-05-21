package term

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glow/utils"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/termenv"
)

func init() {
	// pre-cache the glamour renderer
	getGlamourRenderer()
}

var (
	cachedGlamourRenderer      *glamour.TermRenderer
	cachedGlamourRendererWidth int
)

func getGlamourRenderer() (*glamour.TermRenderer, error) {
	width := GetTerminalWidth()

	if cachedGlamourRenderer != nil && cachedGlamourRendererWidth == width {
		return cachedGlamourRenderer, nil
	}

	// Build the renderer options.
	var opts []glamour.TermRendererOption

	// Check for a GLAMOUR_STYLE env variable.
	if style, ok := os.LookupEnv("GLAMOUR_STYLE"); ok && style != "" {
		opts = append(opts, glamour.WithStandardStyle(style))
	} else {
		// Fallback to auto style detection.
		opts = append(opts, glamour.WithAutoStyle())
	}

	// Always set word wrap and preserved newlines.
	opts = append(opts,
		glamour.WithWordWrap(width-2),
		glamour.WithPreservedNewLines(),
	)

	r, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create glamour renderer: %w", err)
	}

	cachedGlamourRenderer = r
	cachedGlamourRendererWidth = width
	return r, nil
}

func GetMarkdown(input string) (string, error) {
	inputBytes := utils.RemoveFrontmatter([]byte(input))

	r, err := getGlamourRenderer()
	if err != nil {
		return "", err
	}

	out, err := r.RenderBytes(inputBytes)
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func GetPlain(input string) string {
	width := GetTerminalWidth()

	s := wordwrap.String(input, width-2)
	// add padding
	lines := strings.Split(s, "\n")
	// for i := range lines {
	// 	lines[i] = "  " + lines[i]
	// }
	s = strings.Join(lines, "\n")

	s = termenv.String(s).Foreground(GetStreamForegroundColor()).String()

	return s
}
