package streamtui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

var borderColor = lipgloss.Color("#444")
var helpTextColor = lipgloss.Color("#ddd")

func (m streamUIModel) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderMainView(),
		m.renderProcessing(),
		m.renderBuild(),
		m.renderHelp(),
	)
}

func (m streamUIModel) renderMainView() string {
	return m.replyViewport.View()
}

func (m streamUIModel) renderHelp() string {
	style := lipgloss.NewStyle().Width(m.width).Foreground(lipgloss.Color(helpTextColor)).BorderStyle(lipgloss.NormalBorder()).BorderTop(true).BorderForeground(lipgloss.Color(borderColor))

	return style.Render(" (s)top")
}

func (m streamUIModel) renderProcessing() string {
	if m.processing {
		style := lipgloss.NewStyle().Width(m.width).BorderStyle(lipgloss.NormalBorder()).BorderTop(true).BorderForeground(lipgloss.Color(borderColor))

		return style.Render(m.spinner.View())
	} else {
		return ""
	}
}

func (m streamUIModel) renderBuild() string {
	if !m.building {
		return ""
	}

	style := lipgloss.NewStyle().Width(m.width).BorderStyle(lipgloss.NormalBorder()).BorderTop(true).BorderForeground(lipgloss.Color(borderColor))

	head := color.New(color.BgGreen, color.FgHiWhite, color.Bold).Sprint(" 🏗  ") + color.New(color.BgGreen, color.FgHiWhite).Sprint("Building plan ")

	var lines []string
	lines = append(lines, head)
	for filePath, tokens := range m.tokensByPath {
		finished := m.finishedByPath[filePath]
		line := fmt.Sprintf("  📄 %s", filePath)
		if tokens > 0 {
			line += fmt.Sprintf(" | %d 🪙", tokens)
		}
		if finished {
			line += " | ✅"
		}
		lines = append(lines, line)
	}

	return style.Render(strings.Join(lines, "\n"))
}
