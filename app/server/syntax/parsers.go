package syntax

import (
	"path/filepath"
	"strings"

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

func GetLanguageForPath(path string) shared.Language {
	ext := filepath.Ext(path)
	lang, ok := shared.LanguageByExtension[ext]
	if !ok {
		if strings.Contains(strings.ToLower(path), "dockerfile") {
			return shared.LanguageDockerfile
		}
		if strings.Contains(strings.ToLower(path), "rakefile") {
			return shared.LanguageRuby
		}

		if strings.Contains(strings.ToLower(path), "gemfile") {
			return shared.LanguageRuby
		}

		if strings.Contains(strings.ToLower(path), "gemfile.lock") {
			return shared.LanguageRuby
		}

		if strings.Contains(strings.ToLower(path), "gemspec") {
			return shared.LanguageRuby
		}

		if strings.Contains(strings.ToLower(path), "guardfile") {
			return shared.LanguageRuby
		}

		return ""
	}
	return lang
}

func GetParserForPath(path string) (*tree_sitter.Parser, shared.Language, *tree_sitter.Parser, shared.Language) {
	lang := GetLanguageForPath(path)
	if lang == "" {
		return nil, "", nil, ""
	}

	parser := GetParserForLanguage(lang)

	ext := filepath.Ext(path)
	fallback := shared.LanguageFallbackByExtension[ext]
	var fallbackParser *tree_sitter.Parser
	if fallback != "" {
		fallbackParser = GetParserForLanguage(fallback)
	}

	return parser, lang, fallbackParser, fallback
}

func GetParserForLanguage(lang shared.Language) *tree_sitter.Parser {
	parser := tree_sitter.NewParser()
	switch lang {
	case shared.LanguageBash:
		parser.SetLanguage(bash.GetLanguage())
	case shared.LanguageC:
		parser.SetLanguage(c.GetLanguage())
	case shared.LanguageCpp:
		parser.SetLanguage(cpp.GetLanguage())
	case shared.LanguageCsharp:
		parser.SetLanguage(csharp.GetLanguage())
	case shared.LanguageCss:
		parser.SetLanguage(css.GetLanguage())
	case shared.LanguageCue:
		parser.SetLanguage(cue.GetLanguage())
	case shared.LanguageDockerfile:
		parser.SetLanguage(dockerfile.GetLanguage())
	case shared.LanguageElixir:
		parser.SetLanguage(elixir.GetLanguage())
	case shared.LanguageElm:
		parser.SetLanguage(elm.GetLanguage())
	case shared.LanguageGo:
		parser.SetLanguage(golang.GetLanguage())
	case shared.LanguageGroovy:
		parser.SetLanguage(groovy.GetLanguage())
	case shared.LanguageHcl:
		parser.SetLanguage(hcl.GetLanguage())
	case shared.LanguageHtml:
		parser.SetLanguage(html.GetLanguage())
	case shared.LanguageJava:
		parser.SetLanguage(java.GetLanguage())
	case shared.LanguageJavascript, shared.LanguageJson:
		parser.SetLanguage(javascript.GetLanguage())
	case shared.LanguageKotlin:
		parser.SetLanguage(kotlin.GetLanguage())
	case shared.LanguageLua:
		parser.SetLanguage(lua.GetLanguage())
	case shared.LanguageOCaml:
		parser.SetLanguage(ocaml.GetLanguage())
	case shared.LanguagePhp:
		parser.SetLanguage(php.GetLanguage())
	case shared.LanguageProtobuf:
		parser.SetLanguage(protobuf.GetLanguage())
	case shared.LanguagePython:
		parser.SetLanguage(python.GetLanguage())
	case shared.LanguageRuby:
		parser.SetLanguage(ruby.GetLanguage())
	case shared.LanguageRust:
		parser.SetLanguage(rust.GetLanguage())
	case shared.LanguageScala:
		parser.SetLanguage(scala.GetLanguage())
	case shared.LanguageSvelte:
		parser.SetLanguage(svelte.GetLanguage())
	case shared.LanguageSwift:
		parser.SetLanguage(swift.GetLanguage())
	case shared.LanguageToml:
		parser.SetLanguage(toml.GetLanguage())
	case shared.LanguageTypescript:
		parser.SetLanguage(typescript.GetLanguage())
	case shared.LanguageJsx, shared.LanguageTsx:
		parser.SetLanguage(tsx.GetLanguage())
	case shared.LanguageYaml:
		parser.SetLanguage(yaml.GetLanguage())
	default:
		return nil
	}
	return parser
}
