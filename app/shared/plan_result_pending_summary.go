package shared

import (
	"fmt"
	"log"
	"sort"
	"strings"
)

func (state *CurrentPlanState) PendingChangesSummaryForBuild() string {
	return state.pendingChangesSummary(false)
}

func (state *CurrentPlanState) PendingChangesSummaryForApply() string {
	return state.pendingChangesSummary(true)
}

func (state *CurrentPlanState) pendingChangesSummary(forApply bool) string {
	var msgs []string

	descByConvoMessageId := make(map[string]*ConvoMessageDescription)

	for _, desc := range state.ConvoMessageDescriptions {
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
		// log.Println("result:")
		// spew.Dump(result)

		convoIds := map[string]bool{}
		if descByConvoMessageId[result.ConvoMessageId] != nil {
			convoIds[result.ConvoMessageId] = true
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

		// log.Println("uniqueConvoIds:", uniqueConvoIds)

		for _, convoMessageId := range uniqueConvoIds {
			if desc, ok := descByConvoMessageId[convoMessageId]; ok {
				if !ch.descsSet[convoMessageId] && (!(desc.DidBuild && !forApply) || len(desc.BuildPathsInvalidated) > 0) {
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

	isRebuild := true
	rebuildPathsSet := make(map[string]bool)

	if forApply {
		msgs = append(msgs, "🤖 Plandex → apply pending changes")
	} else {
		for _, ch := range sortedChangesets {
			allRebuild := true
			for _, desc := range ch.descs {
				if len(desc.BuildPathsInvalidated) == 0 {
					allRebuild = false
					break
				} else {
					for path := range desc.BuildPathsInvalidated {
						rebuildPathsSet[path] = true
					}
				}
			}
			if !allRebuild {
				isRebuild = false
				break
			}
		}

		if isRebuild {
			msgs = append(msgs, "🏗️  Rebuild paths invalidated by context update")
			for path := range rebuildPathsSet {
				msgs = append(msgs, fmt.Sprintf("  • rebuild → %s", path))
			}
			return strings.Join(msgs, "\n")
		}

		msgs = append(msgs, "🏗️  Build pending changes")
	}

	for _, ch := range sortedChangesets {
		var descMsgs []string

		if len(ch.descs) == 0 {
			// log.Println("Warning: no descriptions for changeset")
			// spew.Dump(ch)
			continue
			// descMsgs = append(descMsgs, "  ✏️  Changes")
		}

		for _, desc := range ch.descs {
			descMsgs = append(descMsgs, fmt.Sprintf("  ✏️  %s", desc.CommitMsg))
		}

		msgs = append(msgs, descMsgs...)

		// for an apply commit message, we don't need to list file updates
		if forApply {
			continue
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
			for _, path := range pendingNewFiles {
				msgs = append(msgs, fmt.Sprintf("    • new file → %s", path))
			}
		}

		if len(pendingReplacementPaths) > 0 {
			for _, path := range pendingReplacementPaths {
				msgs = append(msgs, fmt.Sprintf("    • edit → %s", path))
			}

		}

	}
	return strings.Join(msgs, "\n")
}
