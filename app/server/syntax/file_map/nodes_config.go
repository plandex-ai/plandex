package file_map

import "github.com/plandex/plandex/shared"

type nodeType string

type matchType string

const (
	matchTypeEqual  matchType = "equal"
	matchTypePrefix matchType = "prefix"
	matchTypeSuffix matchType = "suffix"
)

type langSet map[shared.TreeSitterLanguage]bool

type nodeConfig struct {
	nodeMatch    matchType
	ignore       bool
	all          bool
	except       langSet
	languages    langSet
	onlyChildren map[nodeType]bool
}

type nodeMap map[nodeType]nodeConfig

var assignmentNodeMap = nodeMap{
	// Common patterns across many languages
	"assignment_expression": {
		nodeMatch: matchTypeEqual,
		all:       true,
	},
	"declaration": {
		nodeMatch: matchTypeEqual,
		all:       true,
	},
	"variable_declaration": {
		nodeMatch: matchTypeEqual,
		all:       true,
	},
	"variable_assignment": {
		nodeMatch: matchTypeEqual,
		all:       true,
	},

	"const_spec": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageGo: true,
		},
	},
	"var_spec": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageGo: true,
		},
	},

	"lexical_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageJavascript: true,
			shared.TreeSitterLanguageTypescript: true,
			shared.TreeSitterLanguageJsx:        true,
			shared.TreeSitterLanguageTsx:        true,
		},
	},

	"field_definition": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageJavascript: true,
			shared.TreeSitterLanguageTypescript: true,
			shared.TreeSitterLanguageJsx:        true,
			shared.TreeSitterLanguageTsx:        true,
		},
	},

	"interface_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageTypescript: true,
			shared.TreeSitterLanguageTsx:        true,
			shared.TreeSitterLanguagePhp:        true,
		},
	},

	"trait_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguagePhp: true,
		},
	},

	"enum_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguagePhp:    true,
			shared.TreeSitterLanguageCsharp: true,
		},
	},

	"global_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguagePhp: true,
		},
	},
	"static_variable_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguagePhp: true,
		},
	},

	"type_alias_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageTypescript: true,
			shared.TreeSitterLanguageTsx:        true,
			shared.TreeSitterLanguageElm:        true,
		},
	},

	"assignment_statement": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguagePython: true,
		},
	},

	"class_variable_assignment": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRuby: true,
		},
	},
	"assignment": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRuby: true,
		},
	},

	"let_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRust: true,
		},
	},
	"const_item": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRust: true,
		},
	},

	"static_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageJava:   true,
			shared.TreeSitterLanguageCsharp: true,
		},
	},
	"member_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageJava:   true,
			shared.TreeSitterLanguageCsharp: true,
		},
	},
	"field_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageJava:   true,
			shared.TreeSitterLanguageCsharp: true,
		},
	},

	"property_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageKotlin: true,
			shared.TreeSitterLanguageScala:  true,
			shared.TreeSitterLanguagePhp:    true,
			shared.TreeSitterLanguageSwift:  true,
		},
	},

	"let_binding": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageOCaml: true,
		},
	},
	"value_binding": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageOCaml: true,
		},
	},

	"unary_operator": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageElixir: true,
		},
	},

	"value_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageElm: true,
		},
	},

	"declaration_command": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageBash: true,
		},
	},

	"type_item": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRust: true,
		},
	},

	"val_definition": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageScala: true,
		},
	},

	"type_definition": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageScala: true,
		},
	},

	"typealias_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageSwift: true,
		},
	},
}

