package syntax

import (
	tree_sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/css"
	"github.com/smacker/go-tree-sitter/cue"
	"github.com/smacker/go-tree-sitter/dockerfile"
	"github.com/smacker/go-tree-sitter/elixir"
	"github.com/smacker/go-tree-sitter/elm"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/groovy"
	"github.com/smacker/go-tree-sitter/hcl"
	"github.com/smacker/go-tree-sitter/html"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/lua"
	"github.com/smacker/go-tree-sitter/ocaml"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/protobuf"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/scala"
	"github.com/smacker/go-tree-sitter/svelte"
	"github.com/smacker/go-tree-sitter/swift"
	"github.com/smacker/go-tree-sitter/toml"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	"github.com/smacker/go-tree-sitter/yaml"

	"github.com/plandex/plandex/shared"
)

func GetParserForExt(ext string) (*tree_sitter.Parser, shared.TreeSitterLanguage, *tree_sitter.Parser, shared.TreeSitterLanguage) {
	lang, ok := shared.TreeSitterLanguageByExtension[ext]
	if !ok {
		return nil, "", nil, ""
	}

	parser := getParserForLanguage(lang)

	fallback := shared.TreeSitterLanguageFallbackByExtension[ext]
	var fallbackParser *tree_sitter.Parser
	if fallback != "" {
		fallbackParser = getParserForLanguage(fallback)
	}

	return parser, lang, fallbackParser, fallback
}

func getParserForLanguage(lang shared.TreeSitterLanguage) *tree_sitter.Parser {
	parser := tree_sitter.NewParser()
	switch lang {
	case shared.TreeSitterLanguageBash:
		parser.SetLanguage(bash.GetLanguage())
	case shared.TreeSitterLanguageC:
		parser.SetLanguage(c.GetLanguage())
	case shared.TreeSitterLanguageCpp:
		parser.SetLanguage(cpp.GetLanguage())
	case shared.TreeSitterLanguageCsharp:
		parser.SetLanguage(csharp.GetLanguage())
	case shared.TreeSitterLanguageCss:
		parser.SetLanguage(css.GetLanguage())
	case shared.TreeSitterLanguageCue:
		parser.SetLanguage(cue.GetLanguage())
	case shared.TreeSitterLanguageDockerfile:
		parser.SetLanguage(dockerfile.GetLanguage())
	case shared.TreeSitterLanguageElixir:
		parser.SetLanguage(elixir.GetLanguage())
	case shared.TreeSitterLanguageElm:
		parser.SetLanguage(elm.GetLanguage())
	case shared.TreeSitterLanguageGo:
		parser.SetLanguage(golang.GetLanguage())
	case shared.TreeSitterLanguageGroovy:
		parser.SetLanguage(groovy.GetLanguage())
	case shared.TreeSitterLanguageHcl:
		parser.SetLanguage(hcl.GetLanguage())
	case shared.TreeSitterLanguageHtml:
		parser.SetLanguage(html.GetLanguage())
	case shared.TreeSitterLanguageJava:
		parser.SetLanguage(java.GetLanguage())
	case shared.TreeSitterLanguageJavascript, shared.TreeSitterLanguageJson:
		parser.SetLanguage(javascript.GetLanguage())
	case shared.TreeSitterLanguageKotlin:
		parser.SetLanguage(kotlin.GetLanguage())
	case shared.TreeSitterLanguageLua:
		parser.SetLanguage(lua.GetLanguage())
	case shared.TreeSitterLanguageOCaml:
		parser.SetLanguage(ocaml.GetLanguage())
	case shared.TreeSitterLanguagePhp:
		parser.SetLanguage(php.GetLanguage())
	case shared.TreeSitterLanguageProtobuf:
		parser.SetLanguage(protobuf.GetLanguage())
	case shared.TreeSitterLanguagePython:
		parser.SetLanguage(python.GetLanguage())
	case shared.TreeSitterLanguageRuby:
		parser.SetLanguage(ruby.GetLanguage())
	case shared.TreeSitterLanguageRust:
		parser.SetLanguage(rust.GetLanguage())
	case shared.TreeSitterLanguageScala:
		parser.SetLanguage(scala.GetLanguage())
	case shared.TreeSitterLanguageSvelte:
		parser.SetLanguage(svelte.GetLanguage())
	case shared.TreeSitterLanguageSwift:
		parser.SetLanguage(swift.GetLanguage())
	case shared.TreeSitterLanguageToml:
		parser.SetLanguage(toml.GetLanguage())
	case shared.TreeSitterLanguageTypescript:
		parser.SetLanguage(typescript.GetLanguage())
	case shared.TreeSitterLanguageJsx, shared.TreeSitterLanguageTsx:
		parser.SetLanguage(tsx.GetLanguage())
	case shared.TreeSitterLanguageYaml:
		parser.SetLanguage(yaml.GetLanguage())
	default:
		return nil
	}
	return parser
}
