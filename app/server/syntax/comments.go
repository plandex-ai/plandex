package syntax

import shared "plandex-shared"

// // FindComments parses the given source code for the language implied by path
// // and returns a slice of all comment strings plus an IsRef field indicating
// // whether the comment is referencing original code (heuristic).
// func FindComments(ctx context.Context, parser *tree_sitter.Parser, source string) ([]Comment, error) {
// 	nodes, err := findCommentNodesForPath(ctx, parser, source)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var comments []Comment
// 	for _, node := range nodes {
// 		start := node.StartByte()
// 		end := node.EndByte()
// 		raw := source[start:end]

// 		comments = append(comments, Comment{
// 			Txt:   raw,
// 			IsRef: isRef(raw), // your existing logic
// 		})
// 	}
// 	return comments, nil
// }

// // StripComments removes all comments from the given source code using the appropriate parser
// func StripComments(ctx context.Context, parser *tree_sitter.Parser, source string) (string, error) {
// 	// Find the comment nodes first:
// 	commentNodes, err := findCommentNodesForPath(ctx, parser, source)
// 	if err != nil {
// 		// If parsing fails, return the source as-is along with an error.
// 		return source, fmt.Errorf("failed to parse the content: %v", err)
// 	}

// 	// If no parser is available or no comments found, just return the source unmodified.
// 	if len(commentNodes) == 0 {
// 		return source, nil
// 	}

// 	// Sort comment nodes in reverse order to remove them from the source safely.
// 	sort.Slice(commentNodes, func(i, j int) bool {
// 		return commentNodes[i].StartByte() > commentNodes[j].StartByte()
// 	})

// 	// Remove comments from the source.
// 	result := []byte(source)
// 	for _, node := range commentNodes {
// 		start := node.StartByte()
// 		end := node.EndByte()
// 		result = append(result[:start], result[end:]...)
// 	}

// 	return string(result), nil
// }

// func findCommentNodesForPath(ctx context.Context, parser *tree_sitter.Parser, source string) ([]*tree_sitter.Node, error) {
// 	if parser == nil {
// 		// If no parser is available for this file type, return empty.
// 		return nil, nil
// 	}

// 	// Use a context with timeout (from your existing parserTimeout).
// 	ctx, cancel := context.WithTimeout(ctx, parserTimeout)
// 	defer cancel()

// 	tree, err := parser.ParseCtx(ctx, nil, []byte(source))
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer tree.Close()

// 	// Gather all comment nodes.
// 	root := tree.RootNode()
// 	commentNodes := findCommentNodes(root)
// 	return commentNodes, nil
// }

// func findCommentNodes(node *tree_sitter.Node) []*tree_sitter.Node {
// 	var commentNodes []*tree_sitter.Node

// 	visitNodes(node, func(n *tree_sitter.Node) {
// 		if n.Type() == "comment" {
// 			commentNodes = append(commentNodes, n)
// 		}
// 	})

// 	return commentNodes
// }

func GetCommentSymbols(lang shared.Language) (string, string) {
	switch lang {
	case shared.LanguageC, shared.LanguageCpp, shared.LanguageCsharp, shared.LanguageJava, shared.LanguageJavascript, shared.LanguageGo, shared.LanguageRust, shared.LanguageSwift, shared.LanguageKotlin, shared.LanguageGroovy, shared.LanguageScala, shared.LanguageTypescript, shared.LanguagePhp:
		return "//", ""
	case shared.LanguageBash, shared.LanguageDockerfile, shared.LanguageElixir, shared.LanguageHcl, shared.LanguagePython, shared.LanguageRuby, shared.LanguageToml, shared.LanguageYaml:
		return "#", ""
	case shared.LanguageLua, shared.LanguageElm:
		return "--", ""
	case shared.LanguageCss:
		return "/*", "*/"
	case shared.LanguageHtml:
		return "<!--", "-->"
	case shared.LanguageOCaml:
		return "(*", "*)"
	case shared.LanguageSvelte, shared.LanguageJsx, shared.LanguageTsx, shared.LanguageJson:
		return "", "" // comments are either not allowed or correct symbols depend on the context
	}

	return "", ""
}
