package shared

import (
	"time"
)

func (rep *Replacement) IsPending() bool {
	return !rep.Failed && rep.RejectedAt == nil
}

func (rep *Replacement) SetRejected(t time.Time) {
	rep.RejectedAt = &t
}

func (res *PlanFileResult) NumPendingReplacements() int {
	numPending := 0
	for _, rep := range res.Replacements {
		if rep.IsPending() {
			numPending++
		}
	}
	return numPending
}

func (res *PlanFileResult) IsPending() bool {
	return res.AppliedAt == nil && res.RejectedAt == nil && (res.Content != "" || res.NumPendingReplacements() > 0)
}

func (p PlanFileResultsByPath) SetApplied(t time.Time) {
	for _, planResults := range p {
		for _, planResult := range planResults {
			if !planResult.IsPending() {
				continue
			}
			planResult.AppliedAt = &t
		}
	}
}

func (p PlanFileResultsByPath) SetRejected(t time.Time) int {
	numRejected := 0
	for _, planResults := range p {
		for _, planResult := range planResults {
			if !planResult.IsPending() {
				continue
			}
			planResult.RejectedAt = &t
			numRejected++

			for _, rep := range planResult.Replacements {
				rep.SetRejected(t)
			}
		}
	}
	return numRejected
}

func (p PlanFileResultsByPath) NumPending() int {
	numPending := 0
	for _, planResults := range p {
		for _, planResult := range planResults {
			if planResult.IsPending() {
				numPending++
			}
		}
	}
	return numPending
}

func (p PlanFileResultsByPath) ConflictedPaths(filesByPath map[string]string) map[string]bool {
	conflictedPaths := map[string]bool{}

	for path, body := range filesByPath {
		planRes := p[path]

		// log.Println("Checking for conflicts in path:", path)
		// log.Println("Body:")
		// log.Println(body)

		if planRes == nil {
			continue
		}

		updated := body

		noConflicts := true
		for _, res := range planRes {

			// log.Println("res:", res.Id)
			if len(res.Replacements) == 0 {
				continue
			}

			var succeeded bool
			updated, succeeded = ApplyReplacements(updated, res.Replacements, false)

			// log.Println("updated:", updated)
			// log.Println("succeeded:", succeeded)

			if !succeeded {
				noConflicts = false
				break
			}
		}

		// log.Println("No conflicts:", noConflicts)

		if !noConflicts {
			conflictedPaths[path] = true
		}
	}

	return conflictedPaths
}

func (r PlanResult) NumPendingForPath(path string) int {
	res := 0
	results := r.FileResultsByPath[path]
	for _, result := range results {
		if result.IsPending() {
			res += result.NumPendingReplacements()
		}
	}
	return res
}

func (desc *ConvoMessageDescription) NumBuildsPendingByPath() map[string]int {
	res := map[string]int{}
	if (!desc.DidBuild && len(desc.Files) > 0) || len(desc.BuildPathsInvalidated) > 0 {
		for _, file := range desc.Files {
			res[file]++
		}
	}
	return res
}

func (desc *ConvoMessageDescription) HasPendingBuilds() bool {
	return len(desc.NumBuildsPendingByPath()) > 0
}

func NumBuildsPendingByPath(planDescs []*ConvoMessageDescription) map[string]int {
	res := map[string]int{}
	for _, desc := range planDescs {
		for file, num := range desc.NumBuildsPendingByPath() {
			res[file] += num
		}
	}
	return res
}

func HasPendingBuilds(planDescs []*ConvoMessageDescription) bool {
	return len(NumBuildsPendingByPath(planDescs)) > 0
}

func (c *CurrentPlanState) NumBuildsPendingByPath() map[string]int {
	return NumBuildsPendingByPath(c.ConvoMessageDescriptions)
}

func (c *CurrentPlanState) HasPendingBuilds() bool {
	return len(c.NumBuildsPendingByPath()) > 0
}
