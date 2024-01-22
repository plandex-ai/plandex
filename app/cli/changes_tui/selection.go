package changes_tui

import (
	"fmt"
	"log"

	"github.com/plandex/plandex/shared"
)

type selectionInfo struct {
	currentPath                   string
	currentRes                    *shared.PlanFileResult
	currentReplacements           []*shared.Replacement
	currentRep                    *shared.Replacement
	currentFilesBeforeReplacement *shared.CurrentPlanFiles
}

func (m *changesUIModel) setSelectionInfo() {
	paths := m.currentPlan.PlanResult.SortedPaths
	currentPath := paths[m.selectedFileIndex]

	results := m.currentPlan.PlanResult.FileResultsByPath[currentPath]

	var currentRes *shared.PlanFileResult
	var currentRep *shared.Replacement

	var pathReplacements []*shared.Replacement

	for _, res := range results {
		pathReplacements = append(pathReplacements, res.Replacements...)
	}

	i := 0
	for _, res := range results {
		for _, rep := range res.Replacements {
			if i == m.selectedReplacementIndex {
				currentRes = res
				currentRep = rep
			}
			i++
		}
	}

	var currentFilesBeforeReplacement *shared.CurrentPlanFiles
	var err error

	if currentRep == nil {
		currentFilesBeforeReplacement = m.currentPlan.CurrentPlanFiles
	} else {
		currentFilesBeforeReplacement, err = m.currentPlan.GetFilesBeforeReplacement(currentRep.Id)
	}
	if err != nil {
		err = fmt.Errorf("error getting current plan state: %v", err)
		log.Println(err)
		panic(err)
	}

	m.selectionInfo = &selectionInfo{
		currentPath:                   currentPath,
		currentRes:                    currentRes,
		currentReplacements:           pathReplacements,
		currentRep:                    currentRep,
		currentFilesBeforeReplacement: currentFilesBeforeReplacement,
	}
}
