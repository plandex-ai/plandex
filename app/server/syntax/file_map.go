package syntax

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/plandex/plandex/shared"
	tree_sitter "github.com/smacker/go-tree-sitter"
)

// FileMap represents a file's important definitions
type FileMap struct {
	Definitions []Definition
}

type Definition struct {
	Type      string       // "function", "class", "key", "selector", "instruction" etc
	Signature string       // The full signature/header without implementation
	Comments  []string     // Any comments that precede this definition
	TagAttrs  []string     // For xml style markup tags, the class and id attributes
	Line      int          // Line number where definition starts
	Children  []Definition // For parent types that can contain nested definitions
}

func MapFile(ctx context.Context, filename string, content []byte) (*FileMap, error) {
	// Get appropriate parser
	ext := filepath.Ext(filename)
	parser, lang, _, _ := GetParserForExt(ext)
	if parser == nil {
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

	// Parse file
	tree, err := parser.ParseCtx(ctx, nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %v", err)
	}
	defer tree.Close()

	// Create map
	m := &FileMap{
		Definitions: mapNode(tree.RootNode(), content, lang),
	}
	return m, nil
}

func mapNode(node *tree_sitter.Node, content []byte, lang shared.TreeSitterLanguage) []Definition {
	switch lang {
	case shared.TreeSitterLanguageDockerfile:
		return mapDockerfile(node, content)
	case shared.TreeSitterLanguageYaml, shared.TreeSitterLanguageToml, shared.TreeSitterLanguageJson, shared.TreeSitterLanguageCue, shared.TreeSitterLanguageHcl:
		return mapConfig(node, content)
	case shared.TreeSitterLanguageHtml, shared.TreeSitterLanguageSvelte:
		return mapMarkup(node, content)
	case shared.TreeSitterLanguageCss:
		return mapCSS(node, content)
	default:
		return mapTraditional(node, content)
	}
}

// For traditional programming languages
func mapTraditional(node *tree_sitter.Node, content []byte) []Definition {
	var defs []Definition
	cursor := tree_sitter.NewTreeCursor(node)
	defer cursor.Close()

	if cursor.GoToFirstChild() {
		for {
			node := cursor.CurrentNode()
			nodeType := node.Type()

			// Check if this is a definition node
			if isDefinitionNode(nodeType) {
				def := Definition{
					Type: nodeType,
					Line: int(node.StartPoint().Row) + 1,
				}

				if isAssignmentNode(nodeType) {
					// Try different field names for identifiers
					// fmt.Printf("assignment node: %s\n", nodeType)
					sig := ""
					for i := 0; i < int(node.ChildCount()); i++ {
						child := node.Child(i)
						// fmt.Printf("child: %s\n", child.Type())

						if strings.HasSuffix(child.Type(), "_spec") {
							for j := 0; j < int(child.ChildCount()); j++ {
								subChild := child.Child(j)
								// fmt.Printf("sub child: %s\n", subChild.Type())
								if subChild.Type() == "identifier" {
									sig += string(subChild.Content(content)) + " "
								}
							}
							break
						} else {
							sig += string(child.Content(content)) + " "
						}
					}

					def.Signature = sig
				} else {
					// Get signature (up to body)
					if body := findImplementationBoundary(node); body != nil {
						start := node.StartByte()
						end := body.StartByte()
						def.Signature = string(content[start:end])

						// If this is a parent type node, recurse into the body
						if isParentNode(nodeType) {
							def.Children = mapTraditional(body, content)
						}
					} else {
						def.Signature = string(node.Content(content))
					}
				}

				// Get preceding comments
				def.Comments = getPrecedingComments(node, content)

				defs = append(defs, def)
			}

			if !cursor.GoToNextSibling() {
				break
			}
		}
	}

	return defs
}

// Helper functions for definitions and parents
func isDefinitionNode(nodeType string) bool {
	// Ignore import/require/use/include and similar nodes
	if strings.HasPrefix(nodeType, "import_") ||
		strings.HasPrefix(nodeType, "require_") ||
		strings.HasPrefix(nodeType, "use_") ||
		strings.HasPrefix(nodeType, "using_") ||
		strings.HasPrefix(nodeType, "include_") ||
		strings.HasPrefix(nodeType, "package_") ||
		strings.HasPrefix(nodeType, "module_") {
		return false
	}

	if isAssignmentNode(nodeType) ||

		strings.HasSuffix(nodeType, "_definition") ||
		strings.HasSuffix(nodeType, "_declaration") ||
		strings.HasSuffix(nodeType, "_declarator") ||
		strings.HasSuffix(nodeType, "_spec") ||
		strings.HasSuffix(nodeType, "_binding") ||
		strings.HasSuffix(nodeType, "_signature") {

		return true
	}
	return false
}

var parentNodeTypes = map[string]bool{
	"namespace_definition":  true, // C++
	"namespace_declaration": true, // C#
	"module_definition":     true, // Ruby
	"class_definition":      true, // Python/Ruby
	"class_declaration":     true, // Java/C#/TypeScript
	"interface_declaration": true, // Java/TypeScript
	"trait_definition":      true, // Rust
	"trait_declaration":     true, // PHP
	"impl_block":            true, // Rust implementations
}

func isParentNode(nodeType string) bool {
	return parentNodeTypes[nodeType]
}

var assignmentNodeTypes = map[string]bool{
	// Common patterns
	"assignment_expression": true,
	"declaration":           true,
	"variable_declaration":  true,

	// Go
	"var_declaration":   true,
	"const_declaration": true,

	// JS/TS
	"lexical_declaration": true,

	// Python
	"assignment_statement": true,

	// Ruby
	"variable_assignment":       true,
	"class_variable_assignment": true,

	// Rust
	"let_declaration": true,
	"const_item":      true,

	// Java, C#
	"static_declaration": true,
	"member_declaration": true,
	"field_declaration":  true,

	// Kotlin/Scala
	"property_declaration": true,

	// OCaml
	"let_binding":   true,
	"value_binding": true,

	// Elixir
	"module_attribute": true,

	// PHP
	"global_declaration":          true,
	"static_variable_declaration": true,
}

func isAssignmentNode(nodeType string) bool {
	return assignmentNodeTypes[nodeType]
}

// Implementation boundary detection
var bodyNodeTypes = map[string]bool{
	"block":                  true,
	"body":                   true,
	"suite":                  true,
	"statement_block":        true,
	"compound_statement":     true,
	"function_body":          true,
	"do_block":               true,
	"implementation":         true,
	"field_declaration_list": true,
}

func findImplementationBoundary(node *tree_sitter.Node) *tree_sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if bodyNodeTypes[child.Type()] {
			return child
		}
	}
	return nil
}

