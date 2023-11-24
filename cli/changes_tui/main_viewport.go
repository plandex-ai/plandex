package changes_tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m changesUIModel) renderMainView() string {
	oldView := m.changeOldViewport.View()
	newView := m.changeNewViewport.View()

	oldViews := []string{oldView}
	newViews := []string{newView}

	if m.oldScrollable() && m.selectedViewport == 0 {
		oldViews = append(oldViews, m.renderScrollFooter())
	} else if m.newScrollable() {
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
		m.renderMainViewHeader(),
		lipgloss.JoinHorizontal(lipgloss.Top,
			oldContainer,
			newContainer,
		),
		m.renderMainViewFooter(),
	)
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

	header := " 📝 " + m.selectionInfo.currentRep.Summary
	return style.Render(header)
}

func (m changesUIModel) renderMainViewFooter() string {
	sidebarWidth := lipgloss.Width(m.renderSidebar())
	style := lipgloss.NewStyle().Width(m.width - sidebarWidth).Inherit(topBorderStyle).Foreground(lipgloss.Color(helpTextColor))
	footer := ` (d)iscard selected change • (c)opy to clipboard`
	return style.Render(footer)
}

func (m changesUIModel) renderScrollFooter() string {
	if m.selectionInfo == nil {
		return ""
	}

	width, _ := m.getMainViewDims()

	if m.selectionInfo.currentRes != nil {
		width = width / 2
	}

	footer := ` (j/k) scroll`

	if m.oldScrollable() && m.newScrollable() {
		footer += ` • (tab) switch view`
	}

	style := lipgloss.NewStyle().Width(width).Inherit(topBorderStyle).Foreground(lipgloss.Color(helpTextColor))

	return style.Render(footer)
}
