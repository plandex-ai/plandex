package changes_tui

import (
	"fmt"
	"log"
	"plandex/api"
	"plandex/lib"

	"github.com/atotto/clipboard"
	"github.com/muesli/reflow/wrap"
	"github.com/plandex/plandex/shared"
)

func (m *changesUIModel) rejectFile() (*shared.CurrentPlanState, *shared.ApiError) {
	err := api.Client.RejectFile(lib.CurrentPlanId, lib.CurrentBranch, m.selectionInfo.currentPath)

	if err != nil {
		log.Printf("error rejecting file changes: %v", err)
		return nil, err
	}

	planState, err := api.Client.GetCurrentPlanState(lib.CurrentPlanId, lib.CurrentBranch)

	if err != nil {
		log.Printf("error getting current plan state: %v", err)
		return nil, err
	}

	return planState, nil
}

func (m *changesUIModel) copyCurrentChange() error {
	selectionInfo := m.selectionInfo
	if selectionInfo.currentRep == nil {
		return fmt.Errorf("no change is currently selected")
	}

	// Copy the 'New' content of the replacement to the clipboard
	if err := clipboard.WriteAll(selectionInfo.currentRep.New); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %v", err)
	}

	m.didCopy = true

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
	paths := m.currentPlan.PlanResult.SortedPaths

	if m.selectedFileIndex < len(paths)-1 {
		m.selectedFileIndex++
		m.selectedReplacementIndex = 0
	}

	m.setSelectionInfo()
	m.updateMainView(true)
}

func (m *changesUIModel) up() {
	if m.selectedReplacementIndex > 0 {
		// log.Println("up")
		m.selectedReplacementIndex--
		m.setSelectionInfo()
		m.updateMainView(true)
	}
}

func (m *changesUIModel) down() {
	var currentReplacements []*shared.Replacement
	if m.selectionInfo != nil {
		currentReplacements = m.selectionInfo.currentReplacements
	}

	max := len(currentReplacements) - 1

	// allow for selection of 'full file' option at bottom of replacement list in sidebar
	if m.currentPlan.PlanResult.NumPendingForPath(m.selectionInfo.currentPath) > 0 {
		max++
	}

	// allow for selection of 'new file' option at top of replacement list in sidebar
	if m.hasNewFile() {
		max++
	}

	if m.selectedReplacementIndex < max {
		// log.Println("down")
		m.selectedReplacementIndex++
		m.setSelectionInfo()
		m.updateMainView(true)
	}
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

func (m *changesUIModel) start() {
	if m.selectionInfo.currentRep == nil && m.fileScrollable() {
		m.fileViewport.GotoTop()
	} else if m.selectedViewport == 0 && m.oldScrollable() {
		m.changeOldViewport.GotoTop()
	} else if m.newScrollable() {
		m.changeNewViewport.GotoTop()
	}
}

func (m *changesUIModel) end() {
	if m.selectionInfo.currentRep == nil && m.fileScrollable() {
		m.fileViewport.GotoBottom()
	} else if m.selectedViewport == 0 && m.oldScrollable() {
		m.changeOldViewport.GotoBottom()
	} else if m.newScrollable() {
		m.changeNewViewport.GotoBottom()
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
	// log.Println("updateMainView")

	// var updateMsg types.ChangesUIViewportsUpdate

	if m.selectedNewFile() || m.selectedFullFile() {
		var updatedFile string

		if m.selectedNewFile() {
			updatedFile = m.selectionInfo.currentRes.Content
		} else {
			updatedFile = m.currentPlan.CurrentPlanFiles.Files[m.selectionInfo.currentPath]
		}

		wrapped := wrap.String(updatedFile, m.fileViewport.Width-2)
		m.fileViewport.SetContent(wrapped)
	} else {
		oldRes := m.getReplacementOldDisplay()
		m.changeOldViewport.SetContent(oldRes.oldDisplay)
		// log.Println("set old content")
		newContent, newContentDisplay := m.getReplacementNewDisplay(oldRes.prependContent, oldRes.appendContent)
		m.changeNewViewport.SetContent(newContentDisplay)
		// log.Println("set new content")

		if scrollReplacement {
			m.scrollReplacementIntoView(oldRes.old, newContent, oldRes.numLinesPrepended)
		}
	}

	m.updateViewportSizes()
}
