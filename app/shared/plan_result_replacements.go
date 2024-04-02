package shared

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
)

func ApplyReplacements(content string, replacements []*Replacement, setFailed bool) (string, bool) {
	updated := content

	allSucceeded := true

	for _, replacement := range replacements {
		originalIdx := strings.Index(updated, replacement.Old)

		// log.Println("originalIdx:", originalIdx)

		if originalIdx == -1 {
			allSucceeded = false
			if setFailed {
				replacement.Failed = true
			}

			log.Println("Replacement failed:")
			log.Println(spew.Sdump(replacement))

			log.Println("Updated:")
			log.Println(updated)

		} else {
			updated = strings.Replace(updated, replacement.Old, replacement.New, 1)
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
		// log.Println("path: ", path)

	PlanResLoop:
		for _, planRes := range planResults {

			// log.Println("planRes: ", planRes.Id)

			if !planRes.IsPending() {
				// log.Println("Plan result is not pending -- continuing loop")
				continue
			}

			if len(planRes.Replacements) == 0 {
				if updated != "" {
					return nil, fmt.Errorf("plan updates out of order: %s", path)
				}

				updated = planRes.Content
				files[path] = updated
				updatedAtByPath[path] = planRes.CreatedAt

				// log.Println("No replacements for plan result -- creating file and continuing loop")

				continue
			} else if updated == "" {
				context := planState.ContextsByPath[path]

				if context == nil {
					log.Printf("No context for path: %s\n", path)
					return nil, fmt.Errorf("no context for path: %s", path)
				}

				// log.Println("No updated content -- setting to context body")

				updated = context.Body
				shas[path] = context.Sha
			}

			replacements := []*Replacement{}
			foundTarget := false
			for _, replacement := range planRes.Replacements {
				if replacement.Id == replacementId {
					// log.Println("Found target replacement")
					foundTarget = true
					break
				}
				replacements = append(replacements, replacement)
			}

			if len(replacements) > 0 {
				// log.Println("Applying replacements: ")
				// for _, replacement := range replacements {
				// 	log.Println(replacement.Id)
				// }

				var allSucceeded bool
				updated, allSucceeded = ApplyReplacements(updated, replacements, false)

				if !allSucceeded {
					return nil, fmt.Errorf("plan replacement failed - %s", path)
				}

				// log.Println("Updated content: ")
				// log.Println(updated)

				updatedAtByPath[path] = planRes.CreatedAt
			}

			if foundTarget {
				break PlanResLoop
			}
		}

		// log.Println("Setting updated content for path: ", path)

		files[path] = updated
	}

	return &CurrentPlanFiles{Files: files, UpdatedAtByPath: updatedAtByPath}, nil
}
