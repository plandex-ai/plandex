package syntax

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	shared "plandex-shared"

	"github.com/davecgh/go-spew/spew"
	sitter "github.com/smacker/go-tree-sitter"
)

// const duplicationThreshold = 20

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
	BlocksRemoved      []struct {
		Start   int
		End     int
		Content string
	}
	// Comments           []Comment
}

type AnchorMap = map[int]int

type ReferenceBlock struct {
	start int // Start line number in original file (inclusive)
	end   int // End line number in original file (inclusive)
}

type Comment struct {
	Txt   string
	IsRef bool
}

type RemovalRange struct {
	Start int
	End   int
}

func (r RemovalRange) Overlaps(other RemovalRange) bool {
	return (other.Start <= r.End && other.Start >= r.Start) ||
		(other.End <= r.End && other.End >= r.Start) ||
		(other.Start <= r.Start && other.End >= r.End)
}

const verboseLogging = false

func isRef(content string) bool {
	trimmedLower := strings.ToLower(strings.TrimSpace(content))
	if strings.Contains(trimmedLower, "... existing code ...") {
		return true
	}
	regex := regexp.MustCompile(`(\.\.\.)?.*?(existing|rest of|start of|end of).*?\.\.\.$`)
	return regex.MatchString(trimmedLower)
}

func isRemoval(content string) bool {
	return strings.Contains(strings.ToLower(content), "plandex: removed")
}

type ApplyChangesParams struct {
	Original               string
	Proposed               string
	Desc                   string
	AddMissingStartEndRefs bool
	Parser                 *sitter.Parser
	Language               shared.Language
}

