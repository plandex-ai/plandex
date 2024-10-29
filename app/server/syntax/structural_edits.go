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

type Anchor struct {
	Open  int
	Close int
}

type TreeSitterSection []*tree_sitter.Node

const verboseLogging = false

func (s TreeSitterSection) String(sourceLines []string, bytes []byte) string {
	if verboseLogging {
		fmt.Println("\n=== section.String START ===")
		fmt.Printf("Section has %d nodes\n", len(s))

		for i, node := range s {
			fmt.Printf("Node %d:\n", i)
			fmt.Printf("  Type: %s\n", node.Type())
			fmt.Printf("  StartPoint: row=%d col=%d\n", node.StartPoint().Row, node.StartPoint().Column)
			fmt.Printf("  EndPoint: row=%d col=%d\n", node.EndPoint().Row, node.EndPoint().Column)
			fmt.Printf("  Content: %q\n", node.Content(bytes))
		}
	}

	var b strings.Builder
	lines := map[int]bool{}
	for i, node := range s {
		lineNum := int(node.StartPoint().Row)
		if verboseLogging {
			fmt.Printf("Adding line %d from node %d\n", lineNum, i)
		}
		lines[lineNum] = true
	}

	var lineNums []int
	for lineNum := range lines {
		lineNums = append(lineNums, lineNum)
	}
	sort.Ints(lineNums)
	if verboseLogging {
		fmt.Printf("Sorted line numbers: %v\n", lineNums)
	}

	for _, lineNum := range lineNums {
		idx := lineNum
		line := sourceLines[idx]
		if verboseLogging {
			fmt.Printf("Writing line %d: %q\n", idx, line)
		}
		b.WriteString(line)
		if idx < len(sourceLines)-1 {
			if verboseLogging {
				fmt.Printf("Adding newline after line %d\n", idx)
			}
			b.WriteByte('\n')
		}
	}

	result := b.String()
	if verboseLogging {
		fmt.Printf("Final result: %q\n", result)
		fmt.Println("=== section.String END ===")
	}
	return result
}

