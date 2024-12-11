package syntax

import (
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"
	tree_sitter "github.com/smacker/go-tree-sitter"
)

func getSections(parent *tree_sitter.Node, bytes []byte, numSections, fromLine, upToLine int, foundAnyAnchor bool) []TreeSitterSection {
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
					fmt.Println("startLineNum < fromLine, skipping")
					fmt.Printf("skipping lineNum: %d | before fromLine: %d\n", startLineNum, fromLine)
				}
				if !cursor.GoToNextSibling() {
					if verboseLogging {
						fmt.Println("no next sibling, breaking")
					}
					break
				}
				continue
			}

			if startLineNum == parentFirstLineNum && foundAnyAnchor {
				if verboseLogging {
					fmt.Printf("skipping first line\n")
				}
				if !cursor.GoToNextSibling() {
					if verboseLogging {
						fmt.Println("no next sibling, breaking")
					}
					break
				}
				continue
			}

			if endLineNum > upToLine {
				if verboseLogging {
					fmt.Println("endLineNum > upToLine, breaking")
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
				if verboseLogging {
					fmt.Println("endLineNum == parentEndLineNum && !isStructuralNode(node), breaking")
				}
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
				if verboseLogging {
					fmt.Println("no next sibling, breaking")
				}
				break
			}
		}
		if len(latestStructure) > 0 {
			if verboseLogging {
				fmt.Println("appending latestStructure to structures")
			}
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
