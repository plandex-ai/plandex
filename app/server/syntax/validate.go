package syntax

import (
	"strings"
	"time"

	"github.com/plandex/plandex/shared"
	tree_sitter "github.com/smacker/go-tree-sitter"

	"context"
	"fmt"
)

const parserTimeout = 500 * time.Millisecond

type ValidationRes = struct {
	Lang     shared.Language
	Parser   *tree_sitter.Parser
	TimedOut bool
	Valid    bool
	Errors   []string
}

func ValidateFile(ctx context.Context, path string, file string) (*ValidationRes, error) {
	parser, lang, fallbackParser, fallbackLang := GetParserForPath(path)

	if parser == nil {
		return &ValidationRes{Lang: lang, Parser: nil}, nil
	}

	return ValidateWithParsers(ctx, lang, parser, fallbackLang, fallbackParser, file)
}

func ValidateWithParsers(ctx context.Context, lang shared.Language, parser *tree_sitter.Parser, fallbackLang shared.Language, fallbackParser *tree_sitter.Parser, file string) (*ValidationRes, error) {
	if file == "" {
		return &ValidationRes{Lang: lang, Parser: parser, Valid: true}, nil
	}

	// Set a timeout duration for the parsing operations
	ctx, cancel := context.WithTimeout(ctx, parserTimeout)
	defer cancel()

	// Parse the content
	tree, err := parser.ParseCtx(ctx, nil, []byte(file))

	if err != nil || tree == nil {
		if err != nil && err.Error() == "operation limit was hit" {
			return &ValidationRes{Lang: lang, Parser: parser, TimedOut: true}, nil
		}

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
					return &ValidationRes{Lang: lang, Parser: parser, TimedOut: true}, nil
				}

				return nil, fmt.Errorf("failed to parse the content with fallback parser: %v", err)
			}
			defer fallbackTree.Close()

			root = fallbackTree.RootNode()

			if !root.HasError() {
				return &ValidationRes{Lang: fallbackLang, Parser: fallbackParser, Valid: true}, nil
			}
		}

		errorMarkers := insertErrorMarkers(file, root)

		return &ValidationRes{
			Lang:   lang,
			Parser: parser,
			Valid:  false,
			Errors: errorMarkers,
		}, nil

	}

	return &ValidationRes{Lang: lang, Parser: parser, Valid: true}, nil
}

func insertErrorMarkers(source string, node *tree_sitter.Node) []string {
	if source == "" {
		return []string{}
	}

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
