package syntax

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/davecgh/go-spew/spew"
	tree_sitter "github.com/smacker/go-tree-sitter"
)

type Reference int
type Removal int

type Anchor struct {
	Open  int
	Close int
}

type TreeSitterSection []*tree_sitter.Node

const verboseLogging = true

func ApplyReferences(
	ctx context.Context,
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
			fmt.Printf("writing: %q\n", s)
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

	for i, line := range proposedLines {
		// keep indentation for syntax parsing
		content := strings.TrimSpace(line)
		if removalsByLine[Removal(i+1)] {
			proposedLines[i] = strings.Replace(line, content, "", 1)
		} else if refsByLine[Reference(i+1)] {
			proposedLines[i] = strings.Replace(line, content, "", 1)
		}
	}

	proposedWithoutRemovals := strings.Join(proposedLines, "\n")

	originalBytes := []byte(original)
	proposedBytes := []byte(proposedWithoutRemovals)

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

	originalNodesByLine := buildNodeIndex(originalTree)
	proposedNodesByLine := buildNodeIndex(proposedTree)

	findNextAnchor := func(s string, pLineNum int, pNode *tree_sitter.Node, fromLine int) *Anchor {
		oLineNum, ok := anchorLines[pLineNum]
		if ok {
			oNode := originalNodesByLine[oLineNum-1]
			return &Anchor{Open: oLineNum, Close: int(oNode.EndPoint().Row) + 1}
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

			oNode := originalNodesByLine[idx]

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
					fmt.Printf("oNode.Type(): %s\n", oNode.Type())
					fmt.Printf("oNode.EndPoint().Row: %d\n", oNode.EndPoint().Row)
					// fmt.Printf("%v\n", oNode)
					// fmt.Printf("oNode.Content(originalBytes):\n%q\n", oNode.Content(originalBytes))
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
				end := oLineNum - 1
				if verboseLogging {
					fmt.Printf("writing fullRef\n")
					fmt.Printf("refStart: %d, oLineNum: %d\n", refStart, oLineNum)
					fmt.Printf("start: %d, end: %d\n", start, end)
				}
				fullRef = originalLines[start:end]
				if verboseLogging {
					fmt.Printf("fullRef refStart: %d, oLineNum-1: %d\n", refStart, oLineNum-1)
					fmt.Printf("originalLines[refStart-1]: %q\n", originalLines[refStart-1])
					fmt.Printf("originalLines[oLineNum-1]: %q\n", originalLines[oLineNum-1])
					fmt.Printf("depth: %d\n", depth)
				}
			}

			write(strings.Join(fullRef, "\n"), depth > 0)

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
				fmt.Println("original parent type:", refOriginalParent.Type())
				fmt.Printf("numRefs: %d, oLineNum: %d\n", numRefs, oLineNum)
			}

			sections := getSections(refOriginalParent, originalBytes, numRefs, oLineNum)

			for i, section := range sections {
				if verboseLogging {
					fmt.Printf("writing i: %d, section:\n%q\n	", i, section.String(originalLines, originalBytes))
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
					pnode := proposedNodesByLine[idx]
					fmt.Printf("pnode.Type(): %s\n", pnode.Type())
					fmt.Printf("pnode.Content(proposedBytes):\n%q\n", pnode.Content(proposedBytes))
				}

				refOpen = true
				setOLineNum(oLineNum + 1)
				refStart = oLineNum

				if verboseLogging {
					fmt.Printf("setting refStart: %d\n", refStart)
				}

				if depth > 0 {
					refNode := originalNodesByLine[refStart-1]
					refOriginalParent = refNode.Parent()
				} else {
					refOriginalParent = originalTree.RootNode()
				}

			}

			addNewPostRefBuffer()

			continue
		}

		if !refOpen && lastLineMatched && !currentPNodeMatches {
			if strings.TrimSpace(pLine) == "" {
				write(pLine, true)
				if verboseLogging {
					fmt.Printf("newline, incrementing oLineNum: %d\n", oLineNum)
				}
				setOLineNum(oLineNum + 1)
				continue
			}
		}

		pNode := proposedNodesByLine[idx]
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
				fmt.Printf("content: %q\n", pNode.Content(proposedBytes))
				fmt.Println(pNode)
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
		fmt.Printf("final result:\n%s\n", b.String())
	}

	return b.String(), nil
}

// just using exact string matches for now since there's too much ambiguity in node-based matching
// func nodesMatch(n1, n2 *tree_sitter.Node, source1, source2 []byte) bool {
// 	// if verboseLogging {
// 	// 	fmt.Println("check nodesMatch")
// 	// }
// 	if n1 == nil || n2 == nil || !n1.IsNamed() || !n2.IsNamed() {
// 		// if verboseLogging {
// 		// 	fmt.Println("nodes not named")
// 		// }
// 		return false
// 	}

// 	if n1.Type() != n2.Type() {
// 		// if verboseLogging {
// 		// 	fmt.Println("n1 type != n2 type")
// 		// }
// 		return false
// 	}

// 	// Find first declaration/definition node
// 	findDeclNode := func(n *tree_sitter.Node) *tree_sitter.Node {
// 		cursor := tree_sitter.NewTreeCursor(n)
// 		defer cursor.Close()