var definitionNodeMap = nodeMap{
	// Common patterns across languages
	"_definition": {
		nodeMatch: matchTypeSuffix,
		all:       true,
	},
	"_declaration": {
		nodeMatch: matchTypeSuffix,
		all:       true,
	},
	"_declarator": {
		nodeMatch: matchTypeSuffix,
		all:       true,
	},
	"_spec": {
		nodeMatch: matchTypeSuffix,
		all:       true,
	},
	"_binding": {
		nodeMatch: matchTypeSuffix,
		all:       true,
	},
	"_signature": {
		nodeMatch: matchTypeSuffix,
		all:       true,
	},
	"declaration": {
		nodeMatch: matchTypeEqual,
		all:       true,
	},
	"_specifier": {
		nodeMatch: matchTypeSuffix,
		all:       true,
	},
	"interface_": {
		nodeMatch: matchTypePrefix,
		all:       true,
	},

	// Language-specific definitions
	"def": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRuby: true,
		},
	},
	"method": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRuby: true,
		},
	},
	"defmodule": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageElixir: true,
		},
	},
	"defmacro": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageElixir: true,
		},
	},
	"port": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageElm: true,
		},
	},
	"functor": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageOCaml: true,
		},
	},
	"_def": {
		nodeMatch: matchTypeSuffix,
		languages: langSet{
			shared.TreeSitterLanguageC:   true,
			shared.TreeSitterLanguageCpp: true,
		},
	},
	"type_annotation": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageElm: true,
		},
	},
	"function_statement": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageLua: true,
		},
	},
	"rule_set": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageCss: true,
		},
	},
	"from_instruction": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageDockerfile: true,
		},
	},
	"entrypoint_instruction": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageDockerfile: true,
		},
	},
	"cmd_instruction": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageDockerfile: true,
		},
	},
	"expose_instruction": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageDockerfile: true,
		},
	},
	"copy_instruction": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageDockerfile: true,
		},
	},
	"env_instruction": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageDockerfile: true,
		},
	},

	// Ignored patterns
	"import_": {
		nodeMatch: matchTypePrefix,
		ignore:    true,
		all:       true,
	},
	"require_": {
		nodeMatch: matchTypePrefix,
		ignore:    true,
		all:       true,
	},
	"use_": {
		nodeMatch: matchTypePrefix,
		ignore:    true,
		all:       true,
	},
	"namespace_use_": {
		nodeMatch: matchTypePrefix,
		ignore:    true,
		all:       true,
	},
	"using_": {
		nodeMatch: matchTypePrefix,
		ignore:    true,
		all:       true,
	},
	"include_": {
		nodeMatch: matchTypePrefix,
		ignore:    true,
		all:       true,
	},
	"package_": {
		nodeMatch: matchTypePrefix,
		ignore:    true,
		all:       true,
	},
	"_include": {
		nodeMatch: matchTypeSuffix,
		ignore:    true,
		languages: langSet{
			shared.TreeSitterLanguageC:   true,
			shared.TreeSitterLanguageCpp: true,
		},
	},
	"_item": {
		nodeMatch: matchTypeSuffix,
		languages: langSet{
			shared.TreeSitterLanguageRust: true,
		},
	},
}

var parentNodeMap = nodeMap{
	// Common patterns
	"class_": {
		nodeMatch: matchTypePrefix,
		all:       true,
	},
	"module_": {
		nodeMatch: matchTypePrefix,
		all:       true,
	},
	"namespace_": {
		nodeMatch: matchTypePrefix,
		all:       true,
	},

	// Language-specific parents
	"object": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageScala: true,
		},
	},
	"object_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageKotlin: true,
		},
	},
	"protocol": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageSwift: true,
		},
	},
	"protocol_": {
		nodeMatch: matchTypePrefix,
		languages: langSet{
			shared.TreeSitterLanguageElixir: true,
		},
	},
	"extension": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageSwift: true,
		},
	},

	"const_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageGo: true,
		},
	},

	"var_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageGo: true,
		},
	},

	"template_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageCpp: true,
		},
	},

	"module": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRuby: true,
		},
	},

	"class": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRuby: true,
		},
	},

	"singleton_class": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRuby: true,
		},
	},

	"enum_entry": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageKotlin: true,
		},
	},

	"closure": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageGroovy: true,
		},
	},

	"impl_item": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRust: true,
		},
	},

	"trait_item": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRust: true,
		},
	},

	"trait_definition": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageScala: true,
		},
	},

	"object_definition": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageScala: true,
		},
	},

	"function_statement": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageLua: true,
		},
		onlyChildren: map[nodeType]bool{
			"function_statement": true,
		},
	},
}

