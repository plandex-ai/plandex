package syntax

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	tree_sitter "github.com/smacker/go-tree-sitter"
)

// StripComments removes all comments from the given source code using the appropriate parser
func StripComments(ctx context.Context, path, source string) (string, error) {
	ext := filepath.Ext(path)
	parser, _, _, _ := GetParserForExt(ext)

	// If no parser is available, return the source as is
	if parser == nil {
		return source, nil
	}

	ctx, cancel := context.WithTimeout(ctx, parserTimeout)
	defer cancel()

	tree, err := parser.ParseCtx(ctx, nil, []byte(source))
	if err != nil {
		return source, fmt.Errorf("failed to parse the content: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	commentNodes := findCommentNodes(root)

	// Sort comment nodes in reverse order to avoid shifting positions
	sort.Slice(commentNodes, func(i, j int) bool {
		return commentNodes[i].StartByte() > commentNodes[j].StartByte()
	})

	// Remove comments from the source
	result := []byte(source)
	for _, node := range commentNodes {
		start := node.StartByte()
		end := node.EndByte()
		result = append(result[:start], result[end:]...)
	}

	return string(result), nil
}

func findCommentNodes(node *tree_sitter.Node) []*tree_sitter.Node {
	var commentNodes []*tree_sitter.Node

	visitNodes(node, func(n *tree_sitter.Node) {
		if n.Type() == "comment" {
			commentNodes = append(commentNodes, n)
		}
	})

	return commentNodes
}

func GetCommentSymbols(lang string) (string, string) {
	switch lang {
	case "c", "cpp", "csharp", "java", "javascript", "go", "rust", "swift", "kotlin", "groovy", "scala", "typescript", "php":
		return "//", ""
	case "bash", "dockerfile", "elixir", "hcl", "python", "ruby", "toml", "yaml":
		return "#", ""
	case "lua", "elm":
		return "--", ""
	case "css":
		return "/*", "*/"
	case "html":
		return "<!--", "-->"
	case "ocaml":
		return "(*", "*)"
	case "svelte", "jsx", "tsx", "json":
		return "", "" // comments are either not allowed or correct symbols depend on the context
	}

	return "", ""
}