func ApplyReferences(ctx context.Context, original, proposed string, references []Reference, parser *tree_sitter.Parser) (string, error) {
	var b strings.Builder

	matchCurrentN := map[string]int{}

	originalLines := strings.Split(original, "\n")
	proposedLines := strings.Split(proposed, "\n")

	originalBytes := []byte(original)
	proposedBytes := []byte(proposed)

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

	originalLinesCache := map[string][]Anchor{}
	findAnchors := func(s string, pNode *tree_sitter.Node) []Anchor {
		if idxs, ok := originalLinesCache[s]; ok {
			return idxs
		}
		var result []Anchor
		for idx, line := range originalLines {
			if strings.TrimSpace(line) == "" {
				continue
			}

			// fmt.Printf("line: %s, idx: %d\n", line, idx)

			oNode := originalNodesByLine[idx]

			// fmt.Println("node:")
			// fmt.Println(node.Type())
			// fmt.Println(node.Content(originalBytes))

			stringMatch := line == s
			nodeMatch := oNode != nil && oNode.IsNamed() && nodesMatch(oNode, pNode, originalBytes, proposedBytes)

			if stringMatch || nodeMatch {
				var endLineNum int
				if oNode != nil {
					endLineNum = int(oNode.EndPoint().Row) + 1
				}
				if verboseLogging {
					fmt.Printf("found match: num: %d, endLineNum: %d\n", idx+1, endLineNum)
				}
				// fmt.Println(oNode.Content(originalBytes))
				result = append(result, Anchor{Open: idx + 1, Close: endLineNum})
			}
		}
		originalLinesCache[s] = result
		return result
	}

	proposedUpdatesHaveLine := func(line string, afterLine int) bool {
		for idx, pLine := range proposedLines {
			lineNum := idx + 1
			if lineNum > afterLine && pLine == line {
				return true
			}
		}
		return false
	}

	var oLineNum int = 1

	var refOpen bool
	var refStart int
	var refOriginalParent *tree_sitter.Node
	var postRefBuffers []strings.Builder
	closingLinesByPLineNum := map[int]int{}
	var noMatchUntilStructureClose string

	writeRefs := func() {
		numRefs := len(postRefBuffers)
		if numRefs == 1 {
			if verboseLogging {
				fmt.Printf("numRefs == 1, refStart: %d, oLineNum: %d\n", refStart, oLineNum)
			}
			fullRef := originalLines[refStart-1 : oLineNum-1]
			if verboseLogging {
				fmt.Printf("writing fullRef: %s\n", strings.Join(fullRef, "\n"))
			}
			b.WriteString(strings.Join(fullRef, "\n"))
			b.WriteByte('\n')
			if verboseLogging {
				fmt.Printf("writing postRefBuffer:\n%s\n", postRefBuffers[0].String())
			}
			b.WriteString(postRefBuffers[0].String())
		} else {
			if verboseLogging {
				fmt.Printf("numRefs > 1, refOriginalParent: %s\n", refOriginalParent.Type())
				fmt.Printf("refOriginalParent.Content(originalBytes):\n%s\n", refOriginalParent.Content(originalBytes))
			}

			sections := getSections(refOriginalParent, originalBytes, numRefs, oLineNum)

			for i, section := range sections {
				if verboseLogging {
					fmt.Printf("writing i: %d, section:\n%s\n	", i, section.String(originalLines, originalBytes))
				}
				b.WriteString(section.String(originalLines, originalBytes))
				if verboseLogging {
					fmt.Printf("writing postRefBuffer:\n%s\n", postRefBuffers[i].String())
				}
				b.WriteString(postRefBuffers[i].String())
			}
		}
	}

	for idx, pLine := range proposedLines {
		if verboseLogging {
			fmt.Printf("i: %d, pLine: %s\n", idx, pLine)
		}

		finalLine := idx == len(proposedLines)-1
		pLineNum := idx + 1

		isRef := false
		for _, ref := range references {
			if int(ref) == pLineNum {
				isRef = true
				break
			}
		}

		if verboseLogging {
			fmt.Printf("isRef: %v\n", isRef)
		}

		if isRef {
			if refOpen {
				if verboseLogging {
					fmt.Printf("refOpen already true, adding to postRefBuffers\n")
				}
				latestBuffer := &postRefBuffers[len(postRefBuffers)-1]
				if len(latestBuffer.String()) > 0 {
					postRefBuffers = append(postRefBuffers, strings.Builder{})
				}
			} else {
				refOpen = true
				refStart = oLineNum + 1
				if verboseLogging {
					fmt.Printf("setting refStart: %d\n", refStart)
				}
				refNode := originalNodesByLine[refStart-1]
				refOriginalParent = refNode.Parent()
				postRefBuffers = append(postRefBuffers, strings.Builder{})
			}

			continue
		}

		if strings.TrimSpace(pLine) == "" {
			b.WriteString(pLine)
			b.WriteByte('\n')
			oLineNum++
			if verboseLogging {
				fmt.Printf("newline, incrementing oLineNum: %d\n", oLineNum)
			}
			continue
		}

		var matching bool
		isClosingAnchor := closingLinesByPLineNum[pLineNum] != 0

		if isClosingAnchor {
			if verboseLogging {
				fmt.Printf("isClosingAnchor: %v\n", isClosingAnchor)
			}
			matching = true
			oLineNum = closingLinesByPLineNum[pLineNum]
			if verboseLogging {
				fmt.Printf("setting oLineNum: %d\n", oLineNum)
			}
			noMatchUntilStructureClose = ""
		} else if noMatchUntilStructureClose != pLine {
			// find all lines in original that match
			pNode := proposedNodesByLine[idx]
			anchors := findAnchors(pLine, pNode)
			if len(anchors) > 0 {
				matching = true
				currentN := matchCurrentN[pLine]
				oLineAnchor := anchors[currentN]
				oLineNum = oLineAnchor.Open
				if verboseLogging {
					fmt.Printf("setting oLineNum: %d\n", oLineNum)
				}

				if oLineAnchor.Close != 0 && oLineAnchor.Close != oLineAnchor.Open {
					originalClosingLine := originalLines[oLineAnchor.Close-1]

					if proposedUpdatesHaveLine(originalClosingLine, oLineAnchor.Open) {
						closingPLineNum := int(pNode.EndPoint().Row) + 1
						closingLinesByPLineNum[closingPLineNum] = oLineAnchor.Close
						noMatchUntilStructureClose = originalClosingLine
					}
				}

				matchCurrentN[pLine] = currentN + 1
			}
		}

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
				writeRefs()
				b.WriteString(pLine)
				if !finalLine {
					b.WriteByte('\n')
				}

				// reset buffers
				postRefBuffers = []strings.Builder{}
				continue
			}
		} else {
			if verboseLogging {
				fmt.Printf("no matching line\n")
			}
		}

		if refOpen {
			if verboseLogging {
				fmt.Printf("writing to latest postRefBuffer: %s\n", pLine)
			}
			latestBuffer := &postRefBuffers[len(postRefBuffers)-1]
			latestBuffer.WriteString(pLine)
			latestBuffer.WriteByte('\n')
			oLineNum++
			if verboseLogging {
				fmt.Printf("incrementing oLineNum: %d\n", oLineNum)
			}
		} else {
			if verboseLogging {
				fmt.Printf("writing pLine: %s\n", pLine)
			}
			b.WriteString(pLine)

			if !finalLine {
				b.WriteByte('\n')
			}
		}

	}

	if refOpen {
		writeRefs()
	}

	if verboseLogging {
		fmt.Printf("final result:\n%s\n", b.String())
	}

	return b.String(), nil
}

