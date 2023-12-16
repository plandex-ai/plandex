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

	for i, res := range results {
		for j, rep := range res.Replacements {
			pathReplacements = append(pathReplacements, rep)

			flatIndex := i*len(res.Replacements) + j
			if flatIndex == m.selectedReplacementIndex {
				currentRes = res
				currentRep = rep
			}
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
