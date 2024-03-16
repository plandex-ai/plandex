package changes_tui

import (
	"fmt"
	"plandex/term"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

func (m changesUIModel) renderMainView() string {
	// log.Println()
	// log.Println("renderMainView")

	mainViewHeader := m.renderMainViewHeader()

	if m.selectedNewFile() || m.selectedFullFile() {
		fileView := m.fileViewport.View()

		fileViews := []string{fileView}

		if m.fileScrollable() {
			fileViews = append(fileViews, m.renderScrollFooter())
		}

		fileContainer := lipgloss.JoinVertical(lipgloss.Left, fileViews...)

		fileContainerStyle := lipgloss.NewStyle().Width(m.fileViewport.Width)
		fileContainer = fileContainerStyle.Render(fileContainer)

		return lipgloss.JoinVertical(lipgloss.Left, mainViewHeader, fileContainer)
	} else {
		oldView := m.changeOldViewport.View()
		newView := m.changeNewViewport.View()

		oldViews := []string{oldView}
		newViews := []string{newView}

		if m.oldScrollable() && (m.selectedViewport == 0 || !m.newScrollable()) {
			oldViews = append(oldViews, m.renderScrollFooter())
		} else if m.newScrollable() && (m.selectedViewport == 1 || !m.oldScrollable()) {
			newViews = append(newViews, m.renderScrollFooter())
		}

		oldContainer := lipgloss.JoinVertical(lipgloss.Left, oldViews...)
		newContainer := lipgloss.JoinVertical(lipgloss.Left, newViews...)

		oldContainerStyle := lipgloss.NewStyle().Width(m.changeOldViewport.Width)
		oldContainer = oldContainerStyle.Render(oldContainer)

		newContainerStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderLeft(true).
			BorderForeground(lipgloss.Color(borderColor)).
			Width(m.changeNewViewport.Width)

		newContainer = newContainerStyle.Render(newContainer)

		return lipgloss.JoinVertical(lipgloss.Left,
			mainViewHeader,
			lipgloss.JoinHorizontal(lipgloss.Top, oldContainer, newContainer),
			m.renderMainViewFooter(),
		)
	}

}

func (m changesUIModel) renderMainViewHeader() string {
	if m.selectionInfo == nil {
		return "\n"
	}

	sidebarWidth := lipgloss.Width(m.renderSidebar())
	style := lipgloss.NewStyle().
		Width(m.width - sidebarWidth).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(borderColor)

	var header string
	if m.selectedFullFile() {
		numChanges := m.currentPlan.PlanResult.NumPendingForPath(m.selectionInfo.currentPath)
		if m.hasNewFile() {
			numChanges++
		}

		if numChanges > 0 {
			suffix := "s"
			if numChanges == 1 {
				suffix = ""
			}
			header = fmt.Sprintf(" ✅ Final state of %s (%d change%s)", m.selectionInfo.currentPath, numChanges, suffix)
		} else {
			header = fmt.Sprintf(" 🌟 New file: %s", m.selectionInfo.currentPath)
		}
	} else if m.selectedNewFile() {
		numChanges := m.currentPlan.PlanResult.NumPendingForPath(m.selectionInfo.currentPath)
		icon := "🌟"
		if numChanges > 1 {
			icon = "👉"
		}

		header = fmt.Sprintf(" %s New file: %s", icon, m.selectionInfo.currentPath)

	} else {
		header = " 👉 " + m.selectionInfo.currentRep.StreamedChange.Summary
	}

	return style.Render(header)
}

func (m changesUIModel) renderMainViewFooter() string {
	if m.selectedFullFile() {
		return ""
	}

	sidebarWidth := lipgloss.Width(m.renderSidebar())
	style := lipgloss.NewStyle().
		Width(m.width - sidebarWidth).
		Inherit(topBorderStyle).
		Foreground(lipgloss.Color(helpTextColor))
	var footer string
	if m.didCopy {
		footer = color.New(color.Bold, term.ColorHiCyan).Sprint(` copied to clipboard`)
	} else {
		footer = ` (c)opy change to clipboard • (r)eject file`
	}
	return style.Render(footer)
}

func (m changesUIModel) renderScrollFooter() string {
	if m.selectionInfo == nil {
		return ""
	}

	width, _ := m.getMainViewDims()

	if !m.selectedNewFile() && !m.selectedFullFile() {
		width = width / 2
	}

	var footer string

	if m.selectedNewFile() || m.selectedFullFile() {
		footer = ` (j/k) scroll • (d/u) page • (g/G) start/end • (r)eject file`
	} else {
		footer = ` (j/k) scroll`
		if m.oldScrollable() && m.newScrollable() {
			footer += ` • (tab) switch view`
		}
	}

	style := lipgloss.NewStyle().
		Width(width).
		Inherit(topBorderStyle).
		Foreground(lipgloss.Color(helpTextColor))

	return style.Render(footer)
}