func nodesMatch(n1, n2 *tree_sitter.Node, source1, source2 []byte) bool {
	// fmt.Println("check nodesMatch")
	if n1 == nil || n2 == nil || !n1.IsNamed() || !n2.IsNamed() {
		// fmt.Println("nodes not named")
		return false
	}

	if n1.Type() != n2.Type() {
		// fmt.Println("n1 type != n2 type")
		return false
	}

	// Find first declaration/definition node
	findDeclNode := func(n *tree_sitter.Node) *tree_sitter.Node {
		cursor := tree_sitter.NewTreeCursor(n)
		defer cursor.Close()

		var findDecl func() *tree_sitter.Node
		findDecl = func() *tree_sitter.Node {
			nodeType := cursor.CurrentNode().Type()
			if strings.HasSuffix(nodeType, "_declaration") ||
				strings.HasSuffix(nodeType, "_definition") {
				return cursor.CurrentNode()
			}

			if cursor.GoToFirstChild() {
				for {
					if node := findDecl(); node != nil {
						return node
					}
					if !cursor.GoToNextSibling() {
						break
					}
				}
				cursor.GoToParent()
			}
			return nil
		}

		return findDecl()
	}

	decl1 := findDeclNode(n1)
	decl2 := findDeclNode(n2)

	if decl1 == nil || decl2 == nil {
		return false
	}

	// fmt.Printf("found declaration nodes of type: %s and %s\n", decl1.Type(), decl2.Type())

	// Get names from first named child
	name1Node := decl1.NamedChild(0)
	name2Node := decl2.NamedChild(0)

	if name1Node == nil || name2Node == nil ||
		name1Node.Type() != "identifier" || name2Node.Type() != "identifier" {
		return false
	}

	name1 := name1Node.Content(source1)
	name2 := name2Node.Content(source2)

	// fmt.Printf("name1: %s, name2: %s\n", name1, name2)

	return name1 == name2
}

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
		// fmt.Printf("node: %s, depth: %d, childCount: %d\n", node.Type(), depth, node.ChildCount())
		// spew.Dump(node.StartPoint())

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

			if parent := findParent(root); parent != nil {
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
