package changes_tui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/muesli/reflow/wrap"
	"github.com/plandex/plandex/shared"
)

func (m changesUIModel) rejectChange() error {

	return nil
}

func (m changesUIModel) applyAllChanges() error {

	return nil
}

func (m changesUIModel) rejectAllChanges() error {

	return nil
}

func (m changesUIModel) copyCurrentChange() error {
	selectionInfo := m.selectionInfo
	if selectionInfo.currentRep == nil {
		return fmt.Errorf("no change is currently selected")
	}

	// Copy the 'New' content of the replacement to the clipboard
	if err := clipboard.WriteAll(selectionInfo.currentRep.New); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %v", err)
	}

	return nil
}

func (m *changesUIModel) left() {
	if m.selectedFileIndex > 0 {
		m.selectedFileIndex--
		m.selectedReplacementIndex = 0
	}
	m.setSelectionInfo()
	m.updateMainView(true)
}

func (m *changesUIModel) right() {
	paths := m.currentPlan.SortedPaths

	if m.selectedFileIndex < len(paths)-1 {
		m.selectedFileIndex++
		m.selectedReplacementIndex = 0
	}

	m.setSelectionInfo()
	m.updateMainView(true)
}

func (m *changesUIModel) up() {
	if m.selectedReplacementIndex > 0 {
		m.selectedReplacementIndex--
	}
	m.setSelectionInfo()
	m.updateMainView(true)
}

func (m *changesUIModel) down() {
	var currentReplacements []*shared.Replacement
	if m.selectionInfo != nil {
		currentReplacements = m.selectionInfo.currentReplacements
	}

	if m.selectedReplacementIndex < len(currentReplacements) {
		m.selectedReplacementIndex++
	}
	m.setSelectionInfo()
	m.updateMainView(true)
}

func (m *changesUIModel) scrollUp() {
	if m.selectionInfo.currentRep == nil && m.fileScrollable() {
		m.fileViewport.LineUp(1)
	} else if m.selectedViewport == 0 && m.oldScrollable() {
		m.changeOldViewport.LineUp(1)
	} else if m.newScrollable() {
		m.changeNewViewport.LineUp(1)
	}
}

func (m *changesUIModel) scrollDown() {
	if m.selectionInfo.currentRep == nil && m.fileScrollable() {
		m.fileViewport.LineDown(1)
	} else if m.selectedViewport == 0 && m.oldScrollable() {
		m.changeOldViewport.LineDown(1)
	} else if m.newScrollable() {
		m.changeNewViewport.LineDown(1)
	}

}

func (m *changesUIModel) pageUp() {
	if m.selectionInfo.currentRep == nil && m.fileScrollable() {
		m.fileViewport.ViewUp()
	} else if m.selectedViewport == 0 && m.oldScrollable() {
		m.changeOldViewport.ViewUp()
	} else if m.newScrollable() {
		m.changeNewViewport.ViewUp()
	}
}

func (m *changesUIModel) pageDown() {
	if m.selectionInfo.currentRep == nil && m.fileScrollable() {
		m.fileViewport.ViewDown()
	} else if m.selectedViewport == 0 && m.oldScrollable() {
		m.changeOldViewport.ViewDown()
	} else if m.newScrollable() {
		m.changeNewViewport.ViewDown()
	}
}

func (m *changesUIModel) switchView() {
	m.selectedViewport = 1 - m.selectedViewport
	m.updateMainView(false)
}

func (m *changesUIModel) windowResized(w, h int) {
	m.width = w
	m.height = h
	didInit := false
	if !m.ready {
		m.initViewports()
		m.ready = true
		didInit = true
	}
	m.updateMainView(didInit)
}

func (m *changesUIModel) updateMainView(scrollReplacement bool) {
	updatedFile := m.selectionInfo.currentPlanBeforeReplacement.CurrentPlanFiles.Files[m.selectionInfo.currentPath]

	oldRes := m.getReplacementOldDisplay()

	m.changeOldViewport.SetContent(oldRes.oldDisplay)

	newContent, newContentDisplay := m.getReplacementNewDisplay(oldRes.prependContent, oldRes.appendContent)
	m.changeNewViewport.SetContent(newContentDisplay)

	m.fileViewport.SetContent(wrap.String(updatedFile, m.fileViewport.Width-2))

	m.updateViewportSizes()

	if scrollReplacement {
		m.scrollReplacementIntoView(oldRes.old, newContent, oldRes.numLinesPrepended)
	}
}

func (m *changesUIModel) scrollReplacementIntoView(oldContent, newContent string, numLinesPrepended int) {
	if numLinesPrepended <= 3 {
		return
	}

	scrollView := func(content string, view *viewport.Model) {
		visibleLines := view.VisibleLineCount()
		contentLines := len(strings.Split(content, "\n"))

		if contentLines >= (visibleLines - 3) {
			view.LineDown(numLinesPrepended - 3)
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