// Get preceding comments
func getPrecedingComments(node *tree_sitter.Node, content []byte) []string {
	var comments []string
	const maxCommentLength = 1000

	prevNode := node.PrevSibling()
	for prevNode != nil {
		if !strings.Contains(prevNode.Type(), "comment") {
			break
		}
		comment := string(prevNode.Content(content))
		if len(comment) > maxCommentLength {
			comment = comment[:maxCommentLength] + "..."
		}
		comments = append([]string{comment}, comments...)
		prevNode = prevNode.PrevSibling()
	}
	return comments
}

// Specialized mappers
func mapConfig(node *tree_sitter.Node, content []byte) []Definition {
	cursor := tree_sitter.NewTreeCursor(node)
	defer cursor.Close()

	var walkConfig func(*tree_sitter.Node) []Definition
	walkConfig = func(node *tree_sitter.Node) []Definition {
		var nodeDefs []Definition

		if node.Type() == "pair" || // YAML
			node.Type() == "key_value" || // TOML
			node.Type() == "field" { // CUE/HCL
			if keyNode := node.ChildByFieldName("key"); keyNode != nil {
				def := Definition{
					Type:      "key",
					Signature: string(keyNode.Content(content)),
					Line:      int(keyNode.StartPoint().Row) + 1,
				}

				// Check for block values to find nested keys
				if valNode := node.ChildByFieldName("value"); valNode != nil {
					if valNode.Type() == "block_mapping" || // YAML
						valNode.Type() == "block" { // TOML/HCL/CUE
						def.Children = walkConfig(valNode)
					}
				}

				nodeDefs = append(nodeDefs, def)
			}
		}

		// Walk siblings
		for i := 0; i < int(node.ChildCount()); i++ {
			nodeDefs = append(nodeDefs, walkConfig(node.Child(i))...)
		}

		return nodeDefs
	}

	return walkConfig(node)
}

