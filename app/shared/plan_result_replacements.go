package shared

import (
	"fmt"
	"log"
	"strings"
	"time"
)

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
	updatedAtByPath := make(map[string]time.Time)

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

			updatedAtByPath[path] = planRes.CreatedAt
		}

		files[path] = updated
	}

	return &CurrentPlanFiles{Files: files, ContextShas: shas}, nil
}
