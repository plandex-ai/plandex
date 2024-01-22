package changes_tui

import (
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

		if len(path) > 20 {
			path = path[:10] + "⋯" + path[len(path)-10:]
		}

		tab := " 📄 " + path + "  "

		results := m.currentPlan.PlanResult.FileResultsByPath[path]
		pathColor := color.FgHiGreen
		bgColor := color.BgGreen
		anyFailed := false
		for _, result := range results {
			if result.AnyFailed {
				anyFailed = true
				break
			}
		}
		if anyFailed {
			pathColor = color.FgHiRed
			bgColor = color.BgRed
		}

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
