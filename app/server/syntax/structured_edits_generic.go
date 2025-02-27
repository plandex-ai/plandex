package syntax

import (
	"fmt"
	"sort"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

type execApplyGenericParams struct {
	original,
	proposed string
	originalLines,
	proposedLines []string
	references    []Reference
	removals      []Removal
	isInsert      bool
	removalRanges []RemovalRange
}

func ExecApplyGeneric(
	params execApplyGenericParams,
) *ApplyChangesResult {
	originalLines := params.originalLines
	proposedLines := params.proposedLines
	references := params.references
	removals := params.removals
	isInsert := params.isInsert
	removalRanges := params.removalRanges

	res := &ApplyChangesResult{}

	var b strings.Builder

	write := func(s string, newline bool) {
		if verboseLogging {
			toLog := s

			if len(toLog) > 200 {
				toLog = toLog[:100] + "\n...\n" + toLog[len(toLog)-100:]
			}

			fmt.Printf("writing: %s\n", toLog)
			fmt.Printf("newline: %v\n", newline)
		}

		b.WriteString(s)
		if newline {
			b.WriteByte('\n')
		}
	}

	refsByLine := map[Reference]bool{}
	removalsByLine := map[Removal]bool{}

	for _, ref := range references {
		refsByLine[ref] = true
	}

	for _, removal := range removals {
		removalsByLine[removal] = true
	}

	anchorMap := buildAnchorMap(
		originalLines,
		proposedLines,
		refsByLine,
		removalsByLine,
	)

	if verboseLogging {
		fmt.Printf("anchorMap:\n%v\n", spew.Sdump(anchorMap))
	}

	findAnchor := func(pLineNum int) *Anchor {
		oLineNum, ok := anchorMap[pLineNum]
		if ok {
			if verboseLogging {
				fmt.Printf("found anchor in anchorLines: oLineNum: %d, pLineNum: %d\n", oLineNum, pLineNum)
			}

			oLine := originalLines[oLineNum-1]
			if strings.TrimSpace(oLine) == "" {
				if verboseLogging {
					fmt.Printf("skipping anchor because oLine is blank: %q\n", oLine)
				}
				return nil
			}

			anchor := Anchor(oLineNum)
			if verboseLogging {
				fmt.Printf("found anchor: %d\n", anchor)
			}
			return &anchor
		} else {
			if verboseLogging {
				fmt.Printf("no anchor found in anchorMap: pLineNum: %d\n", pLineNum)
				fmt.Printf("anchorMap: %v\n", spew.Sdump(anchorMap))

				// for i, line := range originalLines {
				// 	fmt.Printf("originalLines[%d]: %q\n", i, line)
				// }

				// for i, line := range proposedLines {
				// 	fmt.Printf("proposedLines[%d]: %q\n", i, line)
				// }
			}
		}

		return nil
	}

	var oLineNum int = 0
	var refOpen bool
	var refStart int
	var postRefBuffers []strings.Builder

	lastLineMatched := true

	setOLineNum := func(n int) {
		if n < 1 {
			n = 1
		}
		if n > len(originalLines) {
			n = len(originalLines)
		}
		oLineNum = n
		if verboseLogging {
			fmt.Printf("setting oLineNum: %d\n", oLineNum)
		}
	}

	writeToLatestPostRefBuffer := func(s string) {
		latestBuffer := &postRefBuffers[len(postRefBuffers)-1]
		latestBuffer.WriteString(s)
		latestBuffer.WriteByte('\n')

		if verboseLogging {
			fmt.Printf("writing to latest postRefBuffer: %q\n", s)
		}
	}

	addNewPostRefBuffer := func() {
		postRefBuffers = append(postRefBuffers, strings.Builder{})

		if verboseLogging {
			fmt.Printf("adding new postRefBuffer\n")
		}
	}

	resetPostRefBuffers := func() {
		postRefBuffers = []strings.Builder{}

		if verboseLogging {
			fmt.Printf("resetting postRefBuffers\n")
		}
	}

	writeRefs := func(eof bool) bool {
		var fullRef []string
		if eof {
			start := refStart - 1
			if start < 0 {
				start = 0
			}
			if start >= len(originalLines) {
				start = len(originalLines) - 1
			}
			fullRef = originalLines[start:]
			if verboseLogging {
				fmt.Println("eof")
				fmt.Printf("fullRef refStart: %d\n", refStart)
				fmt.Printf("originalLines[refStart]: %q\n", originalLines[refStart])
				fmt.Printf("writing eof fullRef: %q\n", strings.Join(fullRef, "\n"))
			}
		} else {
			start := refStart - 1
			if start < 0 {
				start = 0
			}
			if start >= len(originalLines) {
				start = len(originalLines) - 1
			}
			end := oLineNum - 1
			if end < 1 {
				end = 0
			}
			if end >= len(originalLines) {
				end = len(originalLines) - 1
			}

			// Add detailed diagnostic logging for invalid slice bounds
			if start > end {
				fmt.Printf("\n=== INVALID SLICE BOUNDS DIAGNOSTIC INFO ===\n")
				fmt.Printf("start: %d, end: %d\n", start, end)
				fmt.Printf("refStart: %d, oLineNum: %d\n", refStart, oLineNum)

				// Log relevant lines for context
				fmt.Printf("\nOriginal lines context:\n")
				startContext := max(0, start-2)
				endContext := min(len(originalLines), end+3)
				for i := startContext; i < endContext; i++ {
					fmt.Printf("line %d: %q\n", i+1, originalLines[i])
				}

				fmt.Printf("\nProposed lines context:\n")
				fmt.Printf("=====================================\n\n")

				res.NeedsVerifyReasons = append(res.NeedsVerifyReasons, NeedsVerifyReasonAmbiguousLocation)

				return true
			}

			if verboseLogging {
				fmt.Printf("writing fullRef\n")
				fmt.Printf("refStart: %d, oLineNum: %d\n", refStart, oLineNum)
				fmt.Printf("start: %d, end: %d\n", start, end)
			}
			fullRef = originalLines[start:end]
			if verboseLogging {
				fmt.Printf("fullRef refStart: %d, oLineNum-1: %d\n", refStart, oLineNum-1)
				fmt.Printf("originalLines[start]: %q\n", originalLines[start])
				fmt.Printf("originalLines[end]: %q\n", originalLines[end])
			}
		}

		numRefs := len(postRefBuffers)
		if numRefs == 1 {
			if verboseLogging {
				fmt.Println("writeRefs")
				fmt.Printf("numRefs == 1, refStart: %d, oLineNum: %d\n", refStart, oLineNum)
			}

			write(strings.Join(fullRef, "\n"), !eof)

			postRefContent := postRefBuffers[0].String()

			if strings.TrimSpace(postRefContent) != "" {
				if verboseLogging {
					fmt.Println("writing postRefBuffer")
				}
				write(postRefBuffers[0].String(), false)
			}
		} else {
			if verboseLogging {
				fmt.Printf("writeRefs - ambiguous location - numRefs: %d\n", numRefs)
			}

			res.NeedsVerifyReasons = append(res.NeedsVerifyReasons, NeedsVerifyReasonAmbiguousLocation)

			return true
		}

		return false
	}

	for idx, pLine := range proposedLines {
		finalLine := idx == len(proposedLines)-1
		pLineNum := idx + 1

		if verboseLogging {
			fmt.Printf("\n\ni: %d, num: %d, pLine: %q, refOpen: %v\n", idx, pLineNum, pLine, refOpen)
		}

		isRef := refsByLine[Reference(pLineNum)]
		isRemoval := removalsByLine[Removal(pLineNum)]
		nextPLineIsRef := refsByLine[Reference(pLineNum+1)]

		if verboseLogging {
			fmt.Printf("isRef: %v\n", isRef)
			fmt.Printf("isRemoval: %v\n", isRemoval)
			fmt.Printf("nextPLineIsRef: %v\n", nextPLineIsRef)
			fmt.Printf("oLineNum before: %d\n", oLineNum)
			if oLineNum > 0 && oLineNum < len(originalLines) {
				fmt.Printf("oLine before: %q\n", originalLines[oLineNum-1])
			}
			fmt.Printf("lastLineMatched: %v\n", lastLineMatched)
		}

		if isRemoval {
			if verboseLogging {
				fmt.Println("isRemoval - skip line")
			}
			continue
		}

		if isRef {
			if !refOpen {
				if verboseLogging {
					fmt.Println("isRef - opening ref")
				}

				refOpen = true
				setOLineNum(oLineNum + 1)

				if verboseLogging {
					fmt.Printf("setting refStart: %d\n", refStart)
				}

				refStart = oLineNum
			}

			addNewPostRefBuffer()

			if oLineNum == len(originalLines) {
				break
			}

			continue
		}

		if !refOpen && lastLineMatched {
			if strings.TrimSpace(pLine) == "" {
				if verboseLogging {
					fmt.Printf("pLine is blank\n")
					if oLineNum < len(originalLines) {
						fmt.Printf("nextOLine: %q\n", originalLines[oLineNum])
					}
				}

				nextOLineIsBlank := oLineNum > 0 && oLineNum < len(originalLines) && strings.TrimSpace(originalLines[oLineNum]) == ""

				if verboseLogging {
					fmt.Printf("nextPLineIsRef: %v\n", nextPLineIsRef)
					fmt.Printf("nextOLineIsBlank: %v\n", nextOLineIsBlank)
				}

				if !nextPLineIsRef || nextOLineIsBlank {
					write(pLine, !finalLine)
				}

				// Check if next line in original is also blank
				if nextOLineIsBlank {
					setOLineNum(oLineNum + 1)
				}

				if oLineNum == len(originalLines) {
					break
				}

				continue
			}
		}

		var matching bool

		prevOLineNum := oLineNum
		anchor := findAnchor(pLineNum)
		if anchor != nil {
			matching = true
			setOLineNum(int(*anchor))
		}

		wroteRefs := false
		if matching {
			if verboseLogging {
				fmt.Printf("matching line: %s, oLineNum: %d\n", pLine, oLineNum)
			}

			if refOpen {
				// we found the end of the current reference
				if verboseLogging {
					fmt.Printf("closing ref, oLineNum: %d\n", oLineNum)
				}
				refOpen = false
				willAbort := writeRefs(false)
				write(pLine, !finalLine)
				wroteRefs = true
				if willAbort {
					return res
				}
			} else if oLineNum != prevOLineNum+1 {
				if verboseLogging {
					fmt.Printf("\nExecApplyChanges - found non-adjacent anchor jump:\n")
					fmt.Printf("prevOLineNum: %d ('%s')\n", prevOLineNum, originalLines[prevOLineNum])
					fmt.Printf("oLineNum: %d ('%s')\n", oLineNum, originalLines[oLineNum-1])
					fmt.Printf("Lines that would be removed:\n")
					for i := prevOLineNum; i < oLineNum-1; i++ {
						fmt.Printf("Line %d: '%s'\n", i, originalLines[i])
					}
				}

				removalRange := RemovalRange{
					Start: prevOLineNum,
					End:   oLineNum - 1,
				}

				if isInsert {
					// Write any lines that would have been removed
					for i := prevOLineNum; i < oLineNum-1; i++ {
						write(originalLines[i], true)
					}
				} else if len(removalRanges) > 0 {
					overlapsAny := false
					for _, r := range removalRanges {
						if removalRange.Overlaps(r) {
							overlapsAny = true
							break
						}
					}

					// if the removal doesn't overlap with any of the listed removal ranges, we can deterministically catch a mistake and write the lines that would have been removed
					if !overlapsAny {
						// Write any lines that would have been removed
						for i := prevOLineNum; i < oLineNum-1; i++ {
							write(originalLines[i], true)
						}
					}
				}
			}

		} else {
			if verboseLogging {
				fmt.Printf("no matching line\n")
			}
		}

		if wroteRefs {
			// reset buffers
			resetPostRefBuffers()
		} else {
			if refOpen {
				writeToLatestPostRefBuffer(pLine)
			} else {
				if verboseLogging {
					fmt.Printf("writing pLine: %s\n", pLine)
				}
				write(pLine, !finalLine)
			}

		}

		if oLineNum == len(originalLines) {
			break
		}
	}

	if refOpen {
		willAbort := writeRefs(true)
		if willAbort {
			return res
		}
	}

	if verboseLogging {
		// fmt.Printf("final result:\n%s\n", b.String())
	}

	res.NewFile = b.String()

	return res
}

func buildAnchorMap(
	originalLines []string,
	proposedLines []string,
	refsByLine map[Reference]bool,
	removalsByLine map[Removal]bool,
) AnchorMap {
	result := AnchorMap{}

	setAnchor := func(pLine, oLine int) {
		result[pLine] = oLine

		if verboseLogging {
			fmt.Printf("setAnchor - pLine: %d, oLine: %d\n", pLine, oLine)
		}
	}

	referenceBlocks := []ReferenceBlock{}

	// Helper to check if a line is in a reference block
	isInReference := func(lineNum int) bool {
		for _, block := range referenceBlocks {
			if lineNum >= block.start && lineNum <= block.end {
				return true
			}
		}
		return false
	}

	allRefsByLine := map[int]bool{}
	for ref := range refsByLine {
		allRefsByLine[int(ref)] = true
	}
	for removal := range removalsByLine {
		allRefsByLine[int(removal)] = true
	}

	// Keep track of definitely new code lines
	newCodeLines := make(map[int]bool)
	originalLinesSet := map[string]bool{}
	for _, line := range originalLines {
		originalLinesSet[line] = true
	}
	for idx, line := range proposedLines {
		if _, inOriginal := originalLinesSet[line]; !inOriginal {
			newCodeLines[idx+1] = true
		}
	}

	foundRefBounds := map[int]bool{}

	// When we establish an anchor match, check if we can determine reference bounds
	tryEstablishReferenceBounds := func() {
		if verboseLogging {
			fmt.Println("\n\ntryEstablishReferenceBounds")
		}
		for lineNum := range allRefsByLine {
			if !foundRefBounds[lineNum] {
				if verboseLogging {
					fmt.Printf("tryEstablishReferenceBounds - lineNum: %d\n", lineNum)
				}

				prevSignificantLineNum := lineNum - 1
				linesBack := 1
				for prevSignificantLineNum > 0 && proposedLines[prevSignificantLineNum-1] == "" {
					prevSignificantLineNum--
					linesBack++
				}

				nextSignificantLineNum := lineNum + 1
				linesForward := 1
				for nextSignificantLineNum <= len(proposedLines) && proposedLines[nextSignificantLineNum-1] == "" {
					nextSignificantLineNum++
					linesForward++
				}

				if verboseLogging {
					fmt.Printf("prevSignificantLineNum: %d, nextSignificantLineNum: %d\n", prevSignificantLineNum, nextSignificantLineNum)
				}

				var top, bottom int

				if prevSignificantLineNum <= 1 {
					if verboseLogging {
						fmt.Printf("prevSignificantLineNum <= 1 - setting top to 1\n")
					}
					top = 1
				} else if _, isAnchor := result[prevSignificantLineNum]; isAnchor {
					if verboseLogging {
						fmt.Printf("prevSignificantLineNum is anchor - setting top to %d\n", result[prevSignificantLineNum])
					}
					top = result[prevSignificantLineNum] + linesBack
				}

				if nextSignificantLineNum >= len(proposedLines) {
					if verboseLogging {
						fmt.Printf("nextSignificantLineNum >= len(proposedLines) - setting bottom to len(originalLines)\n")
					}
					bottom = len(originalLines)
				} else if _, isAnchor := result[nextSignificantLineNum]; isAnchor {
					if verboseLogging {
						fmt.Printf("nextSignificantLineNum is anchor - setting bottom to %d\n", result[nextSignificantLineNum])
					}
					bottom = result[nextSignificantLineNum] - linesForward
				} else if newCodeLines[nextSignificantLineNum] {

					if verboseLogging {
						fmt.Printf("nextSignificantLineNum is new code - finding next anchor\n")
					}

					// go forward from here to find the next anchor (or eof)
					foundAnchor := false
					for i := nextSignificantLineNum; i < len(proposedLines)+1; i++ {
						if _, isAnchor := result[i]; isAnchor {
							bottom = result[i] - 1
							foundAnchor = true
							if verboseLogging {
								fmt.Printf("found anchor at %d\n", i)
							}
							break
						}
					}
					if !foundAnchor {
						bottom = len(originalLines)
						if verboseLogging {
							fmt.Printf("no anchor found - setting bottom to len(originalLines)\n")
						}
					}
				}

				if top != 0 && bottom != 0 {
					foundRefBounds[lineNum] = true
					referenceBlocks = append(referenceBlocks, ReferenceBlock{start: top, end: bottom})
					if verboseLogging {
						fmt.Printf("found reference bounds: %d-%d\n", top, bottom)
					}
				} else {
					if verboseLogging {
						fmt.Printf("no reference bounds found for lineNum: %d\n", lineNum)
					}
				}
			}
		}
	}

	if verboseLogging {
		fmt.Printf("\n=== Building Anchor Map ===\n")
	}

	var matchSection func(pStart, pEnd, oStart, oEnd int)
	matchSection = func(pStart, pEnd, oStart, oEnd int) {
		if pEnd <= pStart || oEnd <= oStart {
			return
		}

		if verboseLogging {
			fmt.Printf("\n--- Processing Section ---\n")
			fmt.Printf("Proposed lines %d-%d, Original lines %d-%d\n", pStart, pEnd, oStart, oEnd)
		}

		// First find unique matches in this section
		sectionOriginal := make(map[string][]int)
		sectionProposed := make(map[string][]int)

		// Build frequency maps for this section
		for i := oStart; i < oEnd; i++ {
			content := originalLines[i]
			sectionOriginal[content] = append(sectionOriginal[content], i)
		}

		for i := pStart; i < pEnd; i++ {
			content := proposedLines[i]
			sectionProposed[content] = append(sectionProposed[content], i)
		}

		// Handle unique matches first
		if verboseLogging {
			fmt.Printf("\nProcessing unique matches...\n")
		}
		for content, pIndices := range sectionProposed {
			if oIndices, exists := sectionOriginal[content]; exists && len(oIndices) == 1 && len(pIndices) == 1 {
				pIdx, oIdx := pIndices[0], oIndices[0]
				if _, exists := result[pIdx+1]; !exists {
					setAnchor(pIdx+1, oIdx+1)
				}

				if verboseLogging {
					fmt.Printf("Found unique match: %q\n", content)
					fmt.Printf("Mapping proposed line %d (%q) -> original line %d (%q)\n",
						pIdx+1, proposedLines[pIdx], oIdx+1, originalLines[oIdx])
				}
			}
		}

		// after finding unique anchors, try to establish reference bounds
		tryEstablishReferenceBounds()

		// Get ordered anchors to establish subsections
		var orderedAnchors []struct {
			pLine int
			oLine int
		}
		for pLine := pStart; pLine < pEnd; pLine++ {
			if anchor, exists := result[pLine+1]; exists {
				orderedAnchors = append(orderedAnchors, struct {
					pLine int
					oLine int
				}{pLine, anchor - 1})
			}
		}

		if verboseLogging {
			fmt.Printf("\nOrdered anchors in section: %v\n", orderedAnchors)
		}

		// Sort anchors by proposed line number
		sort.Slice(orderedAnchors, func(i, j int) bool {
			return orderedAnchors[i].pLine < orderedAnchors[j].pLine
		})

		// Process each subsection between anchors
		lastPLine := pStart
		lastOLine := oStart

		for i := 0; i <= len(orderedAnchors); i++ {
			var nextPLine, nextOLine int
			if i < len(orderedAnchors) {
				nextPLine = orderedAnchors[i].pLine
				nextOLine = orderedAnchors[i].oLine
			} else {
				nextPLine = pEnd
				nextOLine = oEnd
			}

			if verboseLogging {
				fmt.Printf("\nProcessing subsection %d\n", i)
				fmt.Printf("Proposed lines %d-%d, Original lines %d-%d\n", lastPLine, nextPLine, lastOLine, nextOLine)
			}

			// Handle duplicates in this subsection using outside-in matching
			subSectionOriginal := make(map[string][]int)
			subSectionProposed := make(map[string][]int)

			// Build frequency maps for this subsection
			for i := lastOLine; i < nextOLine; i++ {
				content := originalLines[i]
				subSectionOriginal[content] = append(subSectionOriginal[content], i)
			}

			for i := lastPLine; i < nextPLine; i++ {
				content := proposedLines[i]
				subSectionProposed[content] = append(subSectionProposed[content], i)
			}

			// Match duplicates from outside-in
			for content, pIndices := range subSectionProposed {
				oIndices, exists := subSectionOriginal[content]
				if !exists {
					if verboseLogging {
						fmt.Printf("New code found: %q at proposed lines %v\n", content, pIndices)
					}

					continue // This is new code to be inserted
				}

				if verboseLogging {
					fmt.Printf("\nMatching duplicates for content: %q\n", content)
					fmt.Printf("Found in proposed lines: %v\n", pIndices)
					fmt.Printf("Found in original lines: %v\n", oIndices)
				}

				// Filter oIndices to only include matches within subsection boundaries
				var validOIndices []int
				for _, idx := range oIndices {
					if idx >= lastOLine && idx < nextOLine {
						validOIndices = append(validOIndices, idx)
					}
				}
				oIndices = validOIndices

				// Match from outside in until we run out of original occurrences
				matched := 0
				for matched < len(oIndices) && matched*2 < len(pIndices) {

					if isInReference(oIndices[matched] + 1) {
						if verboseLogging {
							fmt.Printf("Skipping reference line %d\n", oIndices[matched]+1)
						}
						matched++
						continue
					}

					// Match first unmatched occurrence
					if _, exists := result[pIndices[matched]+1]; !exists {
						setAnchor(pIndices[matched]+1, oIndices[matched]+1)

						if verboseLogging {
							fmt.Printf("Matched first occurrence: proposed line %d (%q) -> original line %d (%q)\n",
								pIndices[matched]+1, proposedLines[pIndices[matched]],
								oIndices[matched]+1, originalLines[oIndices[matched]])
						}

						// after finding anchor, try to establish reference bounds
						tryEstablishReferenceBounds()
					}

					// Match last unmatched occurrence if we have more to match
					if matched*2+1 < len(pIndices) {
						lastOrigIdx := oIndices[len(oIndices)-1-matched]
						lastPropIdx := pIndices[len(pIndices)-1-matched]
						if isInReference(lastOrigIdx + 1) {
							if verboseLogging {
								fmt.Printf("Skipping reference line %d\n", lastOrigIdx+1)
							}
							matched++
							continue
						}
						if _, exists := result[lastPropIdx+1]; !exists {
							setAnchor(lastPropIdx+1, lastOrigIdx+1)
							if verboseLogging {
								fmt.Printf("Matched last occurrence: proposed line %d (%q) -> original line %d (%q)\n",
									lastPropIdx+1, proposedLines[lastPropIdx],
									lastOrigIdx+1, originalLines[lastOrigIdx])
							}

							// after finding anchor, try to establish reference bounds
							tryEstablishReferenceBounds()
						}

					}
					matched++
				}
				// Any remaining occurrences in pIndices are new code to be inserted
			}

			// Recursively process the next subsection
			if i < len(orderedAnchors) {
				lastPLine = nextPLine
				lastOLine = nextOLine
			}
		}
	}

	// Start recursive matching with full file
	matchSection(0, len(proposedLines), 0, len(originalLines))

	if verboseLogging {
		fmt.Printf("\n=== Final Anchor Map ===\n")
		fmt.Printf("Result: %v\n", result)
	}

	return result
}
