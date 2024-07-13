package syntax

import (
	"path/filepath"
	"strings"
	"time"

	tree_sitter "github.com/smacker/go-tree-sitter"

	"context"
	"fmt"
)

const parserTimeout = 500 * time.Millisecond

type ValidationRes = struct {
	Ext       string
	Lang      string
	HasParser bool
	TimedOut  bool
	Valid     bool
	Errors    []string
}

func Validate(ctx context.Context, path, file string) (*ValidationRes, error) {
	ext := filepath.Ext(path)

	parser, lang, fallbackParser, fallbackLang := getParserForExt(ext)

	if parser == nil {
		return &ValidationRes{Ext: ext, Lang: lang, HasParser: false}, nil
	}

	// Set a timeout duration for the parsing operations
	ctx, cancel := context.WithTimeout(ctx, parserTimeout)
	defer cancel()

	// Parse the content
	tree, err := parser.ParseCtx(ctx, nil, []byte(file))

	if err != nil || tree == nil {
		return nil, fmt.Errorf("failed to parse the content: %v", err)
	}
	defer tree.Close()

	// Get the root node of the syntax tree and check for errors
	root := tree.RootNode()

	if root.HasError() {
		if fallbackParser != nil {
			fallbackTree, err := fallbackParser.ParseCtx(ctx, nil, []byte(file))
			if err != nil || fallbackTree == nil {

				if err != nil && strings.Contains(err.Error(), "timeout") {
					return &ValidationRes{Ext: ext, Lang: lang, HasParser: true, TimedOut: true}, nil
				}

				return nil, fmt.Errorf("failed to parse the content with fallback parser: %v", err)
			}
			defer fallbackTree.Close()

			root = fallbackTree.RootNode()

			if !root.HasError() {
				return &ValidationRes{Ext: ext, Lang: fallbackLang, HasParser: true, Valid: true}, nil
			}
		}

		errorMarkers := insertErrorMarkers(file, root)

		return &ValidationRes{
			Ext:       ext,
			Lang:      lang,
			HasParser: true,
			Valid:     false,
			Errors:    errorMarkers,
		}, nil

	}

	return &ValidationRes{Ext: ext, Lang: lang, HasParser: true, Valid: true}, nil
}

func insertErrorMarkers(source string, node *tree_sitter.Node) []string {
	var markers []string
	var uniqueMarkers = map[string]bool{}

	// Function to calculate line numbers
	calculateLineNumber := func(position int) int {
		return strings.Count(source[:position], "\n") + 1
	}

	hasChildError := func(n *tree_sitter.Node) bool {
		for i := 0; i < int(n.ChildCount()); i++ {
			if n.Child(i).HasError() {
				return true
			}
		}
		return false
	}

	visitNodes(node, func(n *tree_sitter.Node) {
		if n.HasError() && !hasChildError(n) {
			startPosition := int(n.StartByte())
			endPosition := int(n.EndByte())
			startLineNumber := calculateLineNumber(startPosition)
			endLineNumber := calculateLineNumber(endPosition)

			if startLineNumber == endLineNumber {
				uniqueMarkers[fmt.Sprintf("Invalid syntax on line %d", startLineNumber)] = true
			} else {
				uniqueMarkers[fmt.Sprintf("Invalid syntax on lines %d to %d", startLineNumber, endLineNumber)] = true
			}

		}
	})

	for marker := range uniqueMarkers {
		markers = append(markers, marker)
	}

	return markers
}

// visitNodes recursively visits nodes in the syntax tree
func visitNodes(n *tree_sitter.Node, f func(node *tree_sitter.Node)) {
	f(n)
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		visitNodes(child, f)
	}
}