func ApplyChanges(
	ctx context.Context,
	params ApplyChangesParams,
) *ApplyChangesResult {
	original := params.Original
	proposed := params.Proposed
	desc := params.Desc
	addMissingStartEndRefs := params.AddMissingStartEndRefs
	parser := params.Parser
	language := params.Language

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

	desc = strings.ToLower(desc)
	desc = strings.TrimSpace(desc)
	desc = strings.ReplaceAll(desc, "*", "")
	desc = strings.ReplaceAll(desc, "`", "")
	desc = strings.ReplaceAll(desc, "'", "")
	desc = strings.ReplaceAll(desc, `"`, "")

	isEntireFileUpdate := strings.Contains(desc, "type: overwrite")
	isReplace := strings.Contains(desc, "type: replace")
	isRemove := strings.Contains(desc, "type: remove")
	isAdd := strings.Contains(desc, "type: add")
	isPrepend := strings.Contains(desc, "type: prepend")
	isAppend := strings.Contains(desc, "type: append")

	var removalRanges []RemovalRange

	// Parse line ranges from summary field
	// Matches formats like:
	// Replace: line 10
	// Remove: line 10
	// Replace: lines 10-20
	// Remove: lines 10-20
	singleLineRegex := regexp.MustCompile(`(?i)(Replace|Replaces|Remove|Removes).*?(\d+)`)
	lineRangeRegex := regexp.MustCompile(`(?i)(Replace|Replaces|Remove|Removes).*?(\d+)-(\d+)`)

	descLines := strings.Split(desc, "\n")

	for _, line := range descLines {
		line = strings.TrimSpace(line)

		matchesRemaining := true
		for matchesRemaining {
			// first see if there's a multi-line match
			if lineRangeMatch := lineRangeRegex.FindStringSubmatch(strings.ToLower(line)); len(lineRangeMatch) > 3 {
				// we have a multi-line match
				start, startErr := strconv.Atoi(lineRangeMatch[2])
				end, endErr := strconv.Atoi(lineRangeMatch[3])
				if startErr == nil && endErr == nil {
					removalRanges = append(removalRanges, RemovalRange{
						Start: start,
						End:   end,
					})
					line = strings.Replace(line, fmt.Sprintf("%d-%d", start, end), "", 1)
				} else {
					log.Println("ApplyChanges - multi-line match error")
					log.Println(spew.Sdump(startErr))
					log.Println(spew.Sdump(endErr))
					matchesRemaining = false
				}
			} else {

				// try single-line match
				if singleLineMatch := singleLineRegex.FindStringSubmatch(strings.ToLower(line)); len(singleLineMatch) > 2 {
					if lineNum, err := strconv.Atoi(singleLineMatch[2]); err == nil {
						removalRanges = append(removalRanges, RemovalRange{
							Start: lineNum,
							End:   lineNum,
						})
						line = strings.Replace(line, singleLineMatch[2], "", 1)
					} else {
						log.Println("ApplyChanges - single-line match error")
						log.Println(spew.Sdump(err))
						matchesRemaining = false
					}
				} else {
					matchesRemaining = false
				}
			}
		}
	}

	if verboseLogging {
		log.Println("ApplyChanges - removal ranges:")
		log.Println(spew.Sdump(removalRanges))
	}

	var res *ApplyChangesResult

	if isEntireFileUpdate {
		hasRefsOrRemovals := len(references) > 0 || len(removals) > 0

		if !hasRefsOrRemovals {
			// shortcut to just return the full updated file and skip verification
			// if any references were included or the first and last lines of the proposed file don't match the first and last lines of the original file
			res = &ApplyChangesResult{
				NewFile:  proposed,
				Proposed: proposed,
			}
		}
	}

	if res == nil {
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
				!isEntireFileUpdate &&
				!((isReplace || isRemove) &&
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
				!isEntireFileUpdate &&
				!((isReplace || isRemove) &&
					strings.Contains(desc, "end of the file")) {

				if verboseLogging {
					fmt.Println("adding ... existing code ... to end of file")
				}

				proposedLines = append(proposedLines, "")
				references = append(references, Reference(len(proposedLines)))
			}
		}

		proposed = strings.Join(proposedLines, "\n")

		isPureInsert := !isEntireFileUpdate && !isReplace && !isRemove && (isAdd || isPrepend || isAppend)

		if isPureInsert {
			if verboseLogging {
				fmt.Println("isPureInsert")
			}
		}

		if verboseLogging {
			fmt.Println("proposed:")
			fmt.Println(proposed)
			log.Println("ApplyChanges - references:")
			spew.Dump(references)
			log.Println("ApplyChanges - removals:")
			spew.Dump(removals)
		}

		if !isPureInsert {
			log.Println("ApplyChanges - removal ranges:")
			log.Println(spew.Sdump(removalRanges))
		}

		res = ExecApplyGeneric(
			execApplyGenericParams{
				original:      original,
				proposed:      proposed,
				originalLines: originalLines,
				proposedLines: proposedLines,
				references:    references,
				removals:      removals,
				isInsert:      isPureInsert,
				removalRanges: removalRanges,
			},
		)

		res.Proposed = proposed

		if len(res.NeedsVerifyReasons) > 0 {
			if verboseLogging {
				log.Println("ApplyChanges - needs verify reasons:")
				log.Println(spew.Sdump(res.NeedsVerifyReasons))

				log.Println("ApplyChanges - proposed:")
				log.Println(proposedInitial)
				log.Println("--------------------------------")

				// log.Println("ApplyChanges - original:")
				// log.Println(original)
				// log.Println("--------------------------------")
			}

			if len(res.NeedsVerifyReasons) == 1 && res.NeedsVerifyReasons[0] == NeedsVerifyReasonAmbiguousLocation && parser != nil {
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
	}

	// we want to verify if any code was removed/replaced based on length comparison or description
	// check if code was removed based on length comparison
	if len(res.NewFile) < len(original) {
		log.Println("ApplyChanges - code removed based on length comparison - needs verification")
		res.NeedsVerifyReasons = append(res.NeedsVerifyReasons, NeedsVerifyReasonCodeRemoved)
	} else if isRemove || isReplace || isEntireFileUpdate {
		log.Println("ApplyChanges - code removed based on description - needs verification")
		res.NeedsVerifyReasons = append(res.NeedsVerifyReasons, NeedsVerifyReasonCodeRemoved)
	} else {
		if verboseLogging {
			log.Println("ApplyChanges - checking for removed lines")
		}

		originalLineMap := make(map[string]bool)
		for _, line := range originalLines {
			originalLineMap[strings.TrimSpace(line)] = true
		}

		// log.Println("NewFile:")
		// log.Println(strconv.Quote(res.NewFile))

		newLines := strings.Split(res.NewFile, "\n")
		newLineMap := make(map[string]bool)
		for _, line := range newLines {
			newLineMap[strings.TrimSpace(line)] = true
		}

		log.Println("ApplyChanges - originalLineMap:")
		spew.Dump(originalLineMap)
		log.Println("ApplyChanges - newLineMap:")
		spew.Dump(newLineMap)

		// Check for removed lines (lines in original that are not in new)
		for line := range originalLineMap {
			if !newLineMap[line] {
				log.Println("ApplyChanges - code removed based on line comparison - needs verification")
				log.Println("line:")
				log.Println(line)

				res.NeedsVerifyReasons = append(res.NeedsVerifyReasons, NeedsVerifyReasonCodeRemoved)
				break
			}
		}
	}

	// Duplication check is now redundant since we verify all replacements
	// if strings.Contains(desc, " replace ") {
	// 	if verboseLogging {
	// 		log.Println("ApplyChanges - checking for duplicated lines")
	// 	}
	//
	// 	// Check for lines in proposed updates that are duplicated in new file
	// 	newLineFreq := make(map[string]int)
	// 	originalLineFreq := make(map[string]int)
	// 	proposedLineFreq := make(map[string]int)
	//
	// 	// First count frequencies in original file
	// 	for _, line := range originalLines {
	// 		trimmed := strings.TrimSpace(line)
	// 		if len(trimmed) > duplicationThreshold {
	// 			originalLineFreq[line]++
	// 		}
	// 	}
	//
	// 	// Count frequencies in proposed file
	// 	for _, line := range proposedLines {
	// 		trimmed := strings.TrimSpace(line)
	// 		if len(trimmed) > duplicationThreshold {
	// 			proposedLineFreq[line]++
	// 		}
	// 	}
	//
	// 	// Count frequencies in new file
	// 	for _, line := range newLines {
	// 		trimmed := strings.TrimSpace(line)
	// 		if len(trimmed) > duplicationThreshold {
	// 			newLineFreq[line]++
	// 		}
	// 	}
	//
	// 	// Check proposed lines against new frequencies, accounting for original duplicates
	// 	for _, line := range proposedLines {
	// 		trimmed := strings.TrimSpace(line)
	// 		if len(trimmed) > duplicationThreshold {
	// 			originalCount := originalLineFreq[line]
	// 			proposedCount := proposedLineFreq[line]
	// 			newCount := newLineFreq[line]
	// 			if newCount > originalCount && newCount > proposedCount {
	// 				if verboseLogging {
	// 					log.Println("ApplyChanges - code duplicated")
	// 					log.Println("line:")
	// 					log.Println(line)
	// 					log.Printf("original occurrences: %d, new occurrences: %d", originalCount, newLineFreq[line])
	// 				}
	// 				res.NeedsVerifyReasons = append(res.NeedsVerifyReasons, NeedsVerifyReasonCodeDuplicated)
	// 				break
	// 			}
	// 		}
	// 	}
	// }

	// if parser != nil {
	// 	comments, err := FindComments(ctx, parser, proposed)
	// 	if err != nil {
	// 		log.Printf("ApplyChanges - error finding comments: %v", err)
	// 	}
	// 	res.Comments = comments
	// }

	if verboseLogging && len(res.BlocksRemoved) > 0 {
		log.Printf("ApplyChanges - detected %d removed code blocks", len(res.BlocksRemoved))
		for i, block := range res.BlocksRemoved {
			log.Printf("Block %d: lines %d-%d", i+1, block.Start, block.End)
			if verboseLogging {
				log.Printf("Content: %s", block.Content)
			}
		}
	}

	return res
}
