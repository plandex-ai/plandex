package shared

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
)

func ApplyReplacements(content string, replacements []*Replacement, setFailed bool) (string, bool) {
	apply := func(replacements []*Replacement) (string, int) {
		updated := content
		lastInsertedIdx := 0

		for i, replacement := range replacements {
			// log.Println("replacement.Old:\n", replacement.Old)
			// log.Println("updated:\n", updated)
			// log.Println("lastInsertedIdx:", lastInsertedIdx)

			pre := updated[:lastInsertedIdx]
			sub := updated[lastInsertedIdx:]

			var originalIdx int

			if replacement.EntireFile {
				originalIdx = 0
			} else {
				originalIdx = strings.Index(sub, replacement.Old)
			}

			// log.Println("originalIdx:", originalIdx)

			// only for use with full replacements, which we aren't using now
			// if originalIdx == -1 {
			// 	originalIdx = getUniqueFuzzyIndex(updated, replacement.Old)
			// }

			if originalIdx == -1 {
				if setFailed {
					replacement.Failed = true
				}

				log.Println("Replacement failed at index:", i)
				log.Println("Replacement:")
				log.Println(spew.Sdump(replacement))

				log.Println("Updated:")
				log.Println(updated)

				log.Println("All replacements:")
				log.Println(spew.Sdump(replacements))

				return updated, i

			} else if replacement.EntireFile {
				updated = replacement.New
				lastInsertedIdx = 0
			} else {
				// log.Printf("originalIdx: %d, len(replacement.Old): %d\n", originalIdx, len(replacement.Old))
				// log.Println("Old: ", replacement.Old)
				// log.Println("New: ", replacement.New)
				replaced := strings.Replace(sub, replacement.Old, replacement.New, 1)

				// log.Println("replaced:")
				// log.Println(replaced)

				updated = pre + replaced

				// log.Printf("lastInsertedIdx: %d, originalIdx: %d, len(replacement.New): %d\n", lastInsertedIdx, originalIdx, len(replacement.New))

				// log.Println("updated after replacement:")
				// log.Println(updated)

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

					spew.Dump(planRes)

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

				maybeWithLineNums := updated
				if planRes.ReplaceWithLineNums {
					maybeWithLineNums = AddLineNums(updated)
				}

				updated, allSucceeded = ApplyReplacements(maybeWithLineNums, replacements, false)

				updated = RemoveLineNums(updated)

				if !allSucceeded {
					log.Println("updated:")
					log.Println(updated)

					log.Println("replacements:")
					log.Println(spew.Sdump(replacements))

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

// func getUniqueFuzzyIndex(doc, s string) int {
// 	words := strings.Fields(s)
// 	if len(words) < 2 {
// 		return -1
// 	}

// 	searchStart := regexp.QuoteMeta(words[0])
// 	searchEnd := regexp.QuoteMeta(words[len(words)-1])

// 	// Loop to expand search pattern gradually from both ends towards the center
// 	for i := 0; i < len(words)/2; i++ { // Ensure we do not go out of bounds
// 		// Construct the regex pattern
// 		regexPattern := fmt.Sprintf("%s.*%s", searchStart, searchEnd)
// 		re, err := regexp.Compile(regexPattern)
// 		if err != nil {
// 			return -1 // Handle regex compilation error
// 		}

// 		// Find all matches of the pattern in the document
// 		matches := re.FindAllStringIndex(doc, -1)
// 		if len(matches) == 1 {
// 			return matches[0][0] // Return the start index of the unique match
// 		}

// 		// Update search strings if possible
// 		if i+1 < len(words)/2 {
// 			searchStart += " " + regexp.QuoteMeta(words[i+1])
// 			searchEnd = regexp.QuoteMeta(words[len(words)-i-2]) + " " + searchEnd
// 		}
// 	}

// 	return -1
// }
