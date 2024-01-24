package streamtui

import (
	"fmt"
	"sort"
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
	if m.starting || m.processing {
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

	filePaths := make([]string, 0, len(m.tokensByPath))
	for filePath := range m.tokensByPath {
		filePaths = append(filePaths, filePath)
	}

	sort.Strings(filePaths)

	var rows [][]string
	lineWidth := 0
	lineNum := 0

	for _, filePath := range filePaths {
		tokens := m.tokensByPath[filePath]
		finished := m.finishedByPath[filePath]
		block := fmt.Sprintf("📄 %s", filePath)

		if finished {
			block += " ✅"
		} else if tokens > 0 {
			block += fmt.Sprintf(" %d 🪙", tokens)
		}

		blockWidth := lipgloss.Width(block)

		if lineWidth+blockWidth > m.width {
			lineWidth = 0
			lineNum++
		}

		if len(rows) <= lineNum {
			rows = append(rows, []string{})
		}

		row := rows[lineNum]
		row = append(row, block)
		rows[lineNum] = row

		lineWidth += blockWidth
	}

	resRows := make([]string, len(rows)+1)

	resRows[0] = head
	for i, row := range rows {
		resRows[i+1] = strings.Join(row, " | ")
	}

	return style.Render(strings.Join(resRows, "\n"))
}
