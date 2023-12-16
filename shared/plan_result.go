package shared

import (
	"fmt"
	"strings"
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

func ApplyReplacements(content string, replacements []*Replacement, setFailed bool) (string, bool) {
	updated := content
	lastInsertedIdx := 0

	allSucceeded := true

	for _, replacement := range replacements {
		pre := updated[:lastInsertedIdx]
		sub := updated[lastInsertedIdx:]
		originalIdx := strings.Index(sub, replacement.Old)

		if originalIdx == -1 {
			allSucceeded = false
			if setFailed {
				replacement.Failed = true
			}

			// jsonBytes, _ := json.Marshal(replacement)
			// log.Println(string(jsonBytes))

			// log.Println("Replacement: " + replacement.Old + " -> " + replacement.New)

		} else {
			replaced := strings.Replace(sub, replacement.Old, replacement.New, 1)

			// log.Println("Replacement: " + replacement.Old + " -> " + replacement.New)
			// log.Println("Pre: " + pre)
			// log.Println("Sub: " + sub)
			// log.Println("Idx: " + fmt.Sprintf("%d", idx))
			// log.Println("Updated: " + updated)

			updated = pre + replaced
			lastInsertedIdx = lastInsertedIdx + originalIdx + len(replacement.New)
		}
	}

	return updated, allSucceeded
}

func (planState *CurrentPlanState) GetFiles() (*CurrentPlanFiles, error) {
	return planState.GetFilesBeforeReplacement("")
}

func (planState *CurrentPlanState) GetFilesBeforeReplacement(
	replacementId string,
) (*CurrentPlanFiles, error) {
	contexts := planState.Contexts
	planRes := planState.PlanResult

	files := make(map[string]string)
	shas := make(map[string]string)

	for _, contextPart := range contexts {
		if contextPart.FilePath == "" {
			continue
		}

		_, hasPath := planRes.FileResultsByPath[contextPart.FilePath]

		// fmt.Printf("hasPath: %v\n", hasPath)

		if hasPath {
			files[contextPart.FilePath] = contextPart.Body
			shas[contextPart.FilePath] = contextPart.Sha
		}
	}

	for path, planResults := range planRes.FileResultsByPath {
		updated := files[path]

		// fmt.Printf("path: %s\n", path)
		// fmt.Printf("updated: %s\n", updated)

	PlanResLoop:
		for _, planRes := range planResults {
			if !planRes.IsPending() {
				continue
			}

			if len(planRes.Replacements) == 0 {
				if updated != "" {
					return nil, fmt.Errorf("plan updates out of order: %s", path)
				}

				updated = planRes.Content
				files[path] = updated
				continue
			}

			contextSha := shas[path]

			if contextSha != "" && planRes.ContextSha != contextSha {
				return nil, fmt.Errorf("result sha doesn't match context sha: %s", path)
			}

			if len(planRes.Replacements) == 0 {
				continue
			}

			replacements := []*Replacement{}
			for _, replacement := range planRes.Replacements {
				if replacement.Id == replacementId {
					break PlanResLoop
				}
				replacements = append(replacements, replacement)
			}

			var allSucceeded bool
			updated, allSucceeded = ApplyReplacements(updated, replacements, false)

			if !allSucceeded {
				return nil, fmt.Errorf("plan replacement failed: %s", path)
			}
		}

		files[path] = updated
	}

	return &CurrentPlanFiles{Files: files, ContextShas: shas}, nil
}
