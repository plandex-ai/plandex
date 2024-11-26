package file_map

import (
	"fmt"
	"strings"

	"github.com/plandex/plandex/shared"
)

func isDefinitionNode(node Node, parentNode *Node) bool {
	setNodeType(&node)

	res := false

	config := definitionNodeMap.getConfig(node.Type, node.Lang)

	var parentConfig *nodeConfig
	if parentNode != nil {
		parentConfig = parentNodeMap.getConfig(parentNode.Type, parentNode.Lang)
	}

	if config == nil {
		res = isAssignmentNode(node) || isParentNode(node)
		if res && parentConfig != nil && parentConfig.onlyChildren != nil {
			res = parentConfig.onlyChildren[nodeType(node.Type)]
		}
		return res
	}

	res = !config.ignore
	if res {
		if parentConfig != nil && parentConfig.onlyChildren != nil {
			res = parentConfig.onlyChildren[nodeType(node.Type)]
		}
	}
	return res
}

func isAssignmentNode(node Node) bool {
	setNodeType(&node)
	config := assignmentNodeMap.getConfig(node.Type, node.Lang)
	if config == nil {
		return false
	}
	return !config.ignore
}

func isParentNode(node Node) bool {
	setNodeType(&node)
	config := parentNodeMap.getConfig(node.Type, node.Lang)
	if config == nil {
		return false
	}
	return !config.ignore
}

func isImplBoundaryNode(node Node) bool {
	setNodeType(&node)
	config := implBoundaryNodeMap.getConfig(node.Type, node.Lang)
	if config == nil {
		return false
	}
	return !config.ignore
}

func isAssignmentBoundaryNode(node Node) bool {
	setNodeType(&node)
	config := assignmentBoundaryNodeMap.getConfig(node.Type, node.Lang)
	if config == nil {
		return false
	}
	return !config.ignore
}

func isIdentifierNode(node Node) bool {
	setNodeType(&node)
	config := identifierNodeMap.getConfig(node.Type, node.Lang)
	if config == nil {
		return false
	}
	return !config.ignore
}

func isPassThroughParentNode(node Node) bool {
	setNodeType(&node)
	config := passThroughParentNodeMap.getConfig(node.Type, node.Lang)
	if config == nil {
		return false
	}
	return !config.ignore
}

func isIncludeAndContinueNode(node Node) bool {
	setNodeType(&node)
	config := includeAndContinueNodeMap.getConfig(node.Type, node.Lang)
	if config == nil {
		return false
	}
	return !config.ignore
}

func (m nodeMap) getConfig(t string, lang shared.TreeSitterLanguage) *nodeConfig {
	// first look for exact match
	config, ok := m[nodeType(t)]
	if ok {
		if config.all {
			if config.except == nil || !config.except[lang] {
				return &config
			}
		}

		if config.languages != nil && config.languages[lang] {
			return &config
		}
	}

	// then look for prefix or suffix match
	var foundConfig *nodeConfig
	for k, config := range m {
		var maybeConfig *nodeConfig
		if config.nodeMatch == matchTypePrefix && strings.HasPrefix(t, string(k)) {
			maybeConfig = &config
		} else if config.nodeMatch == matchTypeSuffix && strings.HasSuffix(t, string(k)) {
			maybeConfig = &config
		}

		var wouldSet *nodeConfig
		if maybeConfig != nil {
			if maybeConfig.all {
				if maybeConfig.except == nil || !maybeConfig.except[lang] {
					wouldSet = maybeConfig
				}
			} else if maybeConfig.languages != nil && maybeConfig.languages[lang] {
				wouldSet = maybeConfig
			}
		}

		if wouldSet != nil {
			if foundConfig != nil {
				// ignore takes precedence
				if wouldSet.ignore {
					foundConfig = wouldSet
				}
			} else {
				foundConfig = wouldSet
			}
		}
	}

	return foundConfig
}