// 		var findDecl func() *tree_sitter.Node
// 		findDecl = func() *tree_sitter.Node {
// 			nodeType := cursor.CurrentNode().Type()
// 			if strings.HasSuffix(nodeType, "_declaration") ||
// 				strings.HasSuffix(nodeType, "_definition") ||
// 				nodeType == "pair" {
// 				return cursor.CurrentNode()
// 			}

// 			if cursor.GoToFirstChild() {
// 				for {
// 					if node := findDecl(); node != nil {
// 						return node
// 					}
// 					if !cursor.GoToNextSibling() {
// 						break
// 					}
// 				}
// 				cursor.GoToParent()
// 			}
// 			return nil
// 		}

// 		return findDecl()
// 	}

// 	decl1 := findDeclNode(n1)
// 	decl2 := findDeclNode(n2)

// 	if decl1 == nil || decl2 == nil {
// 		return false
// 	}

// 	// if verboseLogging {
// 	// 	fmt.Printf("found declaration nodes of type: %s and %s\n", decl1.Type(), decl2.Type())
// 	// }

// 	// Get names from first named child
// 	name1Node := decl1.NamedChild(0)
// 	name2Node := decl2.NamedChild(0)

// 	if name1Node == nil || name2Node == nil ||
// 		!strings.HasSuffix(name1Node.Type(), "identifier") || !strings.HasSuffix(name2Node.Type(), "identifier") {
// 		return false
// 	}

// 	name1 := name1Node.Content(source1)
// 	name2 := name2Node.Content(source2)

// 	// if verboseLogging {
// 	// 	fmt.Printf("name1: %s, name2: %s\n", name1, name2)
// 	// }

// 	return name1 == name2
// }

func getSections(parent *tree_sitter.Node, bytes []byte, numSections, upToLine int) []TreeSitterSection {
	sections := make([]TreeSitterSection, numSections)
	structures := [][]*tree_sitter.Node{}
	latestStructure := []*tree_sitter.Node{}

	cursor := tree_sitter.NewTreeCursor(parent)
	defer cursor.Close()

	firstLineNum := int(parent.StartPoint().Row)
	if verboseLogging {
		fmt.Printf("firstLineNum: %d\n", firstLineNum)
	}

	if cursor.GoToFirstChild() {
		for {
			node := cursor.CurrentNode()
			lineNum := int(node.StartPoint().Row)
			if verboseLogging {
				fmt.Printf("lineNum: %d\n", lineNum)
				fmt.Println(node.Content(bytes))
			}

			if lineNum == firstLineNum {
				if verboseLogging {
					fmt.Printf("skipping first line\n")
				}
				if !cursor.GoToNextSibling() {
					break
				}
				continue
			}

			if node.EndPoint().Row >= uint32(upToLine-1) {
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

		if verboseLogging {
			fmt.Printf("group:\n")
			spew.Dump(group)
		}

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
				fmt.Println(node.Content(bytes))
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

func buildNodeIndex(tree *tree_sitter.Tree) map[int]*tree_sitter.Node {
	nodesByLine := make(map[int]*tree_sitter.Node)
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
			}
		}
	}

	// spew.Dump(nodesByLine)
	return nodesByLine
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
	// if verboseLogging {
	// 	fmt.Println("\n=== section.String START ===")
	// 	fmt.Printf("Section has %d nodes\n", len(s))

	// 	for i, node := range s {
	// 		fmt.Printf("Node %d:\n", i)
	// 		fmt.Printf("  Type: %s\n", node.Type())
	// 		fmt.Printf("  StartPoint: row=%d col=%d\n", node.StartPoint().Row, node.StartPoint().Column)
	// 		fmt.Printf("  EndPoint: row=%d col=%d\n", node.EndPoint().Row, node.EndPoint().Column)
	// 		fmt.Printf("  Content: %q\n", node.Content(bytes))
	// 	}
	// }

	var b strings.Builder
	lines := map[int]bool{}
	for _, node := range s {
		lineNum := int(node.StartPoint().Row)
		// if verboseLogging {
		// 	fmt.Printf("Adding line %d from node %d\n", lineNum, i)
		// }
		lines[lineNum] = true
	}

	var lineNums []int
	for lineNum := range lines {
		lineNums = append(lineNums, lineNum)
	}
	sort.Ints(lineNums)
	// if verboseLogging {
	// 	fmt.Printf("Sorted line numbers: %v\n", lineNums)
	// }

	for _, lineNum := range lineNums {
		idx := lineNum
		line := sourceLines[idx]
		// if verboseLogging {
		// 	fmt.Printf("Writing line %d: %q\n", idx, line)
		// }
		b.WriteString(line)
		if idx < len(sourceLines)-1 {
			// if verboseLogging {
			// 	fmt.Printf("Adding newline after line %d\n", idx)
			// }
			b.WriteByte('\n')
		}
	}

	result := b.String()
	// if verboseLogging {
	// 	fmt.Printf("Final result: %q\n", result)
	// 	fmt.Println("=== section.String END ===")
	// }
	return result
}
