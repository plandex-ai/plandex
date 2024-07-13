package changes_tui

import (
	"plandex/term"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

func (m changesUIModel) renderPathTabs() string {
	if m.selectionInfo == nil || m.width == 0 {
		return ""
	}

	var tabRows [][]string

	lineWidth := 0
	lineNum := 0

	paths := m.currentPlan.PlanResult.SortedPaths

	for i, path := range paths {
		selected := i == m.selectedFileIndex

		if len(path) > 40 {
			path = path[:20] + "⋯" + path[len(path)-20:]
		}

		tab := " 📄 " + path + "  "

		pathColor := term.ColorHiGreen
		bgColor := color.BgGreen

		if selected {
			tab = color.New(color.Bold, bgColor, color.FgHiWhite).Sprint(tab)
		} else {
			tab = color.New(pathColor).Sprint(tab)
		}

		tabWidth := lipgloss.Width(tab)

		if tabWidth > m.width {
			return ""
		}

		if lineWidth+tabWidth > m.width {
			lineWidth = 0
			lineNum++
		}

		if len(tabRows) <= lineNum {
			tabRows = append(tabRows, []string{})
		}

		tabs := tabRows[lineNum]
		tabs = append(tabs, tab)
		tabRows[lineNum] = tabs

		lineWidth += tabWidth
	}

	resRows := make([]string, len(tabRows))

	for i, row := range tabRows {
		resRows[i] = strings.Join(row, "")
	}

	style := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).BorderForeground(lipgloss.Color(borderColor)).Width(m.width)
	tabs := strings.Join(resRows, "\n")
	return style.Render(tabs)
}
