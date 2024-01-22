package term

import (
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glow/utils"
)

func GetMarkdown(input string) (string, error) {
	// Default values
	style := "dark"

	// Removing potential frontmatter
	inputBytes := utils.RemoveFrontmatter([]byte(input))

	// Render
	out, err := glamour.RenderBytes(inputBytes, style)
	if err != nil {
		return "", err
	}

	return string(out), nil
}
