package changes_tui

import (
	"fmt"
	"log"
	"plandex/lib"
	"plandex/types"

	"github.com/plandex/plandex/shared"
)

type selectionInfo struct {
	currentPath                  string
	currentRes                   *shared.PlanResult
	currentReplacements          []*shared.Replacement
	currentRep                   *shared.Replacement
	currentPlanBeforeReplacement *types.CurrentPlanState
}

func (m *changesUIModel) setSelectionInfo() {
	paths := m.currentPlan.SortedPaths
	currentPath := paths[m.selectedFileIndex]

	results := m.currentPlan.PlanResByPath[currentPath]

	var currentRes *shared.PlanResult
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

	var currentPlanAtReplacement *types.CurrentPlanState
	var err error

	if currentRep == nil {
		currentPlanAtReplacement = m.currentPlan
	} else {
		currentPlanAtReplacement, err = lib.GetCurrentPlanStateBeforeReplacement(currentRep.Id)
	}
	if err != nil {
		err = fmt.Errorf("error getting current plan state: %v", err)
		log.Println(err)
		panic(err)
	}

	m.selectionInfo = &selectionInfo{
		currentPath:                  currentPath,
		currentRes:                   currentRes,
		currentReplacements:          pathReplacements,
		currentRep:                   currentRep,
		currentPlanBeforeReplacement: currentPlanAtReplacement,
	}
}
