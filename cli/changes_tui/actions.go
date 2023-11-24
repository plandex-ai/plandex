package changes_tui

import (
	"fmt"

	"github.com/atotto/clipboard"
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

func (m *changesUIModel) up() {
	paths := m.currentPlan.SortedPaths
	if m.selectedReplacementIndex > 0 {
		m.selectedReplacementIndex--
	} else if m.selectedFileIndex > 0 {
		m.selectedFileIndex--
		m.selectedReplacementIndex = len(m.resultsInfo.ReplacementsByPath[paths[m.selectedFileIndex]]) - 1
	}
	m.setSelectionInfo()
	m.updateMainView()
}

func (m *changesUIModel) down() {
	paths := m.currentPlan.SortedPaths
	var currentReplacements []*shared.Replacement
	if m.selectionInfo != nil {
		currentReplacements = m.selectionInfo.currentReplacements
	}

	if m.selectedReplacementIndex < len(currentReplacements)-1 {
		m.selectedReplacementIndex++
	} else if m.selectedFileIndex < len(paths)-1 {
		m.selectedFileIndex++
		m.selectedReplacementIndex = 0
	}
	m.setSelectionInfo()
	m.updateMainView()
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

}

func (m *changesUIModel) pageDown() {

}

func (m *changesUIModel) switchView() {
	m.selectedViewport = 1 - m.selectedViewport
	m.updateMainView()
}

func (m *changesUIModel) windowResized(w, h int) {
	m.width = w
	m.height = h
	if !m.ready {
		m.initViewports()
		m.ready = true
	}
	m.updateMainView()
}

func (m *changesUIModel) updateMainView() {
	updatedFile := m.selectionInfo.currentPlanBeforeReplacement.CurrentPlanFiles.Files[m.selectionInfo.currentPath]

	oldContentDisplay, prependContent, appendContent := m.getReplacementOldDisplay()

	m.changeOldViewport.SetContent(oldContentDisplay)

	newContentDisplay := m.getReplacementNewDisplay(prependContent, appendContent)
	m.changeNewViewport.SetContent(newContentDisplay)

	m.fileViewport.SetContent(wrap.String(updatedFile, m.fileViewport.Width-2))

	m.updateViewportSizes()
}
