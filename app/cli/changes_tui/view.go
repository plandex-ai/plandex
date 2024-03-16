package changes_tui

import (
	"plandex/term"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

var borderColor = lipgloss.Color("#444")
var helpTextColor = lipgloss.Color("#ddd")
var topBorderStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderTop(true).
	BorderForeground(borderColor)

func (m changesUIModel) View() string {
	if m.isConfirmingRejectFile {
		return m.renderConfirmRejectFile()
	}

	if m.isRejectingFile {
		return m.renderIsRejectingFile()
	}

	help := m.renderHelp()

	tabs := m.renderPathTabs()

	sidebar := m.renderSidebar()

	mainView := m.renderMainView()

	layout := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainView)

	view := lipgloss.JoinVertical(lipgloss.Left,
		tabs,
		layout,
		help,
	)

	return view
}

func (m changesUIModel) getMainViewDims() (int, int) {
	tabsHeight := lipgloss.Height(m.renderPathTabs())
	helpHeight := lipgloss.Height(m.renderHelp())
	sidebarWidth := lipgloss.Width(m.renderSidebar())
	mainViewHeaderHeight := lipgloss.Height(m.renderMainViewHeader())
	mainViewFooterHeight := lipgloss.Height(m.renderMainViewFooter())

	mainViewWidth := m.width - sidebarWidth
	mainViewHeight := m.height - (helpHeight + tabsHeight)

	if m.selectedNewFile() || m.selectedFullFile() {
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
	// log.Println()
	// log.Println()
	// log.Println("updateViewportSizes")

	mainViewWidth, mainViewHeight := m.getMainViewDims()

	if m.selectedNewFile() || m.selectedFullFile() {
		fileViewHeight := mainViewHeight

		if m.fileScrollable() {
			footerHeight := lipgloss.Height(m.renderScrollFooter())
			fileViewHeight -= footerHeight
		}

		m.fileViewport.Width = mainViewWidth
		m.fileViewport.Height = fileViewHeight

	} else {
		// log.Println("mainViewHeight", mainViewHeight)

		mainViewHeight := mainViewHeight
		oldViewHeight := mainViewHeight
		newViewHeight := mainViewHeight
		// set widths and reset heights
		// log.Println("resetting widths and heights")
		m.resetViewportDims()

		// log.Println("oldScrollable", m.oldScrollable())
		// log.Println("newScrollable", m.newScrollable())
		// log.Println("selectedViewport", m.selectedViewport)

		if m.oldScrollable() && (m.selectedViewport == 0 || !m.newScrollable()) {
			footerHeight := lipgloss.Height(m.renderScrollFooter())
			oldViewHeight -= footerHeight
		} else if m.newScrollable() && (m.selectedViewport == 1 || !m.oldScrollable()) {
			footerHeight := lipgloss.Height(m.renderScrollFooter())
			newViewHeight -= footerHeight
		}

		// log.Println("oldViewHeight", oldViewHeight)
		// log.Println("newViewHeight", newViewHeight)

		// set updated heights
		m.changeOldViewport.Height = oldViewHeight
		m.changeNewViewport.Height = newViewHeight

		// log.Println("updated heights")
	}
}

func (m *changesUIModel) resetViewportDims() {
	mainViewWidth, mainViewHeight := m.getMainViewDims()
	m.fileViewport.Width = mainViewWidth
	m.fileViewport.Height = mainViewHeight
	m.changeOldViewport.Width = mainViewWidth / 2
	m.changeOldViewport.Height = mainViewHeight
	m.changeNewViewport.Width = mainViewWidth / 2
	m.changeNewViewport.Height = mainViewHeight
}

func (m changesUIModel) renderHelp() string {
	help := " "

	if len(m.currentPlan.PlanResult.SortedPaths) > 1 {
		help += "(←/→) select file • "
	}

	if m.selectionInfo != nil {
		help += "(↑/↓) select change • "
	}

	help += "(ctrl+a) apply all changes • (q)uit"
	style := lipgloss.NewStyle().Width(m.width).Inherit(topBorderStyle).Foreground(lipgloss.Color(helpTextColor))
	return style.Render(help)
}

func (m changesUIModel) oldScrollable() bool {
	// log.Println("oldScrollable")
	// log.Println("TotalLineCount", m.changeOldViewport.TotalLineCount())
	// log.Println("VisibleLineCount", m.changeOldViewport.VisibleLineCount())

	return m.changeOldViewport.TotalLineCount() > m.changeOldViewport.VisibleLineCount()
}

func (m changesUIModel) newScrollable() bool {
	// log.Println("newScrollable")
	// log.Println("TotalLineCount", m.changeNewViewport.TotalLineCount())
	// log.Println("VisibleLineCount", m.changeNewViewport.VisibleLineCount())

	return m.changeNewViewport.TotalLineCount() > m.changeNewViewport.VisibleLineCount()
}

func (m changesUIModel) fileScrollable() bool {
	return m.fileViewport.TotalLineCount() > m.fileViewport.VisibleLineCount()
}

func (m changesUIModel) hasNewFile() bool {
	firstRes := m.currentPlan.PlanResult.FileResultsByPath[m.selectionInfo.currentPath][0]
	return len(firstRes.Replacements) == 0 && firstRes.Content != ""
}

func (m changesUIModel) selectedNewFile() bool {
	return m.selectionInfo != nil &&
		m.hasNewFile() &&
		m.selectedReplacementIndex == 0 &&
		m.selectionInfo.currentRes != nil &&
		len(m.selectionInfo.currentRes.Replacements) == 0 &&
		m.selectionInfo.currentRes.Content != ""
}

func (m changesUIModel) selectedFullFile() bool {
	return !m.selectedNewFile() && m.selectionInfo != nil && m.selectionInfo.currentRep == nil
}

func (m *changesUIModel) scrollReplacementIntoView(oldContent, newContent string, numLinesPrepended int) {
	scrollView := func(content string, view *viewport.Model) {
		view.GotoTop()

		if numLinesPrepended <= 2 {
			return
		}

		totalLines := view.TotalLineCount()
		visibleLines := view.VisibleLineCount()
		contentLines := len(strings.Split(content, "\n"))

		if contentLines >= (visibleLines - 2) {
			view.LineDown(numLinesPrepended - 2)
		} else {
			toScroll := getSnippetScrollPosition(totalLines, visibleLines, numLinesPrepended, contentLines)
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

func (m changesUIModel) renderConfirmRejectFile() string {
	style := lipgloss.NewStyle().Padding(1).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color(borderColor)).Width(m.width - 2).Height(m.height - 2)

	prompt := color.New(color.Bold).Sprintf("🧐 Are you sure you want to reject changes to ") +
		color.New(color.Bold, term.ColorHiMagenta).Sprint(m.selectionInfo.currentPath) + "?\n\n" +
		color.New(term.ColorHiCyan, color.Bold).Sprintf("(y)es | (n)o")

	return style.Render(prompt)
}

func (m changesUIModel) renderIsRejectingFile() string {
	style := lipgloss.NewStyle().Padding(1).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color(borderColor)).Width(m.width - 2).Height(m.height - 2)

	return style.Render(m.spinner.View())
}

func getSnippetScrollPosition(totalLines, viewportHeight, snippetLineIndex, snippetHeight int) int {
	snippetMiddleLine := snippetLineIndex + snippetHeight/2
	viewportMiddleLine := viewportHeight / 2

	// Initial target scroll position
	scrollPosition := snippetMiddleLine - viewportMiddleLine

	// Adjust for bounds
	scrollPosition = max(scrollPosition, 0)                         // Adjust for top bound
	scrollPosition = min(scrollPosition, totalLines-viewportHeight) // Adjust for bottom bound

	return scrollPosition
}
