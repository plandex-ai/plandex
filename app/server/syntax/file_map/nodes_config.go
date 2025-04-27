package file_map

import shared "plandex-shared"

type nodeType string

type matchType string

const (
	matchTypeEqual  matchType = "equal"
	matchTypePrefix matchType = "prefix"
	matchTypeSuffix matchType = "suffix"
)

type langSet map[shared.Language]bool

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
			shared.LanguageGo: true,
		},
	},
	"var_spec": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageGo: true,
		},
	},

	"lexical_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageJavascript: true,
			shared.LanguageTypescript: true,
			shared.LanguageJsx:        true,
			shared.LanguageTsx:        true,
		},
	},

	"field_definition": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageJavascript: true,
			shared.LanguageTypescript: true,
			shared.LanguageJsx:        true,
			shared.LanguageTsx:        true,
		},
	},

	"interface_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageTypescript: true,
			shared.LanguageTsx:        true,
			shared.LanguagePhp:        true,
		},
	},

	"trait_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguagePhp: true,
		},
	},

	"enum_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguagePhp:    true,
			shared.LanguageCsharp: true,
		},
	},

	"global_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguagePhp: true,
		},
	},
	"static_variable_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguagePhp: true,
		},
	},

	"type_alias_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageTypescript: true,
			shared.LanguageTsx:        true,
			shared.LanguageElm:        true,
		},
	},

	"assignment_statement": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguagePython: true,
		},
	},

	"class_variable_assignment": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageRuby: true,
		},
	},
	"assignment": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageRuby: true,
		},
	},

	"let_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageRust: true,
		},
	},
	"const_item": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageRust: true,
		},
	},

	"static_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageJava:   true,
			shared.LanguageCsharp: true,
		},
	},
	"member_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageJava:   true,
			shared.LanguageCsharp: true,
		},
	},
	"field_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageJava:   true,
			shared.LanguageCsharp: true,
		},
	},

	"property_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageKotlin: true,
			shared.LanguageScala:  true,
			shared.LanguagePhp:    true,
			shared.LanguageSwift:  true,
		},
	},

	"let_binding": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageOCaml: true,
		},
	},
	"value_binding": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageOCaml: true,
		},
	},

	"unary_operator": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageElixir: true,
		},
	},

	"value_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageElm: true,
		},
	},

	"declaration_command": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageBash: true,
		},
	},

	"type_item": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageRust: true,
		},
	},

	"val_definition": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageScala: true,
		},
	},

	"type_definition": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageScala: true,
		},
	},

	"typealias_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageSwift: true,
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
			shared.LanguageRuby: true,
		},
	},
	"method": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageRuby: true,
		},
	},
	"defmodule": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageElixir: true,
		},
	},
	"defmacro": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageElixir: true,
		},
	},
	"port": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageElm: true,
		},
	},
	"functor": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageOCaml: true,
		},
	},
	"_def": {
		nodeMatch: matchTypeSuffix,
		languages: langSet{
			shared.LanguageC:   true,
			shared.LanguageCpp: true,
		},
	},
	"type_annotation": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageElm: true,
		},
	},
	"function_statement": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageLua: true,
		},
	},
	"rule_set": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageCss: true,
		},
	},
	"from_instruction": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageDockerfile: true,
		},
	},
	"entrypoint_instruction": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageDockerfile: true,
		},
	},
	"cmd_instruction": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageDockerfile: true,
		},
	},
	"expose_instruction": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageDockerfile: true,
		},
	},
	"copy_instruction": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageDockerfile: true,
		},
	},
	"env_instruction": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageDockerfile: true,
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
			shared.LanguageC:   true,
			shared.LanguageCpp: true,
		},
	},
	"_item": {
		nodeMatch: matchTypeSuffix,
		languages: langSet{
			shared.LanguageRust: true,
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
			shared.LanguageScala: true,
		},
	},
	"object_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageKotlin: true,
		},
	},
	"protocol": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageSwift: true,
		},
	},
	"protocol_": {
		nodeMatch: matchTypePrefix,
		languages: langSet{
			shared.LanguageElixir: true,
		},
	},
	"extension": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageSwift: true,
		},
	},

	"const_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageGo: true,
		},
	},

	"var_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageGo: true,
		},
	},

	"template_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageCpp: true,
		},
	},

	"module": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageRuby: true,
		},
	},

	"class": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageRuby: true,
		},
	},

	"singleton_class": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageRuby: true,
		},
	},

	"enum_entry": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageKotlin: true,
		},
	},

	"closure": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageGroovy: true,
		},
	},

	"impl_item": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageRust: true,
		},
	},

	"trait_item": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageRust: true,
		},
	},

	"trait_definition": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageScala: true,
		},
	},

	"object_definition": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageScala: true,
		},
	},

	"function_statement": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageLua: true,
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
			shared.LanguageRuby:   true,
			shared.LanguageElixir: true,
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
			shared.LanguageRuby:   true,
			shared.LanguageElixir: true,
		},
	},
	"body_statement": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageRuby: true,
		},
	},
	"field_declaration_list": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageGo:  true,
			shared.LanguageC:   true,
			shared.LanguageCpp: true,
		},
	},
	"property_accessors": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageCsharp: true,
			shared.LanguageSwift:  true,
		},
	},
	"const_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageGo: true,
		},
	},
	"var_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageGo: true,
		},
	},
	"method_elem": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageGo: true,
		},
	},

	"preproc_arg": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageC:   true,
			shared.LanguageCpp: true,
		},
	},

	"statement_block": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageJavascript: true,
			shared.LanguageTypescript: true,
			shared.LanguageJsx:        true,
			shared.LanguageTsx:        true,
		},
	},

	"declaration_list": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguagePhp:    true,
			shared.LanguageCsharp: true,
			shared.LanguageRust:   true,
		},
	},

	"_declaration_list": {
		nodeMatch: matchTypeSuffix,
		languages: langSet{
			shared.LanguageRust: true,
		},
	},

	"enum_variant_list": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageRust: true,
		},
	},

	"=": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageScala: true,
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
			shared.LanguageTypescript: true,
			shared.LanguageTsx:        true,
		},
	},
	"declaration_list": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguagePhp:    true,
			shared.LanguageCsharp: true,
		},
	},
	"_declaration_list": {
		nodeMatch: matchTypeSuffix,
		languages: langSet{
			shared.LanguagePhp:    true,
			shared.LanguageCsharp: true,
		},
	},
	"lambda_literal": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageKotlin: true,
		},
	},
	"arguments": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageElixir: true,
		},
	},
	"eq": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageElm: true,
		},
	},
	"statements": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageSwift: true,
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
			shared.LanguageRuby: true,
		},
	},
	"name": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguagePhp: true,
		},
	},
	"bare_key": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageToml: true,
			shared.LanguageYaml: true,
		},
	},
}

var passThroughParentNodeMap = nodeMap{
	"template_declaration": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageCpp: true,
		},
	},
}

var includeAndContinueNodeMap = nodeMap{
	"template_parameter_list": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageCpp: true,
		},
	},
}

var unwrapNodeMap = nodeMap{
	"export_statement": {
		nodeMatch: matchTypeEqual,
		languages: langSet{
			shared.LanguageTypescript: true,
			shared.LanguageTsx:        true,
		},
	},
}
