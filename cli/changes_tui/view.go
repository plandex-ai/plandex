package changes_tui

import (
	"strings"

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
	mainViewHeaderHeight := lipgloss.Height(m.renderMainViewHeader())
	mainViewFooterHeight := lipgloss.Height(m.renderMainViewFooter())
	mainViewWidth := m.width - sidebarWidth

	mainViewHeight := m.height - (helpHeight + tabsHeight)

	if m.selectedFullFile() {
		mainViewHeight -= mainViewHeaderHeight
	} else {
		mainViewHeight -= (mainViewHeaderHeight + mainViewFooterHeight)
	}

	return mainViewWidth, mainViewHeight
}

func (m *changesUIModel) initViewports() {
	mainViewWidth, mainViewHeight := m.getMainViewDims()
	m.changeOldViewport = viewport.New(mainViewWidth/2, mainViewHeight)
	m.changeOldViewport.Style = lipgloss.NewStyle().Padding(0, 1, 0, 1)
	m.changeNewViewport = viewport.New(mainViewWidth/2, mainViewHeight)
	m.changeNewViewport.Style = lipgloss.NewStyle().Padding(0, 1, 0, 1)
	m.fileViewport = viewport.New(mainViewWidth, mainViewHeight)
	m.fileViewport.Style = lipgloss.NewStyle().Padding(0, 1, 0, 1)
}

func (m *changesUIModel) updateViewportSizes() {
	mainViewWidth, mainViewHeight := m.getMainViewDims()

	if m.selectedFullFile() {
		fileViewHeight := mainViewHeight

		if m.fileScrollable() {
			footerHeight := lipgloss.Height(m.renderScrollFooter())
			fileViewHeight -= footerHeight
		}

		m.fileViewport.Width = mainViewWidth
		m.fileViewport.Height = fileViewHeight

	} else {
		oldViewHeight := mainViewHeight

		if m.oldScrollable() && (m.selectedViewport == 0 || !m.newScrollable()) {
			footerHeight := lipgloss.Height(m.renderScrollFooter())
			oldViewHeight -= footerHeight
		}

		newViewHeight := mainViewHeight

		if m.newScrollable() && (m.selectedViewport == 1 || !m.oldScrollable()) {
			footerHeight := lipgloss.Height(m.renderScrollFooter())
			newViewHeight -= footerHeight
		}

		m.changeOldViewport.Width = mainViewWidth / 2
		m.changeOldViewport.Height = oldViewHeight
		m.changeNewViewport.Width = mainViewWidth / 2
		m.changeNewViewport.Height = newViewHeight
	}
}

func (m changesUIModel) renderHelp() string {
	help := " "

	if len(m.currentPlan.SortedPaths) > 1 {
		help += "(←/→) select file • "
	}

	if m.renderSidebar() != "" {
		help += "(↑/↓) select change • "
	}

	help += "(ctrl+a) apply pending • (ctrl+d) drop all • (q)uit"
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

func (m changesUIModel) selectedFullFile() bool {
	return m.selectionInfo != nil && m.selectionInfo.currentRep == nil
}

func (m *changesUIModel) scrollReplacementIntoView(oldContent, newContent string, numLinesPrepended int) {
	scrollView := func(content string, view *viewport.Model) {
		view.GotoTop()

		if numLinesPrepended <= 2 {
			return
		}

		visibleLines := view.VisibleLineCount()
		contentLines := len(strings.Split(content, "\n"))

		if contentLines >= (visibleLines - 2) {
			view.LineDown(numLinesPrepended - 2)
		} else {
			diffAround := visibleLines - contentLines
			toScroll := numLinesPrepended - diffAround/2
			view.LineDown(toScroll)
		}
	}

	if m.oldScrollable() {
		scrollView(oldContent, &m.changeOldViewport)
	}
	if m.newScrollable() {
		scrollView(newContent, &m.changeNewViewport)
	}
}
