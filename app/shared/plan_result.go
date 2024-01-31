package shared

import (
	"fmt"
	"log"
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

func (p PlanFileResultsByPath) OriginalContextForPath(path string) string {
	planResults := p[path]
	if len(planResults) == 0 {
		return ""
	}
	return planResults[0].ContextBody
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

			log.Println("Replacement failed: " + replacement.Old + " -> " + replacement.New)
			log.Println("Pre: " + pre)
			log.Println("Sub: " + sub)
			log.Println("Updated: " + updated)

		} else {
			replaced := strings.Replace(sub, replacement.Old, replacement.New, 1)

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
	planRes := planState.PlanResult

	files := make(map[string]string)
	shas := make(map[string]string)

	for path, planResults := range planRes.FileResultsByPath {
		updated := files[path]

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
			} else if updated == "" {
				updated = planRes.ContextBody
				shas[path] = planRes.ContextSha
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

	descByConvoMessageId := make(map[string]*ConvoMessageDescription)
	for _, desc := range state.PendingBuildDescriptions {
		if desc.ConvoMessageId == "" {
			log.Println("Warning: ConvoMessageId is empty for description:", desc)
			continue
		}

		descByConvoMessageId[desc.ConvoMessageId] = desc
	}

	type changeset struct {
		descsSet map[string]bool
		descs    []*ConvoMessageDescription
		results  []*PlanFileResult
	}
	byDescs := map[string]*changeset{}

	for _, result := range state.PlanResult.Results {
		convoIds := map[string]bool{}
		for _, convoMessageId := range result.ConvoMessageIds {
			if descByConvoMessageId[convoMessageId] != nil {
				convoIds[convoMessageId] = true
			}
		}
		var uniqueConvoIds []string
		for convoId := range convoIds {
			uniqueConvoIds = append(uniqueConvoIds, convoId)
		}

		composite := strings.Join(uniqueConvoIds, "|")
		if _, ok := byDescs[composite]; !ok {
			byDescs[composite] = &changeset{
				descsSet: make(map[string]bool),
			}
		}

		ch := byDescs[composite]
		ch.results = append(byDescs[composite].results, result)

		for _, convoMessageId := range uniqueConvoIds {
			if desc, ok := descByConvoMessageId[convoMessageId]; ok {
				if !ch.descsSet[convoMessageId] {
					ch.descs = append(ch.descs, desc)
					ch.descsSet[convoMessageId] = true
				}
			} else {
				log.Println("Warning: no description for convo message id:", convoMessageId)
			}
		}
	}

	var sortedChangesets []*changeset
	for _, ch := range byDescs {
		sortedChangesets = append(sortedChangesets, ch)
	}

	sort.Slice(sortedChangesets, func(i, j int) bool {
		// put changesets with no descriptions last, otherwise sort by date
		if len(sortedChangesets[i].descs) == 0 {
			return false
		}
		if len(sortedChangesets[j].descs) == 0 {
			return true
		}
		return sortedChangesets[i].descs[0].CreatedAt.Before(sortedChangesets[j].descs[0].CreatedAt)
	})

	for _, ch := range sortedChangesets {
		var descMsgs []string

		if len(ch.descs) == 0 {
			descMsgs = append(descMsgs, "📝 Changes")
		} else {
			for _, desc := range ch.descs {
				descMsgs = append(descMsgs, fmt.Sprintf("📝 %s", desc.CommitMsg))
			}
		}

		pendingNewFilesSet := make(map[string]bool)
		pendingReplacementPathsSet := make(map[string]bool)
		pendingReplacementsByPath := make(map[string][]*Replacement)

		for _, result := range ch.results {

			if result.IsPending() {
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

		msgs = append(msgs, descMsgs...)

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

		if len(pendingNewFiles) > 0 {
			newMsg := "  📄 New files:\n"
			for _, path := range pendingNewFiles {
				newMsg += fmt.Sprintf("  • %s\n", path)
			}
			msgs = append(msgs, newMsg)
		}

		if len(pendingReplacementPaths) > 0 {
			updatesMsg := "  ✏️ Edits:\n"

			for _, path := range pendingReplacementPaths {
				updatesMsg += fmt.Sprintf("    • %s\n", path)

				replacements := pendingReplacementsByPath[path]

				for _, replacement := range replacements {
					updatesMsg += fmt.Sprintf("      ✅ %s\n", replacement.Summary)
				}
			}
			msgs = append(msgs, updatesMsg)
		}

	}
	return strings.Join(msgs, "\n")
}
