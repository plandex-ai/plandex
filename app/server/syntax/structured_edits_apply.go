package syntax

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	sitter "github.com/smacker/go-tree-sitter"
)

const duplicationThreshold = 20

type Reference int
type Removal int

type Anchor int

type NeedsVerifyReason string

const (
	NeedsVerifyReasonCodeRemoved       NeedsVerifyReason = "code_removed"
	NeedsVerifyReasonCodeDuplicated    NeedsVerifyReason = "code_duplicated"
	NeedsVerifyReasonAmbiguousLocation NeedsVerifyReason = "ambiguous_location"
)

type ApplyChangesResult struct {
	NewFile            string
	Proposed           string
	NeedsVerifyReasons []NeedsVerifyReason
}

type AnchorMap = map[int]int

type ReferenceBlock struct {
	start int // Start line number in original file (inclusive)
	end   int // End line number in original file (inclusive)
}

const verboseLogging = false

func isRef(content string) bool {
	trimmedLower := strings.ToLower(strings.TrimSpace(content))
	if strings.Contains(trimmedLower, "... existing code ...") {
		return true
	}
	regex := regexp.MustCompile(`(\.\.\.)?.*?existing.*?\.\.\.$`)
	return regex.MatchString(trimmedLower)
}

func isRemoval(content string) bool {
	return strings.Contains(strings.ToLower(content), "plandex: removed")
}