func mapMarkup(node *tree_sitter.Node, content []byte) []Definition {
	cursor := tree_sitter.NewTreeCursor(node)
	defer cursor.Close()

	var walkMarkup func(*tree_sitter.Node) []Definition
	walkMarkup = func(node *tree_sitter.Node) []Definition {
		var nodeDefs []Definition

		if node.Type() == "start_tag" || node.Type() == "self_closing_tag" {
			// Get tag name
			nameNode := node.ChildByFieldName("name")
			if nameNode == nil {
				return nodeDefs
			}

			// Get class and id attributes only
			var attrs []string
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "attribute" {
					attrName := child.ChildByFieldName("name")
					if attrName != nil {
						name := string(attrName.Content(content))
						if name == "class" || name == "id" {
							attrVal := child.ChildByFieldName("value")
							if attrVal != nil {
								attrs = append(attrs, fmt.Sprintf("%s=%q",
									name, attrVal.Content(content)))
							}
						}
					}
				}
			}

			def := Definition{
				Type:      "tag",
				Line:      int(node.StartPoint().Row) + 1,
				Signature: string(nameNode.Content(content)),
				TagAttrs:  attrs,
			}

			// Recursively process children
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				def.Children = append(def.Children, walkMarkup(child)...)
			}

			nodeDefs = append(nodeDefs, def)
		}

		return nodeDefs
	}

	return walkMarkup(node)
}

func mapCSS(node *tree_sitter.Node, content []byte) []Definition {
	var defs []Definition
	cursor := tree_sitter.NewTreeCursor(node)
	defer cursor.Close()

	if cursor.GoToFirstChild() {
		for {
			node := cursor.CurrentNode()
			if node.Type() == "selector" {
				defs = append(defs, Definition{
					Type:      "selector",
					Signature: string(node.Content(content)),
					Line:      int(node.StartPoint().Row) + 1,
				})
			}
			if !cursor.GoToNextSibling() {
				break
			}
		}
	}
	return defs
}

var dockerKeyInstructions = map[string]bool{
	"FROM":       true,
	"ENTRYPOINT": true,
	"CMD":        true,
	"EXPOSE":     true,
	"COPY":       true,
	"ENV":        true,
}

func mapDockerfile(node *tree_sitter.Node, content []byte) []Definition {
	var defs []Definition
	cursor := tree_sitter.NewTreeCursor(node)
	defer cursor.Close()

	if cursor.GoToFirstChild() {
		for {
			node := cursor.CurrentNode()
			if node.Type() == "instruction" {
				instruction := string(node.Content(content))
				command := strings.Fields(instruction)[0]
				if dockerKeyInstructions[command] {
					defs = append(defs, Definition{
						Type:      "instruction",
						Signature: instruction,
						Line:      int(node.StartPoint().Row) + 1,
					})
				}
			}
			if !cursor.GoToNextSibling() {
				break
			}
		}
	}
	return defs
}

func (m *FileMap) String() string {
	var b strings.Builder

	var writeDefinition func(def *Definition, depth int)
	writeDefinition = func(def *Definition, depth int) {
		// Indent
		if depth > 0 {
			b.WriteString(strings.Repeat("  ", depth))
			b.WriteString("- ")
		}

		// Write signature (for tags, include attrs)
		if def.Type == "tag" {
			// Extract tag name from signature (it's the first word)
			tagName := strings.Fields(def.Signature)[0]
			// Build full representation with attrs
			if len(def.TagAttrs) > 0 {
				b.WriteString(fmt.Sprintf("%s %s", tagName, strings.Join(def.TagAttrs, " ")))
			} else {
				b.WriteString(tagName)
			}
		} else {
			b.WriteString(strings.TrimSpace(def.Signature))
		}
		b.WriteString("\n")

		// Write children with increased depth
		for _, child := range def.Children {
			writeDefinition(&child, depth+1)
		}
	}

	// Write all top-level definitions
	for _, def := range m.Definitions {
		writeDefinition(&def, 0)
	}

	return b.String()
}