func findImplementationBoundary(node Node) *Node {
	if isImplBoundaryNode(node) {
		return &node
	}

	for i := 0; i < int(node.TsNode.ChildCount()); i++ {
		child := node.TsNode.Child(i)

		if verboseLogging {
			fmt.Println("  findImplementationBoundary child", child.Type())
		}

		childNode := Node{
			Type:   child.Type(),
			Lang:   node.Lang,
			TsNode: child,
			Bytes:  node.Bytes,
		}

		if isImplBoundaryNode(childNode) {
			return &childNode
		} else {
			found := findImplementationBoundary(childNode)
			if found != nil {
				return found
			}
		}
	}
	return nil
}

func findIdentifier(node Node) []Node {
	if isIdentifierNode(node) {
		return []Node{node}
	}

	nodes := []Node{}
	for i := 0; i < int(node.TsNode.ChildCount()); i++ {
		child := node.TsNode.Child(i)
		childNode := Node{
			Type:   child.Type(),
			Lang:   node.Lang,
			TsNode: child,
			Bytes:  node.Bytes,
		}

		if verboseLogging {
			fmt.Println("  child", child.Type())
		}

		found := findIdentifier(childNode)
		if found != nil {
			nodes = append(nodes, found...)
		}
	}
	return nodes
}

func firstDefinitionChild(node Node) *Node {
	for i := 0; i < int(node.TsNode.ChildCount()); i++ {
		child := node.TsNode.Child(i)

		if verboseLogging {
			fmt.Println("  child", child.Type())
		}

		childNode := Node{
			Type:   child.Type(),
			Lang:   node.Lang,
			TsNode: child,
			Bytes:  node.Bytes,
		}

		if isDefinitionNode(childNode, &node) && !isIncludeAndContinueNode(childNode) {
			return &childNode
		}
	}
	return nil
}

func findAssignmentBoundary(node Node) *Node {
	if isAssignmentBoundaryNode(node) {
		return &node
	}

	for i := 0; i < int(node.TsNode.ChildCount()); i++ {
		child := node.TsNode.Child(i)

		childNode := Node{
			Type:   child.Type(),
			Lang:   node.Lang,
			TsNode: child,
			Bytes:  node.Bytes,
		}

		if verboseLogging {
			fmt.Println("  child", child.Type())
		}

		found := findAssignmentBoundary(childNode)
		if found != nil {
			return found
		}
	}
	return nil
}

func setNodeType(node *Node) {
	if node.Lang == shared.TreeSitterLanguageElixir && node.Type == "call" {
		content := node.TsNode.Content(node.Bytes)

		switch {
		case strings.HasPrefix(string(content), "defmodule"):
			node.Type = "module_definition"
		case strings.HasPrefix(string(content), "defprotocol"):
			node.Type = "protocol_definition"
		case strings.HasPrefix(string(content), "defimpl"):
			node.Type = "protocol_implementation"
		case strings.HasPrefix(string(content), "defstruct"):
			node.Type = "struct_definition"
		case strings.HasPrefix(string(content), "defexception"):
			node.Type = "exception_definition"
		case strings.HasPrefix(string(content), "defdelegate"):
			node.Type = "delegate_definition"
		case strings.HasPrefix(string(content), "defoverridable"):
			node.Type = "overridable_definition"
		case strings.HasPrefix(string(content), "defcallback"):
			node.Type = "callback_definition"
		case strings.HasPrefix(string(content), "defmacrocallback"):
			node.Type = "macro_callback_definition"
		case strings.HasPrefix(string(content), "defmacrop"):
			node.Type = "private_macro_definition"
		case strings.HasPrefix(string(content), "defmacro"):
			node.Type = "macro_definition"
		case strings.HasPrefix(string(content), "defguardp"):
			node.Type = "private_guard_definition"
		case strings.HasPrefix(string(content), "defguard"):
			node.Type = "guard_definition"
		case strings.HasPrefix(string(content), "defp"):
			node.Type = "private_function_definition"
		case strings.HasPrefix(string(content), "def"):
			node.Type = "function_definition"
		}
	}
}
