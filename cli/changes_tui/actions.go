package changes_tui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/fatih/color"
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

	// allow for selection of 'full file' option at bottom of replacement list in sidebar
	max := len(currentReplacements) - 1
	if m.selectionInfo.currentPlanBeforeReplacement.NumPendingForPath(m.selectionInfo.currentPath) > 0 {
		max++
	}

	if m.selectedReplacementIndex < max {
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
	if m.selectedFullFile() {
		updatedFile := m.currentPlan.CurrentPlanFiles.Files[m.selectionInfo.currentPath]

		wrapWidth := m.fileViewport.Width - 2

		replacements := m.selectionInfo.currentReplacements
		fileSegments := []string{}
		replacementSegments := map[int]bool{}
		lastReplacementIdx := 0
		for _, rep := range replacements {
			idx := strings.Index(updatedFile, rep.New)
			if idx == -1 || idx < lastReplacementIdx {
				continue
			}

			fileSegments = append(fileSegments, updatedFile[lastReplacementIdx:idx])
			fileSegments = append(fileSegments, rep.New)
			replacementSegments[len(fileSegments)-1] = true
			lastReplacementIdx = idx + len(rep.New)
		}
		fileSegments = append(fileSegments, updatedFile[lastReplacementIdx:])

		for i, segment := range fileSegments {
			wrapped := wrap.String(segment, wrapWidth)
			isReplacement := replacementSegments[i]
			if isReplacement {
				lines := strings.Split(wrapped, "\n")
				for j, line := range lines {
					lines[j] = color.New(color.FgHiGreen).Sprint(line)
				}
				wrapped = strings.Join(lines, "\n")
			}
			fileSegments[i] = wrapped
		}

		m.fileViewport.SetContent(strings.Join(fileSegments, ""))
		m.updateViewportSizes()

	} else {
		oldRes := m.getReplacementOldDisplay()
		m.changeOldViewport.SetContent(oldRes.oldDisplay)
		newContent, newContentDisplay := m.getReplacementNewDisplay(oldRes.prependContent, oldRes.appendContent)
		m.changeNewViewport.SetContent(newContentDisplay)

		m.updateViewportSizes()

		if scrollReplacement {
			m.scrollReplacementIntoView(oldRes.old, newContent, oldRes.numLinesPrepended)
		}
	}
}
