package syntax

import (
	"context"
	"fmt"
	"strings"

	shared "plandex-shared"

	tree_sitter "github.com/smacker/go-tree-sitter"
)

type tsAnchor struct {
	open  int
	close int
}

type execApplyTreeSitterParams struct {
	original,
	proposed string
	references  []Reference
	removals    []Removal
	anchorLines map[int]int
	language    shared.Language
	parser      *tree_sitter.Parser
	ctx         context.Context
}

func ExecApplyTreeSitter(
	params execApplyTreeSitterParams,
) (*ApplyChangesResult, error) {
	original := params.original
	proposed := params.proposed
	references := params.references
	removals := params.removals
	anchorLines := params.anchorLines
	language := params.language
	parser := params.parser
	ctx := params.ctx
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

	originalLines := strings.Split(original, "\n")
	proposedLines := strings.Split(proposed, "\n")

	// normalize comments in case the wrong comment symbols were used
	openingCommentSymbol, closingCommentSymbol := GetCommentSymbols(language)
	if openingCommentSymbol != "" {
		for i, line := range proposedLines {
			// keep indentation for syntax parsing
			content := strings.TrimSpace(line)

			if removalsByLine[Removal(i+1)] || refsByLine[Reference(i+1)] {
				comment := openingCommentSymbol + " ref " + closingCommentSymbol
				proposedLines[i] = strings.Replace(line, content, comment, 1)
			}
		}
	}

	proposedWithNormalizedComments := strings.Join(proposedLines, "\n")
	res.Proposed = proposedWithNormalizedComments

	originalBytes := []byte(original)
	proposedBytes := []byte(proposedWithNormalizedComments)

	originalTree, err := parser.ParseCtx(ctx, nil, originalBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the original content: %v", err)
	}
	defer originalTree.Close()

	proposedTree, err := parser.ParseCtx(ctx, nil, proposedBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the proposed content: %v", err)
	}
	defer proposedTree.Close()

	if verboseLogging {
		fmt.Printf("anchorLines: %v\n", anchorLines)
	}

	oRes := BuildNodeIndex(originalTree)
	pRes := BuildNodeIndex(proposedTree)

	originalNodesByLineIndex := oRes.nodesByLine
	proposedNodesByLineIndex := pRes.nodesByLine
	originalParentsByLineIndex := oRes.parentsByLine

	findNextAnchor := func(s string, pLineNum int, pNode *tree_sitter.Node, fromLine int) *tsAnchor {
		oLineNum, ok := anchorLines[pLineNum]
		if ok {
			if verboseLogging {
				fmt.Printf("found anchor in anchorLines: oLineNum: %d, pLineNum: %d\n", oLineNum, pLineNum)
			}

			oNode := originalNodesByLineIndex[oLineNum-1]

			if verboseLogging {
				fmt.Printf("oNode.Type(): %s\n", oNode.Type())
			}

			anchor := &tsAnchor{open: oLineNum, close: int(oNode.EndPoint().Row) + 1}
			if verboseLogging {
				fmt.Printf("found anchor: %v\n", anchor)
			}
			return anchor
		} else {
			if verboseLogging {
				fmt.Printf("no anchor found in anchorLines: pLineNum: %d\n", pLineNum)
			}
		}

		for idx, line := range originalLines {
			if idx < fromLine {
				continue
			}

			if strings.TrimSpace(line) == "" {
				continue
			}

			// if verboseLogging {
			// 	fmt.Printf("line: %s, idx: %d\n", line, idx)
			// }

			oNode := originalNodesByLineIndex[idx]

			if verboseLogging {
				// fmt.Println("node:")
				// fmt.Println(oNode.Type())
				// fmt.Println(oNode)
				// fmt.Println(oNode.Content(originalBytes))
			}

			// just using string matching for now since there's too much ambiguity in node-based matching
			stringMatch := line == s
			// nodeMatch := oNode != nil && oNode.IsNamed() && nodesMatch(oNode, pNode, originalBytes, proposedBytes)

			if stringMatch {
				var endLineNum int
				if oNode != nil {
					if verboseLogging {
						fmt.Printf("oNode.Type(): %s\n", oNode.Type())
						fmt.Printf("oNode.EndPoint().Row: %d\n", oNode.EndPoint().Row)
						// fmt.Printf("%v\n", oNode)
						// fmt.Printf("oNode.Content(originalBytes):\n%q\n", oNode.Content(originalBytes))
					}
					endLineNum = int(oNode.EndPoint().Row) + 1
				}
				if verboseLogging {
					fmt.Printf("found match: num: %d, endLineNum: %d\n", idx+1, endLineNum)
					//   fmt.Println(oNode.Content(originalBytes))
				}

				return &tsAnchor{open: idx + 1, close: endLineNum}
			}
		}
		return nil
	}

	proposedUpdatesHaveLine := func(line string, afterLine int) bool {
		for idx, pLine := range proposedLines {
			pLineNum := idx + 1

			// if verboseLogging {
			// 	fmt.Printf("proposedUpdatesHaveLine - lineNum: %d, line: %s, pLine: %s, afterLine: %d\n", pLineNum, line, pLine, afterLine)
			// }

			if pLineNum > afterLine && pLine == line {
				return true
			}
		}
		return false
	}

	var oLineNum int = 0
	var refOpen bool
	var refStart int
	var refOriginalParent *tree_sitter.Node
	var postRefBuffers []strings.Builder

	closingLinesByPLineNum := map[int]int{}
	var noMatchUntilStructureClose string
	depth := 0

	var currentPNode *tree_sitter.Node
	var currentPNodeEndsAtIdx int
	var currentPNodeMatches bool

	lastLineMatched := true
	foundAnyAnchor := false

	setOLineNum := func(n int) {
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

	incDepth := func() {
		depth++
		if verboseLogging {
			fmt.Printf("incrementing depth: %d\n", depth)
		}
	}

	decDepth := func() {
		depth--
		if verboseLogging {
			fmt.Printf("decrementing depth: %d\n", depth)
		}
	}

	writeRefs := func(eof bool) bool {
		numRefs := len(postRefBuffers)
		if numRefs == 1 {
			if verboseLogging {
				fmt.Println("writeRefs")
				fmt.Printf("numRefs == 1, refStart: %d, oLineNum: %d\n", refStart, oLineNum)
			}

			var fullRef []string
			if eof {
				start := refStart - 1
				fullRef = originalLines[start:]
				if verboseLogging {
					fmt.Println("eof")
					fmt.Printf("fullRef refStart: %d\n", refStart)
					fmt.Printf("originalLines[refStart]: %q\n", originalLines[refStart])
					fmt.Printf("writing eof fullRef: %q\n", strings.Join(fullRef, "\n"))
					fmt.Printf("depth: %d\n", depth)
				}
			} else {
				start := refStart - 1
				if start < 0 {
					start = 0
				}
				end := oLineNum - 1
				if end < 1 {
					end = 0
				}

				// Add detailed diagnostic logging for invalid slice bounds
				if start > end {
					fmt.Printf("\n=== INVALID SLICE BOUNDS DIAGNOSTIC INFO ===\n")
					fmt.Printf("start: %d, end: %d\n", start, end)
					fmt.Printf("refStart: %d, oLineNum: %d\n", refStart, oLineNum)
					fmt.Printf("depth: %d, refOpen: %v\n", depth, refOpen)

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

					return false
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
					fmt.Printf("depth: %d\n", depth)
				}
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
				fmt.Printf("numRefs > 1, refOriginalParent: %s, eof: %v\n", refOriginalParent.Type(), eof)
				fmt.Printf("refOriginalParent.Content(originalBytes):\n%q\n", refOriginalParent.Content(originalBytes))
				fmt.Printf("numRefs: %d, oLineNum: %d\n", numRefs, oLineNum)
			}

			var upToLine int
			if eof {
				upToLine = len(originalLines)
			} else {
				upToLine = oLineNum
			}

			sections := getSections(
				refOriginalParent,
				originalBytes,
				numRefs,
				refStart,
				upToLine,
				foundAnyAnchor,
			)

			for i, section := range sections {
				if verboseLogging {
					fmt.Printf("writing section i: %d\n", i)
					// fmt.Printf("writing i: %d, section:\n%q\n	", i, section.String(originalLines, originalBytes))
				}
				write(section.String(originalLines, originalBytes), false)
				if verboseLogging {
					fmt.Println("writing postRefBuffer")
				}
				write(postRefBuffers[i].String(), false)
			}
		}

		return true
	}

	for idx, pLine := range proposedLines {
		finalLine := idx == len(proposedLines)-1
		pLineNum := idx + 1

		if verboseLogging {
			fmt.Printf("\n\ni: %d, num: %d, pLine: %q, refOpen: %v\n", idx, pLineNum, pLine, refOpen)
		}

		isRef := refsByLine[Reference(pLineNum)]
		isRemoval := removalsByLine[Removal(pLineNum)]

		if verboseLogging {
			fmt.Printf("isRef: %v\n", isRef)
			fmt.Printf("isRemoval: %v\n", isRemoval)
			fmt.Printf("oLineNum: %d\n", oLineNum)
			fmt.Printf("currentPNode set: %v\n", currentPNode != nil)
			fmt.Printf("currentPNodeEndsAtIdx: %d\n", currentPNodeEndsAtIdx)
			fmt.Printf("currentPNodeMatches: %v\n", currentPNodeMatches)
			fmt.Printf("lastLineMatched: %v\n", lastLineMatched)
			fmt.Printf("depth: %d\n", depth)
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
					pnode := proposedNodesByLineIndex[idx]
					fmt.Printf("pnode.Type(): %s\n", pnode.Type())
					// fmt.Printf("pnode.Content(proposedBytes):\n%q\n", pnode.Content(proposedBytes))
				}

				refOpen = true
				setOLineNum(oLineNum + 1)
				refStart = oLineNum

				if verboseLogging {
					fmt.Printf("setting refStart: %d\n", refStart)
				}

				if depth > 0 {
					refNode := originalNodesByLineIndex[refStart-1]

					if verboseLogging {
						fmt.Printf("refNode.Type(): %s\n", refNode.Type())
						// fmt.Printf("refNode.Content(originalBytes): %q\n", refNode.Content(originalBytes))

						current := refNode
						depth := 0
						for current != nil {
							fmt.Printf("parent depth %d type: %s content: %q\n",
								depth,
								current.Type(),
								current.Content(originalBytes))
							current = current.Parent()
							depth++
						}
					}

					refOriginalParent = originalParentsByLineIndex[refStart-1]

					if verboseLogging {
						fmt.Printf("setting refOriginalParent | refNode.Parent() | Type(): %s\n", refOriginalParent.Type())
					}
				} else {
					refOriginalParent = originalTree.RootNode()

					if verboseLogging {
						fmt.Printf("setting refOriginalParent | originalTree.RootNode() | Type(): %s\n", refOriginalParent.Type())
					}
				}

			}

			addNewPostRefBuffer()

			continue
		}

		if !refOpen && lastLineMatched && !currentPNodeMatches {
			if strings.TrimSpace(pLine) == "" {
				write(pLine, !finalLine)
				// Check if next line in original is also blank
				if oLineNum+1 < len(originalLines) && strings.TrimSpace(originalLines[oLineNum+1]) == "" {
					setOLineNum(oLineNum + 1)
				}
				continue
			}
		}

		pNode := proposedNodesByLineIndex[idx]
		pNodeStartsThisLine := pNode.StartPoint().Row == uint32(idx)
		pNodeEndsAtIdx := int(pNode.EndPoint().Row)
		pNodeMultiline := pNodeEndsAtIdx > idx

		var matching bool
		isClosingAnchor := closingLinesByPLineNum[pLineNum] != 0

		if verboseLogging {
			fmt.Printf("currentPNode != nil: %v\n", currentPNode != nil)
			fmt.Printf("currentPNodeMatches: %v\n", currentPNodeMatches)
		}

		if isClosingAnchor {
			if verboseLogging {
				fmt.Printf("isClosingAnchor: %v\n", isClosingAnchor)
			}
			matching = true
			setOLineNum(closingLinesByPLineNum[pLineNum])
			noMatchUntilStructureClose = ""
		} else if noMatchUntilStructureClose != pLine && !(currentPNode != nil && !currentPNodeMatches) {
			// find next line in original that matches
			anchor := findNextAnchor(pLine, pLineNum, pNode, oLineNum-1)
			if anchor != nil {
				foundAnyAnchor = true
				if verboseLogging {
					fmt.Println("anchor found")
					fmt.Printf("anchor.close: %d, anchor.open: %d\n", anchor.close, anchor.open)
				}
				matching = true
				setOLineNum(anchor.open)

				if pNodeStartsThisLine && pNodeMultiline && currentPNode != nil {
					currentPNodeMatches = true
				}

				if anchor.close != 0 && anchor.close != anchor.open {

					originalClosingLine := originalLines[anchor.close-1]

					if verboseLogging {
						fmt.Printf("originalClosingLine: %s\n", originalClosingLine)
					}

					if proposedUpdatesHaveLine(originalClosingLine, idx) {
						// if verboseLogging {
						// 	fmt.Printf("proposedUpdatesHaveLine: %v\n", proposedUpdatesHaveLine(originalClosingLine, anchor.open))
						// }
						if verboseLogging {
							fmt.Printf("proposedUpdatesHaveLine: %v\n", proposedUpdatesHaveLine(originalClosingLine, anchor.open))
						}
						closingPLineNum := int(pNode.EndPoint().Row) + 1
						closingLinesByPLineNum[closingPLineNum] = anchor.close
						noMatchUntilStructureClose = originalClosingLine
						incDepth()
						if verboseLogging {
							fmt.Printf("anchor.close: %d\n", anchor.close)
							fmt.Printf("closingPLineNum: %d\n", closingPLineNum)
							fmt.Printf("noMatchUntilStructureClose: %s\n", noMatchUntilStructureClose)
						}
					} else {
						if verboseLogging {
							fmt.Println("proposed updates do not have originalClosingLine")
						}
					}
				}
			}
		}

		if pNodeStartsThisLine && pNodeMultiline && (currentPNode == nil || matching != currentPNodeMatches) {
			if verboseLogging {
				fmt.Printf("setting currentPNode: %s\n", pNode.Type())
				fmt.Printf("pNodeEndsAtIdx: %d\n", pNodeEndsAtIdx)
				// fmt.Printf("content: %q\n", pNode.Content(proposedBytes))
				// fmt.Println(pNode)
			}
			currentPNode = pNode
			currentPNodeEndsAtIdx = pNodeEndsAtIdx
			currentPNodeMatches = matching

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
				ok := writeRefs(false)
				if !ok {
					return res, nil
				}
				write(pLine, !finalLine)
				wroteRefs = true
			}
		} else {
			if verboseLogging {
				fmt.Printf("no matching line\n")
			}
		}

		lastLineMatched = matching

		if currentPNodeEndsAtIdx == idx {
			currentPNode = nil
			currentPNodeEndsAtIdx = 0
			currentPNodeMatches = false
		}

		if isClosingAnchor {
			decDepth()
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

	}

	if refOpen {
		ok := writeRefs(true)
		if !ok {
			return res, nil
		}
	}

	if verboseLogging {
		// fmt.Printf("final result:\n%s\n", b.String())
	}

	res.NewFile = b.String()

	return res, nil
}