var implBoundaryNodeMap = nodeMap{
	// Common patterns
	"block": {
		nodeMatch: matchTypeEqual,
		all:       true,
		except: langSet{
			shared.TreeSitterLanguageRuby:   true,
			shared.TreeSitterLanguageElixir: true,
		},
	},
	"body": {
		nodeMatch: matchTypeEqual,
		all:       true,
	},
	"_body": {
		nodeMatch: matchTypeSuffix,
		all:       true,
	},
	"compound_statement": {
		nodeMatch: matchTypeEqual,
		all:       true,
	},

	// Language-specific boundaries
	"do_block": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRuby:   true,
			shared.TreeSitterLanguageElixir: true,
		},
	},
	"body_statement": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRuby: true,
		},
	},
	"field_declaration_list": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageGo:  true,
			shared.TreeSitterLanguageC:   true,
			shared.TreeSitterLanguageCpp: true,
		},
	},
	"property_accessors": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageCsharp: true,
			shared.TreeSitterLanguageSwift:  true,
		},
	},
	"const_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageGo: true,
		},
	},
	"var_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageGo: true,
		},
	},
	"method_elem": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageGo: true,
		},
	},

	"preproc_arg": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageC:   true,
			shared.TreeSitterLanguageCpp: true,
		},
	},

	"statement_block": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageJavascript: true,
			shared.TreeSitterLanguageTypescript: true,
			shared.TreeSitterLanguageJsx:        true,
			shared.TreeSitterLanguageTsx:        true,
		},
	},

	"declaration_list": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguagePhp:    true,
			shared.TreeSitterLanguageCsharp: true,
			shared.TreeSitterLanguageRust:   true,
		},
	},

	"_declaration_list": {
		nodeMatch: matchTypeSuffix,
		languages: langSet{
			shared.TreeSitterLanguageRust: true,
		},
	},

	"enum_variant_list": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRust: true,
		},
	},

	"=": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageScala: true,
		},
	},
}

var assignmentBoundaryNodeMap = nodeMap{
	"=": {
		nodeMatch: matchTypeEqual,
		all:       true,
	},
	":=": {
		nodeMatch: matchTypeEqual,
		all:       true,
	},
	"<-": {
		nodeMatch: matchTypeEqual,
		all:       true,
	},
	"expression": {
		nodeMatch: matchTypeEqual,
		all:       true,
	},
	"interface_body": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageTypescript: true,
			shared.TreeSitterLanguageTsx:        true,
		},
	},
	"declaration_list": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguagePhp:    true,
			shared.TreeSitterLanguageCsharp: true,
		},
	},
	"_declaration_list": {
		nodeMatch: matchTypeSuffix,
		languages: langSet{
			shared.TreeSitterLanguagePhp:    true,
			shared.TreeSitterLanguageCsharp: true,
		},
	},
	"lambda_literal": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageKotlin: true,
		},
	},
	"arguments": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageElixir: true,
		},
	},
	"eq": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageElm: true,
		},
	},
	"statements": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageSwift: true,
		},
	},
}

var identifierNodeMap = nodeMap{
	// Common patterns
	"identifier": {
		nodeMatch: matchTypeEqual,
		all:       true,
	},
	"_identifier": {
		nodeMatch: matchTypeSuffix,
		all:       true,
	},

	// Language-specific identifiers
	"constant": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageRuby: true,
		},
	},
	"name": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguagePhp: true,
		},
	},
	"bare_key": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageToml: true,
			shared.TreeSitterLanguageYaml: true,
		},
	},
}

var passThroughParentNodeMap = nodeMap{
	"template_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageCpp: true,
		},
	},
}

var includeAndContinueNodeMap = nodeMap{
	"template_parameter_list": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.TreeSitterLanguageCpp: true,
		},
	},
}
