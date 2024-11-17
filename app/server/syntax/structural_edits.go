package syntax

import (
	"context"
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	tree_sitter "github.com/smacker/go-tree-sitter"
)

type Reference int
type Removal int

type Anchor struct {
	Open  int
	Close int
}

type TreeSitterSection []*tree_sitter.Node

const verboseLogging = false

func ApplyChanges(
	ctx context.Context,
	language shared.TreeSitterLanguage,
	parser *tree_sitter.Parser,
	original,
	proposed string,
	references []Reference,
	removals []Removal,
	anchorLines map[int]int,
) (string, error) {
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

	originalBytes := []byte(original)
	proposedBytes := []byte(proposedWithNormalizedComments)

	originalTree, err := parser.ParseCtx(ctx, nil, originalBytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse the original content: %v", err)
	}
	defer originalTree.Close()

	proposedTree, err := parser.ParseCtx(ctx, nil, proposedBytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse the proposed content: %v", err)
	}
	defer proposedTree.Close()

	if verboseLogging {
		fmt.Printf("anchorLines: %v\n", anchorLines)
	}

	oRes := buildNodeIndex(originalTree)
	pRes := buildNodeIndex(proposedTree)

	originalNodesByLineIndex := oRes.nodesByLine
	proposedNodesByLineIndex := pRes.nodesByLine
	originalParentsByLineIndex := oRes.parentsByLine

	findNextAnchor := func(s string, pLineNum int, pNode *tree_sitter.Node, fromLine int) *Anchor {
		oLineNum, ok := anchorLines[pLineNum]
		if ok {
			if verboseLogging {
				fmt.Printf("found anchor in anchorLines: oLineNum: %d, pLineNum: %d\n", oLineNum, pLineNum)
			}

			oNode := originalNodesByLineIndex[oLineNum-1]

			if verboseLogging {
				fmt.Printf("oNode.Type(): %s\n", oNode.Type())
			}

			anchor := &Anchor{Open: oLineNum, Close: int(oNode.EndPoint().Row) + 1}
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

				return &Anchor{Open: idx + 1, Close: endLineNum}
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

	writeRefs := func(eof bool) {
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
				fmt.Printf("numRefs > 1, refOriginalParent: %s\n", refOriginalParent.Type())
				fmt.Printf("refOriginalParent.Content(originalBytes):\n%q\n", refOriginalParent.Content(originalBytes))
				fmt.Printf("numRefs: %d, oLineNum: %d\n", numRefs, oLineNum)
			}

			sections := getSections(refOriginalParent, originalBytes, numRefs, refStart, oLineNum)

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
				if verboseLogging {
					fmt.Println("anchor found")
					fmt.Printf("anchor.Close: %d, anchor.Open: %d\n", anchor.Close, anchor.Open)
				}
				matching = true
				setOLineNum(anchor.Open)

				if pNodeStartsThisLine && pNodeMultiline && currentPNode != nil {
					currentPNodeMatches = true
				}

				if anchor.Close != 0 && anchor.Close != anchor.Open {

					originalClosingLine := originalLines[anchor.Close-1]

					if verboseLogging {
						fmt.Printf("originalClosingLine: %s\n", originalClosingLine)
					}

					if proposedUpdatesHaveLine(originalClosingLine, idx) {
						// if verboseLogging {
						// 	fmt.Printf("proposedUpdatesHaveLine: %v\n", proposedUpdatesHaveLine(originalClosingLine, anchor.Open))
						// }
						if verboseLogging {
							fmt.Printf("proposedUpdatesHaveLine: %v\n", proposedUpdatesHaveLine(originalClosingLine, anchor.Open))
						}
						closingPLineNum := int(pNode.EndPoint().Row) + 1
						closingLinesByPLineNum[closingPLineNum] = anchor.Close
						noMatchUntilStructureClose = originalClosingLine
						incDepth()
						if verboseLogging {
							fmt.Printf("anchor.Close: %d\n", anchor.Close)
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
				writeRefs(false)
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
		writeRefs(true)
	}

	if verboseLogging {
		// fmt.Printf("final result:\n%s\n", b.String())
	}

	return b.String(), nil
}

func getSections(parent *tree_sitter.Node, bytes []byte, numSections, fromLine, upToLine int) []TreeSitterSection {
	sections := make([]TreeSitterSection, numSections)
	structures := [][]*tree_sitter.Node{}
	latestStructure := []*tree_sitter.Node{}

	cursor := tree_sitter.NewTreeCursor(parent)
	defer cursor.Close()

	parentFirstLineNum := int(parent.StartPoint().Row) + 1
	parentEndLineNum := int(parent.EndPoint().Row) + 1
	if verboseLogging {
		fmt.Printf("parent.Type(): %s\n", parent.Type())
		fmt.Printf("parent.Content(bytes):\n%q\n", parent.Content(bytes))
		fmt.Printf("parentFirstLineNum: %d\n", parentFirstLineNum)
		fmt.Printf("parentEndLineNum: %d\n", parentEndLineNum)
	}

	if cursor.GoToFirstChild() {
		for {
			node := cursor.CurrentNode()

			startLineNum := int(node.StartPoint().Row) + 1
			endLineNum := int(node.EndPoint().Row) + 1
			if verboseLogging {
				fmt.Printf("startLineNum: %d, endLineNum: %d\n", startLineNum, endLineNum)
				fmt.Println(node.Type())
				fmt.Printf("node.Content(bytes):\n%q\n", node.Content(bytes))
			}

			if startLineNum < fromLine {
				if verboseLogging {
					fmt.Printf("skipping lineNum: %d | before fromLine: %d\n", startLineNum, fromLine)
				}
				if !cursor.GoToNextSibling() {
					break
				}
				continue
			}

			if startLineNum == parentFirstLineNum {
				if verboseLogging {
					fmt.Printf("skipping first line\n")
				}
				if !cursor.GoToNextSibling() {
					break
				}
				continue
			}

			if endLineNum > upToLine {
				if verboseLogging {
					fmt.Printf("upToLine: %d, endLineNum: %d, node.Type(): %s\n", upToLine, endLineNum, node.Type())

					toLog := node.Content(bytes)
					if len(toLog) > 200 {
						toLog = toLog[:100] + "\n...\n" + toLog[len(toLog)-100:]
					}
					fmt.Println(toLog)
				}

				break
			}

			if endLineNum == parentEndLineNum && !isStructuralNode(node) {
				break
			}

			if isStructuralNode(node) {
				if verboseLogging {
					fmt.Printf("found structural node: %s\n", node.Type())
					fmt.Printf("starting new group\n")
				}
				if len(latestStructure) > 0 {
					structures = append(structures, latestStructure)
				}
				latestStructure = []*tree_sitter.Node{node}
			} else {
				if verboseLogging {
					fmt.Printf("not structural: %s\n", node.Type())
				}
				latestStructure = append(latestStructure, node)
			}

			if !cursor.GoToNextSibling() {
				break
			}
		}
		if len(latestStructure) > 0 {
			structures = append(structures, latestStructure)
		}
	}

	if verboseLogging {
		fmt.Printf("structures:\n")
		spew.Dump(structures)
	}

	numStructural := len(structures)
	baseSize := numStructural / numSections
	remainder := numStructural % numSections

	if verboseLogging {
		fmt.Printf("baseSize: %d, remainder: %d\n", baseSize, remainder)
	}

	startIndex := 0

	for i := 0; i < numSections; i++ {
		size := baseSize
		if i < remainder {
			size++
		}

		endIndex := startIndex + size
		if endIndex > len(structures) {
			endIndex = len(structures)
		}

		var section TreeSitterSection
		group := structures[startIndex:endIndex]

		for _, s := range group {
			section = append(section, s...)
		}
		sections[i] = section
		startIndex = endIndex
	}

	if verboseLogging {
		fmt.Printf("sections:\n")
		spew.Dump(sections)

		fmt.Println("Sections content:")
		for i, section := range sections {
			fmt.Printf("section %d:\n", i)
			for j, node := range section {
				fmt.Printf("node %d:\n", j)
				toLog := node.Content(bytes)
				if len(toLog) > 200 {
					toLog = toLog[:100] + "\n...\n" + toLog[len(toLog)-100:]
				}
				fmt.Println(toLog)
			}
		}
	}

	return sections
}

func isStructuralNode(node *tree_sitter.Node) bool {
	nodeType := node.Type()

	if strings.Contains(nodeType, "comment") {
		return false
	}

	if strings.HasSuffix(nodeType, "space") {
		return false
	}

	switch nodeType {
	case "ws", "newline", "indent", "dedent":
		return false
	}

	if node.IsNamed() {
		return true
	}

	if node.ChildCount() > 0 {
		return true
	}

	return false
}

type nodeIndex struct {
	nodesByLine   map[int]*tree_sitter.Node
	parentsByLine map[int]*tree_sitter.Node
}

func buildNodeIndex(tree *tree_sitter.Tree) *nodeIndex {
	nodesByLine := make(map[int]*tree_sitter.Node)
	parentsByLine := make(map[int]*tree_sitter.Node)

	root := tree.RootNode()

	// First pass - index direct nodes
	var indexNodes func(node *tree_sitter.Node, depth int)
	indexNodes = func(node *tree_sitter.Node, depth int) {
		// if verboseLogging {
		// 	fmt.Printf("node: %s, depth: %d, childCount: %d\n", node.Type(), depth, node.ChildCount())
		// 	// spew.Dump(node.StartPoint())
		// }

		if node.Type() != root.Type() {
			line := int(node.StartPoint().Row)
			existing, exists := nodesByLine[line]
			if !exists ||
				node.StartPoint().Column < existing.StartPoint().Column ||
				(node.StartPoint().Column == existing.StartPoint().Column && depth < getNodeDepth(existing)) {
				nodesByLine[line] = node
				parentsByLine[line] = node.Parent()
			}
		}

		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			indexNodes(child, depth+1)
		}
	}

	indexNodes(root, 0)

	// Second pass - fill in missing lines with parent nodes
	endLine := int(root.EndPoint().Row)
	for line := 0; line <= endLine; line++ {
		if _, exists := nodesByLine[line]; !exists {
			// Find containing parent node
			var findParent func(node *tree_sitter.Node) *tree_sitter.Node
			findParent = func(node *tree_sitter.Node) *tree_sitter.Node {
				nodeStart := int(node.StartPoint().Row)
				nodeEnd := int(node.EndPoint().Row)

				if nodeStart <= line && line <= nodeEnd {
					// Check children first for more specific containment
					for i := 0; i < int(node.ChildCount()); i++ {
						child := node.Child(i)
						if found := findParent(child); found != nil {
							return found
						}
					}
					return node
				}
				return nil
			}

			parent := findParent(root)

			if parent == nil {
				nodesByLine[line] = root
			} else {
				nodesByLine[line] = parent
				parentsByLine[line] = parent
			}
		}
	}

	// spew.Dump(nodesByLine)
	return &nodeIndex{
		nodesByLine:   nodesByLine,
		parentsByLine: parentsByLine,
	}
}

func getNodeDepth(node *tree_sitter.Node) int {
	depth := 0
	current := node
	for current.Parent() != nil {
		depth++
		current = current.Parent()
	}
	return depth
}

func (s TreeSitterSection) String(sourceLines []string, bytes []byte) string {
	if len(s) == 0 {
		return ""
	}

	// Find first structural node
	var firstNode *tree_sitter.Node
	for _, n := range s {
		if isStructuralNode(n) {
			firstNode = n
			break
		}
	}

	// Find last structural node
	var lastNode *tree_sitter.Node
	for i := len(s) - 1; i >= 0; i-- {
		if isStructuralNode(s[i]) {
			lastNode = s[i]
			break
		}
	}

	startIdx := int(firstNode.StartPoint().Row)
	endIdx := int(lastNode.EndPoint().Row)

	if verboseLogging {
		for _, node := range s {
			fmt.Printf("node.Type(): %s\n", node.Type())
			fmt.Printf("StartPoint().Row: %d\n", node.StartPoint().Row)
			fmt.Printf("EndPoint().Row: %d\n", node.EndPoint().Row)
		}

		fmt.Printf("section.String startIdx: %d, endIdx: %d\n", startIdx, endIdx)
	}

	parent := lastNode.Parent()
	parentEndNode := parent.Child(int(parent.ChildCount()) - 1)
	lastLine := sourceLines[endIdx]
	if lastLine == parentEndNode.Content(bytes) {
		endIdx--
	}

	result := strings.Join(sourceLines[startIdx:endIdx+1], "\n") + "\n"

	if verboseLogging {
		toLog := result
		if len(toLog) > 200 {
			toLog = toLog[:100] + "\n...\n" + toLog[len(toLog)-100:]
		}
		fmt.Printf("section.String result: %s\n", toLog)
	}

	return result
}
