package shared

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
)

func ApplyReplacements(content string, replacements []*Replacement, setFailed bool) (string, bool) {
	return applyReplacements(content, replacements, setFailed, false)
}

func ApplyReplacementsVerbose(content string, replacements []*Replacement, setFailed bool) (string, bool) {
	return applyReplacements(content, replacements, setFailed, true)
}

func applyReplacements(content string, replacements []*Replacement, setFailed, verbose bool) (string, bool) {
	apply := func(replacements []*Replacement) (string, int) {
		if verbose {
			log.Println("Applying replacements")
			log.Println("original content:\n", content)
		}

		updated := content

		lastInsertedIdx := 0

		for i, replacement := range replacements {
			if verbose {
				log.Println("replacement.Old:\n", replacement.Old)
				log.Println("updated:\n", updated)
				log.Println("lastInsertedIdx:", lastInsertedIdx)
			}

			pre := updated[:lastInsertedIdx]
			sub := updated[lastInsertedIdx:]

			var originalIdx int

			if replacement.EntireFile {
				originalIdx = 0
			} else {
				originalIdx = strings.Index(sub, replacement.Old)
			}

			if verbose {
				log.Println("originalIdx:", originalIdx)
			}

			// only for use with full replacements, which we aren't using now
			// if originalIdx == -1 {
			// 	originalIdx = getUniqueFuzzyIndex(updated, replacement.Old)
			// }

			if originalIdx == -1 {
				if setFailed {
					replacement.Failed = true
				}

				log.Println("Replacement failed at index:", i)
				log.Println("replacement.Old:")
				log.Println(replacement.Old)

				log.Println("Updated:")
				log.Println(updated)

				if verbose {
					log.Println("All replacements:")
					log.Println(spew.Sdump(replacements))
				}

				return updated, i

			} else if replacement.EntireFile {
				updated = replacement.New
				lastInsertedIdx = 0
			} else {
				if verbose {
					log.Printf("originalIdx: %d, len(replacement.Old): %d\n", originalIdx, len(replacement.Old))
					log.Println("Old: ", replacement.Old)
					log.Println("New: ", replacement.New)
				}
				replaced := strings.Replace(sub, replacement.Old, replacement.New, 1)

				if verbose {
					log.Println("replaced:")
					log.Println(replaced)
				}

				updated = pre + replaced

				if verbose {
					log.Printf("lastInsertedIdx: %d, originalIdx: %d, len(replacement.New): %d\n", lastInsertedIdx, originalIdx, len(replacement.New))
					log.Println("updated after replacement:")
					log.Println(updated)
				}

				lastInsertedIdx = lastInsertedIdx + originalIdx + len(replacement.New)
			}
		}

		return updated, -1
	}

	res, failedAtIndex := apply(replacements)

	return res, failedAtIndex == -1

	// for {

	// 	if failedAtIndex == 0 {
	// 		return res, false
	// 	} else if failedAtIndex > 0 {
	// 		// check if there's overlap between the failed replacement and the previous replacement
	// 		// if there is, remove the previous one and try again
	// 		failed := replacements[failedAtIndex]
	// 		prev := replacements[failedAtIndex-1]

	// 		hasOverlap := failed.StreamedChange.Old.StartLine <= prev.StreamedChange.Old.EndLine

	// 		if hasOverlap {
	// 			replacements = append(replacements[:failedAtIndex-1], replacements[failedAtIndex:]...)

	// 			continue
	// 		} else {
	// 			return res, false
	// 		}

	// 	} else {
	// 		return res, true
	// 	}
	// }

}

func (planState *CurrentPlanState) GetFiles() (*CurrentPlanFiles, error) {
	return planState.GetFilesBeforeReplacement("")
}

func (planState *CurrentPlanState) GetFilesBeforeReplacement(
	replacementId string,
) (*CurrentPlanFiles, error) {
	// log.Println("GetFilesBeforeReplacement")

	planRes := planState.PlanResult

	files := make(map[string]string)
	shas := make(map[string]string)
	updatedAtByPath := make(map[string]time.Time)
	removedByPath := make(map[string]bool)

	for path, planResults := range planRes.FileResultsByPath {
		updated := files[path]
		// log.Println("path: ", path)

		// spew.Dump(planResults)
		// log.Println("before PlanResLoop updated:")
		// log.Println(updated)

	PlanResLoop:
		for _, planRes := range planResults {

			// log.Println("planRes: ", planRes.Id)
			// log.Println(spew.Sdump(planRes))

			if !planRes.IsPending() {
				// log.Println("Plan result is not pending -- continuing loop")
				continue
			}

			if planRes.RemovedFile {
				updated = ""
				delete(files, path)
				delete(shas, path)
				delete(updatedAtByPath, path)
				removedByPath[path] = true
				continue
			}

			if len(planRes.Replacements) == 0 {
				if updated != "" {
					log.Println("plan updates out of order:", path)
					log.Println("updated:")
					log.Println(updated)
					log.Println("planRes.Content:")
					log.Println(planRes.Content)
					return nil, fmt.Errorf("plan updates out of order: %s", path)
				}

				updated = planRes.Content
				files[path] = updated
				updatedAtByPath[path] = planRes.CreatedAt
				delete(removedByPath, path)

				continue
			} else if updated == "" {
				context := planState.ContextsByPath[path]

				if context == nil {
					// spew.Dump(planRes)

					return nil, fmt.Errorf("no context for path: %s", path)
				}

				// log.Println("No updated content -- setting to context body")

				updated = context.Body
				shas[path] = context.Sha

				// log.Println("setting updated content to context body")
				// log.Println(updated)
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

				var allSucceeded bool

				maybeWithLineNums := updated
				if planRes.ReplaceWithLineNums {
					maybeWithLineNums = string(AddLineNums(maybeWithLineNums))
				}

				// log.Println("Before replacements. updated:")
				// log.Println(updated)

				updated, allSucceeded = ApplyReplacements(maybeWithLineNums, replacements, false)

				updated = string(RemoveLineNums(LineNumberedTextType(updated)))

				if !allSucceeded {
					log.Println("Failed to apply replacements")

					// log.Println("replacements:")
					// log.Println(spew.Sdump(replacements))

					// log.Println("updated:")
					// log.Println(updated)

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

	return &CurrentPlanFiles{Files: files, UpdatedAtByPath: updatedAtByPath, Removed: removedByPath}, nil
}
