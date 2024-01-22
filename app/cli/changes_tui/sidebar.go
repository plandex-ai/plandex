package changes_tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
)

func (m changesUIModel) renderSidebar() string {
	if m.selectionInfo == nil {
		return ""
	}

	currentRep := m.selectionInfo.currentRep

	var sb strings.Builder
	path := m.selectionInfo.currentPath

	results := m.currentPlan.PlanResult.FileResultsByPath[path]
	anyFailed := false
	anyApplied := false
	anyReplacements := false

	var replacements []*shared.Replacement

	for _, result := range results {
		replacements = append(replacements, result.Replacements...)
	}

	// Change entries
	for i, rep := range replacements {
		anyReplacements = true
		selected := currentRep != nil && rep.Id == currentRep.Id
		s := ""

		fgColor := color.FgHiGreen
		bgColor := color.BgGreen
		if rep.Failed {
			fgColor = color.FgHiRed
			bgColor = color.BgRed
			anyFailed = true
		} else if rep.RejectedAt != nil {
			fgColor = color.FgWhite
			bgColor = color.BgBlack
		}

		var icon string
		if rep.RejectedAt != nil {
			icon = "👎"
		} else if rep.Failed {
			icon = "🚫"
		} else {
			icon = "📝"
		}

		if !rep.Failed && rep.RejectedAt == nil {
			anyApplied = true
		}

		if selected {
			s += color.New(color.Bold, bgColor, color.FgHiWhite).Sprintf(" > %s %d ", icon, i+1)
		} else {
			s += color.New(fgColor).Sprintf(" - %s %d ", icon, i+1)
		}

		s += "\n"

		sb.WriteString(s)
	}

	if !anyReplacements {
		return ""
	}

	if anyApplied {
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