func ApplyChanges(
	original,
	proposed,
	desc string,
	addMissingStartEndRefs bool,
	parser *sitter.Parser,
	language shared.TreeSitterLanguage,
	ctx context.Context,
) *ApplyChangesResult {
	proposedInitial := proposed

	proposedLines := strings.Split(proposed, "\n")
	originalLines := strings.Split(original, "\n")

	var references []Reference
	hasRefByLine := map[int]bool{}

	var removals []Removal
	hasRemovalByLine := map[int]bool{}

	for i, line := range proposedLines {
		lineNum := i + 1
		content := strings.TrimSpace(line)
		found := false
		if isRef(content) {
			if !hasRefByLine[lineNum] {
				references = append(references, Reference(lineNum))
				hasRefByLine[lineNum] = true
			}
			found = true
		} else if isRemoval(content) {
			if !hasRemovalByLine[lineNum] {
				removals = append(removals, Removal(lineNum))
				hasRemovalByLine[lineNum] = true
			}
			found = true
		}

		if found {
			proposedLines[i] = strings.Replace(proposedLines[i], content, "", 1)
		}
	}

	if addMissingStartEndRefs {
		var beginsWithRef bool = false
		var endsWithRef bool = false
		var foundNonRefLine bool = false

		for i, line := range proposedLines {
			hasRef := hasRefByLine[i+1] || hasRemovalByLine[i+1]

			if hasRef {
				if !foundNonRefLine {
					beginsWithRef = true
				}
				endsWithRef = true
			} else if line != "" {
				foundNonRefLine = true
				endsWithRef = false
			}
		}

		if !beginsWithRef &&
			!strings.Contains(desc, "entire file") &&
			!((strings.Contains(desc, "replace code") ||
				strings.Contains(desc, "remove code")) &&
				strings.Contains(desc, "start of the file")) {

			if verboseLogging {
				fmt.Println("adding ... existing code ... to start of file")
			}

			proposedLines = append([]string{""}, proposedLines...)

			// bump all existing references up by 1
			for i, ref := range references {
				references[i] = Reference(int(ref) + 1)
			}
			references = append([]Reference{Reference(1)}, references...)
		}

		if !endsWithRef &&
			!strings.Contains(desc, "entire file") &&
			!((strings.Contains(desc, " replace ") ||
				strings.Contains(desc, " remove ")) &&
				strings.Contains(desc, "end of the file")) {

			if verboseLogging {
				fmt.Println("adding ... existing code ... to end of file")
			}

			proposedLines = append(proposedLines, "")
			references = append(references, Reference(len(proposedLines)))
		}
	}

	proposed = strings.Join(proposedLines, "\n")

	regex := regexp.MustCompile(`(I'll|I will) (add|insert|append|prepend)`)
	isInsert := regex.MatchString(desc)

	if isInsert {
		if verboseLogging {
			fmt.Println("isInsert")
		}
	}

	// if verboseLogging {
	// fmt.Println("proposed:")
	// fmt.Println(proposed)
	// log.Println("ApplyChanges - references:")
	// spew.Dump(references)
	// log.Println("ApplyChanges - removals:")
	// spew.Dump(removals)
	// }

	res := ExecApplyGeneric(
		execApplyGenericParams{
			original:      original,
			proposed:      proposed,
			originalLines: originalLines,
			proposedLines: proposedLines,
			references:    references,
			removals:      removals,
			isInsert:      isInsert,
		},
	)

	res.Proposed = proposed

	if len(res.NeedsVerifyReasons) > 0 {
		log.Println("ApplyChanges - needs verify reasons:")
		log.Println(spew.Sdump(res.NeedsVerifyReasons))

		log.Println("ApplyChanges - proposed:")
		log.Println(proposedInitial)
		log.Println("--------------------------------")

		// log.Println("ApplyChanges - original:")
		// log.Println(original)
		// log.Println("--------------------------------")

		if len(res.NeedsVerifyReasons) == 1 && res.NeedsVerifyReasons[0] == NeedsVerifyReasonAmbiguousLocation {
			var err error
			prevRes := res
			res, err = ExecApplyTreeSitter(
				execApplyTreeSitterParams{
					original:   original,
					proposed:   proposed,
					references: references,
					removals:   removals,
					language:   language,
					parser:     parser,
					ctx:        ctx,
				},
			)

			if err != nil {
				log.Printf("ApplyChanges - error applying tree-sitter: %v", err)
				// since we got an error, give up and go back to the previous result
				res = prevRes
			} else if len(res.NeedsVerifyReasons) > 0 {
				return res
			}
		}
	}

	if !strings.Contains(desc, "entire file") {
		if verboseLogging {
			log.Println("ApplyChanges - checking for removed lines")
		}

		originalLineMap := make(map[string]bool)
		for _, line := range originalLines {
			originalLineMap[strings.TrimSpace(line)] = true
		}

		newLines := strings.Split(res.NewFile, "\n")
		newLineMap := make(map[string]bool)
		for _, line := range newLines {
			newLineMap[strings.TrimSpace(line)] = true
		}

		// Check for removed lines (lines in original that are not in new)
		for line := range originalLineMap {
			if !newLineMap[line] {
				log.Println("ApplyChanges - code removed")
				log.Println("line:")
				log.Println(line)
				res.NeedsVerifyReasons = append(res.NeedsVerifyReasons, NeedsVerifyReasonCodeRemoved)
				break
			}
		}

		if strings.Contains(desc, " replace ") {
			if verboseLogging {
				log.Println("ApplyChanges - checking for duplicated lines")
			}

			// Check for lines in proposed updates that are duplicated in new file
			newLineFreq := make(map[string]int)
			originalLineFreq := make(map[string]int)
			proposedLineFreq := make(map[string]int)

			// First count frequencies in original file
			for _, line := range originalLines {
				trimmed := strings.TrimSpace(line)
				if len(trimmed) > duplicationThreshold {
					originalLineFreq[line]++
				}
			}

			// Count frequencies in proposed file
			for _, line := range proposedLines {
				trimmed := strings.TrimSpace(line)
				if len(trimmed) > duplicationThreshold {
					proposedLineFreq[line]++
				}
			}

			// Count frequencies in new file
			for _, line := range newLines {
				trimmed := strings.TrimSpace(line)
				if len(trimmed) > duplicationThreshold {
					newLineFreq[line]++
				}
			}

			// Check proposed lines against new frequencies, accounting for original duplicates
			for _, line := range proposedLines {
				trimmed := strings.TrimSpace(line)
				if len(trimmed) > duplicationThreshold {
					originalCount := originalLineFreq[line]
					proposedCount := proposedLineFreq[line]
					newCount := newLineFreq[line]
					if newCount > originalCount && newCount > proposedCount {
						log.Println("ApplyChanges - code duplicated")
						log.Println("line:")
						log.Println(line)
						log.Printf("original occurrences: %d, new occurrences: %d", originalCount, newLineFreq[line])
						res.NeedsVerifyReasons = append(res.NeedsVerifyReasons, NeedsVerifyReasonCodeDuplicated)
						break
					}
				}
			}
		}
	}

	return res
}
