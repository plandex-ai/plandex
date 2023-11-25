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
		anyApplied := false

		// Change entries
		for i, result := range results {
			for j, rep := range result.Replacements {
				flatIndex := i*len(result.Replacements) + j
				selected := currentRep != nil && rep.Id == currentRep.Id
				s := ""

				fgColor := color.FgHiGreen
				bgColor := color.BgGreen
				if rep.Failed {
					fgColor = color.FgHiRed
					bgColor = color.BgRed
					anyFailed = true
				} else if rep.RejectedAt != "" {
					fgColor = color.FgWhite
					bgColor = color.BgBlack
				}

				var icon string
				if rep.RejectedAt != "" {
					icon = "👎"
				} else if rep.Failed {
					icon = "🚫"
				} else {
					icon = "📝"
				}

				if !rep.Failed && rep.RejectedAt == "" {
					anyApplied = true
				}

				if selected {
					s += color.New(color.Bold, bgColor, color.FgHiWhite).Sprintf(" > %s %d ", icon, flatIndex+1)
				} else {
					s += color.New(fgColor).Sprintf(" - %s %d ", icon, flatIndex+1)
				}

				s += "\n"

				sb.WriteString(s)
			}

		}

		if !anyApplied {
			continue
		}

		fgColor := color.FgHiGreen
		bgColor := color.BgGreen
		if anyFailed {
			fgColor = color.FgHiRed
			bgColor = color.BgRed
		}

		if m.selectedFullFile() {
			sb.WriteString(color.New(color.Bold, bgColor, color.FgHiWhite).Sprint(" > 🔀   "))
		} else {
			sb.WriteString(color.New(fgColor).Sprint(" - 🔀   "))
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
