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
)

func getParserForExt(ext string) (*tree_sitter.Parser, string, *tree_sitter.Parser, string) {
	lang, ok := languageByExtension[ext]
	if !ok {
		return nil, "", nil, ""
	}

	parser := getParserForLanguage(lang)

	fallback := fallbackByExtension[ext]
	var fallbackParser *tree_sitter.Parser
	if fallback != "" {
		fallbackParser = getParserForLanguage(fallback)
	}

	return parser, lang, fallbackParser, fallback
}

func getParserForLanguage(lang string) *tree_sitter.Parser {
	parser := tree_sitter.NewParser()
	switch lang {
	case "bash":
		parser.SetLanguage(bash.GetLanguage())
	case "c":
		parser.SetLanguage(c.GetLanguage())
	case "cpp":
		parser.SetLanguage(cpp.GetLanguage())
	case "csharp":
		parser.SetLanguage(csharp.GetLanguage())
	case "css":
		parser.SetLanguage(css.GetLanguage())
	case "cue":
		parser.SetLanguage(cue.GetLanguage())
	case "dockerfile":
		parser.SetLanguage(dockerfile.GetLanguage())
	case "elixir":
		parser.SetLanguage(elixir.GetLanguage())
	case "elm":
		parser.SetLanguage(elm.GetLanguage())
	case "go":
		parser.SetLanguage(golang.GetLanguage())
	case "groovy":
		parser.SetLanguage(groovy.GetLanguage())
	case "hcl":
		parser.SetLanguage(hcl.GetLanguage())
	case "html":
		parser.SetLanguage(html.GetLanguage())
	case "java":
		parser.SetLanguage(java.GetLanguage())
	case "javascript":
		parser.SetLanguage(javascript.GetLanguage())
	case "kotlin":
		parser.SetLanguage(kotlin.GetLanguage())
	case "lua":
		parser.SetLanguage(lua.GetLanguage())
	case "ocaml":
		parser.SetLanguage(ocaml.GetLanguage())
	case "php":
		parser.SetLanguage(php.GetLanguage())
	case "protobuf":
		parser.SetLanguage(protobuf.GetLanguage())
	case "python":
		parser.SetLanguage(python.GetLanguage())
	case "ruby":
		parser.SetLanguage(ruby.GetLanguage())
	case "rust":
		parser.SetLanguage(rust.GetLanguage())
	case "scala":
		parser.SetLanguage(scala.GetLanguage())
	case "svelte":
		parser.SetLanguage(svelte.GetLanguage())
	case "swift":
		parser.SetLanguage(swift.GetLanguage())
	case "toml":
		parser.SetLanguage(toml.GetLanguage())
	case "typescript":
		parser.SetLanguage(typescript.GetLanguage())
	case "tsx":
		parser.SetLanguage(tsx.GetLanguage())
	case "yaml":
		parser.SetLanguage(yaml.GetLanguage())
	default:
		return nil
	}
	return parser
}

var languageByExtension = map[string]string{
	".sh":         "bash",
	".bash":       "bash",
	".c":          "c",
	".h":          "c",
	".cpp":        "cpp",
	".cc":         "cpp",
	".cs":         "csharp",
	".css":        "css",
	".cue":        "cue",
	".dockerfile": "dockerfile",
	".ex":         "elixir",
	".exs":        "elixir",
	".elm":        "elm",
	".go":         "go",
	".groovy":     "groovy",
	".hcl":        "hcl",
	".html":       "html",
	".java":       "java",
	".js":         "javascript",
	".jsx":        "tsx",
	".kt":         "kotlin",
	".lua":        "lua",
	".ml":         "ocaml",
	".php":        "php",
	".proto":      "protobuf",
	".py":         "python",
	".rb":         "ruby",
	".rs":         "rust",
	".scala":      "scala",
	".svelte":     "svelte",
	".swift":      "swift",
	".toml":       "toml",
	".ts":         "typescript",
	".tsx":        "tsx",
	".yaml":       "yaml",
	".yml":        "yaml",
}

var fallbackByExtension = map[string]string{
	".ts": "tsx",
	".js": "tsx",
}
