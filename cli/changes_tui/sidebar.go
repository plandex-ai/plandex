package changes_tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

func (m changesUIModel) renderSidebar() string {
	paths := m.currentPlan.SortedPaths

	if m.selectionInfo == nil {
		return ""
	}

	currentRep := m.selectionInfo.currentRep

	var sb strings.Builder

	for _, path := range paths {
		results := m.currentPlan.PlanResByPath[path]
		anyFailed := false

		// Change entries
		for i, result := range results {
			for j, rep := range result.Replacements {
				flatIndex := i*len(result.Replacements) + j
				selected := currentRep != nil && rep.Id == currentRep.Id
				s := ""

				labelColor := color.FgHiGreen
				if rep.Failed {
					labelColor = color.FgHiRed
					anyFailed = true
				}

				if selected {
					s += color.New(color.Bold, labelColor).Sprintf(" > ") + color.New(color.Bold, labelColor).Sprintf("%d", flatIndex)
				} else {
					s += color.New(labelColor).Sprintf(" - %d", flatIndex)
				}

				if result.RejectedAt != "" {
					s += " 🚫"
				}

				s += "\n"

				sb.WriteString(s)
			}

		}

		labelColor := color.FgHiGreen
		if anyFailed {
			labelColor = color.FgHiRed
		}

		if m.selectedFullFile() {
			sb.WriteString(color.New(color.Bold, labelColor).Sprint(" > 📄 \n "))
		} else {
			sb.WriteString(color.New(labelColor).Sprint(" - 📄 \n"))
		}
	}

	helpHeight := lipgloss.Height(m.renderHelp())
	tabsHeight := lipgloss.Height(m.renderPathTabs())
	sidebar := sb.String()

	style := lipgloss.NewStyle().
		Height(m.height - (helpHeight + tabsHeight)).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(borderColor)

	return style.Render(sidebar)
}
