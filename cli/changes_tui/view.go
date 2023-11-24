package changes_tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

var borderColor = lipgloss.Color("#444")
var helpTextColor = lipgloss.Color("#ddd")
var topBorderStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderTop(true).
	BorderForeground(borderColor)

func (m changesUIModel) View() string {
	help := m.renderHelp()

	tabs := m.renderPathTabs()

	sidebar := m.renderSidebar()

	mainView := m.renderMainView()

	layout := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainView)

	return lipgloss.JoinVertical(lipgloss.Left,
		tabs,
		layout,
		help,
	)
}

func (m changesUIModel) getMainViewDims() (int, int) {
	tabsHeight := lipgloss.Height(m.renderPathTabs())
	helpHeight := lipgloss.Height(m.renderHelp())
	sidebarWidth := lipgloss.Width(m.renderSidebar())
	mainViewWidth := m.width - sidebarWidth
	mainViewHeight := m.height - (helpHeight + tabsHeight + lipgloss.Height(m.renderMainViewHeader()) + lipgloss.Height(m.renderMainViewFooter()))

	return mainViewWidth, mainViewHeight
}

func (m *changesUIModel) initViewports() {
	mainViewWidth, mainViewHeight := m.getMainViewDims()
	m.changeOldViewport = viewport.New(mainViewWidth/2, mainViewHeight)
	m.changeOldViewport.Style = lipgloss.NewStyle().Padding(1)
	m.changeNewViewport = viewport.New(mainViewWidth/2, mainViewHeight)
	m.changeNewViewport.Style = lipgloss.NewStyle().Padding(1)
	m.fileViewport = viewport.New(mainViewWidth, mainViewHeight)
	m.fileViewport.Style = lipgloss.NewStyle().Padding(1)
}

func (m *changesUIModel) updateViewportSizes() {
	mainViewWidth, mainViewHeight := m.getMainViewDims()

	oldViewHeight := mainViewHeight

	if m.selectedViewport == 0 && m.changeOldViewport.TotalLineCount() > m.changeOldViewport.VisibleLineCount() {
		footerHeight := lipgloss.Height(m.renderScrollFooter())
		oldViewHeight -= footerHeight
	}

	newViewHeight := mainViewHeight

	if m.selectedViewport == 1 && m.changeNewViewport.TotalLineCount() > m.changeNewViewport.VisibleLineCount() {
		footerHeight := lipgloss.Height(m.renderScrollFooter())
		newViewHeight -= footerHeight
	}

	fileViewHeight := mainViewHeight

	if m.fileViewport.TotalLineCount() > m.fileViewport.VisibleLineCount() {
		footerHeight := lipgloss.Height(m.renderScrollFooter())
		fileViewHeight -= footerHeight
	}

	m.changeOldViewport.Width = mainViewWidth / 2
	m.changeOldViewport.Height = oldViewHeight
	m.changeNewViewport.Width = mainViewWidth / 2
	m.changeNewViewport.Height = newViewHeight
	m.fileViewport.Width = mainViewWidth
	m.fileViewport.Height = fileViewHeight
}

func (m changesUIModel) renderHelp() string {
	help := ` (↑/↓) select change • (ctrl+a) apply pending • (ctrl+d) discard all • (q)uit`
	style := lipgloss.NewStyle().Width(m.width).Inherit(topBorderStyle).Foreground(lipgloss.Color(helpTextColor))
	return style.Render(help)
}

func (m changesUIModel) oldScrollable() bool {
	return m.changeOldViewport.TotalLineCount() > m.changeOldViewport.VisibleLineCount()
}

func (m changesUIModel) newScrollable() bool {
	return m.changeNewViewport.TotalLineCount() > m.changeNewViewport.VisibleLineCount()
}

func (m changesUIModel) fileScrollable() bool {
	return m.fileViewport.TotalLineCount() > m.fileViewport.VisibleLineCount()
}
