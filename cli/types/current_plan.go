package types

import "github.com/plandex/plandex/shared"

type CurrentPlanState struct {
	PlanResultsInfo
	CurrentPlanFiles *shared.CurrentPlanFiles
	ModelContext     shared.ModelContext
	ContextByPath    map[string]*shared.ModelContextPart
}

func (p CurrentPlanState) NumPendingForPath(path string) int {
	res := 0
	results := p.PlanResByPath[path]
	for _, result := range results {
		if result.IsPending() {
			res += result.NumPendingReplacements()
		}
	}
	return res
}
