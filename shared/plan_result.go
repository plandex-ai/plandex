package shared

import (
	"fmt"
	"sort"
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

func (state *CurrentPlanState) PendingChangesSummary() string {
	var msgs []string

	for _, desc := range state.PendingBuildDescriptions {

		pendingNewFilesSet := make(map[string]bool)
		pendingReplacementPathsSet := make(map[string]bool)
		pendingReplacementsByPath := make(map[string][]*Replacement)

		for _, result := range state.PlanResult.Results {
			if result.ConvoMessageId == desc.ConvoMessageId && result.IsPending() {
				if len(result.Replacements) == 0 && result.Content != "" {
					pendingNewFilesSet[result.Path] = true
				} else {
					pendingReplacementPathsSet[result.Path] = true
					pendingReplacementsByPath[result.Path] = append(pendingReplacementsByPath[result.Path], result.Replacements...)
				}
			}
		}

		if len(pendingNewFilesSet) == 0 && len(pendingReplacementPathsSet) == 0 {
			continue
		}

		var pendingNewFiles []string
		var pendingReplacementPaths []string

		for path := range pendingNewFilesSet {
			pendingNewFiles = append(pendingNewFiles, path)
		}

		for path := range pendingReplacementPathsSet {
			pendingReplacementPaths = append(pendingReplacementPaths, path)
		}

		sort.Slice(pendingReplacementPaths, func(i, j int) bool {
			return pendingReplacementPaths[i] < pendingReplacementPaths[j]
		})

		sort.Slice(pendingNewFiles, func(i, j int) bool {
			return pendingNewFiles[i] < pendingNewFiles[j]
		})

		var descMsgs []string

		descMsgs = append(descMsgs, fmt.Sprintf("📝 %s", desc.CommitMsg))

		if len(pendingNewFiles) > 0 {
			newMsg := "  📄 New files:\n\n"
			for _, path := range pendingNewFiles {
				newMsg += fmt.Sprintf("  • %s\n", path)
			}
			descMsgs = append(descMsgs, newMsg)
		}

		if len(pendingReplacementPaths) > 0 {
			updatesMsg := "  ✏️ Edits:\n\n"

			for _, path := range pendingReplacementPaths {
				updatesMsg += fmt.Sprintf("    • %s\n", path)

				replacements := pendingReplacementsByPath[path]

				for _, replacement := range replacements {
					updatesMsg += fmt.Sprintf("      ✅ %s\n", replacement.Summary)
				}
			}
			descMsgs = append(descMsgs, updatesMsg)
		}

		msgs = append(msgs, strings.Join(descMsgs, "\n\n"))
	}
	return strings.Join(msgs, "\n\n")
}
