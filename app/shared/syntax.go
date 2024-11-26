package shared

import (
	"path/filepath"
	"strings"
)

type TreeSitterLanguage string

const (
	TreeSitterLanguageBash       TreeSitterLanguage = "bash"
	TreeSitterLanguageC          TreeSitterLanguage = "c"
	TreeSitterLanguageCpp        TreeSitterLanguage = "cpp"
	TreeSitterLanguageCsharp     TreeSitterLanguage = "csharp"
	TreeSitterLanguageCss        TreeSitterLanguage = "css"
	TreeSitterLanguageCue        TreeSitterLanguage = "cue"
	TreeSitterLanguageDockerfile TreeSitterLanguage = "dockerfile"
	TreeSitterLanguageElixir     TreeSitterLanguage = "elixir"
	TreeSitterLanguageElm        TreeSitterLanguage = "elm"
	TreeSitterLanguageGo         TreeSitterLanguage = "go"
	TreeSitterLanguageGroovy     TreeSitterLanguage = "groovy"
	TreeSitterLanguageHcl        TreeSitterLanguage = "hcl"
	TreeSitterLanguageHtml       TreeSitterLanguage = "html"
	TreeSitterLanguageJava       TreeSitterLanguage = "java"
	TreeSitterLanguageJavascript TreeSitterLanguage = "javascript"
	TreeSitterLanguageJson       TreeSitterLanguage = "json"
	TreeSitterLanguageKotlin     TreeSitterLanguage = "kotlin"
	TreeSitterLanguageLua        TreeSitterLanguage = "lua"
	TreeSitterLanguageOCaml      TreeSitterLanguage = "ocaml"
	TreeSitterLanguagePhp        TreeSitterLanguage = "php"
	TreeSitterLanguageProtobuf   TreeSitterLanguage = "protobuf"
	TreeSitterLanguagePython     TreeSitterLanguage = "python"
	TreeSitterLanguageRuby       TreeSitterLanguage = "ruby"
	TreeSitterLanguageRust       TreeSitterLanguage = "rust"
	TreeSitterLanguageScala      TreeSitterLanguage = "scala"
	TreeSitterLanguageSvelte     TreeSitterLanguage = "svelte"
	TreeSitterLanguageSwift      TreeSitterLanguage = "swift"
	TreeSitterLanguageToml       TreeSitterLanguage = "toml"
	TreeSitterLanguageTypescript TreeSitterLanguage = "typescript"
	TreeSitterLanguageJsx        TreeSitterLanguage = "jsx"
	TreeSitterLanguageTsx        TreeSitterLanguage = "tsx"
	TreeSitterLanguageYaml       TreeSitterLanguage = "yaml"
)

var TreeSitterLanguages = []TreeSitterLanguage{
	TreeSitterLanguageBash,
	TreeSitterLanguageC,
	TreeSitterLanguageCpp,
	TreeSitterLanguageCsharp,
	TreeSitterLanguageCss,
	TreeSitterLanguageCue,
	TreeSitterLanguageDockerfile,
	TreeSitterLanguageElixir,
	TreeSitterLanguageElm,
	TreeSitterLanguageGo,
	TreeSitterLanguageGroovy,
	TreeSitterLanguageHcl,
	TreeSitterLanguageHtml,
	TreeSitterLanguageJava,
	TreeSitterLanguageJavascript,
	TreeSitterLanguageJson,
	TreeSitterLanguageKotlin,
	TreeSitterLanguageLua,
	TreeSitterLanguageOCaml,
	TreeSitterLanguagePhp,
	TreeSitterLanguageProtobuf,
	TreeSitterLanguagePython,
	TreeSitterLanguageRuby,
	TreeSitterLanguageRust,
	TreeSitterLanguageScala,
	TreeSitterLanguageSvelte,
	TreeSitterLanguageSwift,
	TreeSitterLanguageToml,
	TreeSitterLanguageTypescript,
	TreeSitterLanguageJsx,
	TreeSitterLanguageTsx,
	TreeSitterLanguageYaml,
}

var lacksFileMapSupport = []TreeSitterLanguage{
	// config languages aren't mapped, model decides whether to load them based on file name
	TreeSitterLanguageHcl,
	TreeSitterLanguageYaml,
	TreeSitterLanguageToml,
	TreeSitterLanguageCue,
	TreeSitterLanguageJson,
	TreeSitterLanguageProtobuf,

	// these just need more work for mapping
	TreeSitterLanguageGroovy,
	TreeSitterLanguageOCaml,
}

var TreeSitterLanguageSet = map[TreeSitterLanguage]bool{}

var TreeSitterFileMapSupportSet = map[TreeSitterLanguage]bool{}

func init() {
	for _, lang := range TreeSitterLanguages {
		TreeSitterLanguageSet[lang] = true
		TreeSitterFileMapSupportSet[lang] = true
	}
	for _, lang := range lacksFileMapSupport {
		TreeSitterFileMapSupportSet[lang] = false
	}
}

func IsTreeSitterLanguage(lang TreeSitterLanguage) bool {
	return TreeSitterLanguageSet[lang]
}

func HasTreeSitterSupport(path string) bool {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	isDockerfile := strings.Contains(strings.ToLower(base), "dockerfile")
	return isDockerfile || TreeSitterLanguageByExtension[ext] != ""
}

func HasFileMapSupport(path string) bool {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	isDockerfile := strings.Contains(strings.ToLower(base), "dockerfile")
	lang := TreeSitterLanguageByExtension[ext]
	return isDockerfile || (lang != "" && TreeSitterFileMapSupportSet[lang])
}

var TreeSitterLanguageByExtension = map[string]TreeSitterLanguage{
	".sh":     TreeSitterLanguageBash,
	".bash":   TreeSitterLanguageBash,
	".c":      TreeSitterLanguageC,
	".h":      TreeSitterLanguageC,
	".cpp":    TreeSitterLanguageCpp,
	".cc":     TreeSitterLanguageCpp,
	".cs":     TreeSitterLanguageCsharp,
	".css":    TreeSitterLanguageCss,
	".cue":    TreeSitterLanguageCue,
	".ex":     TreeSitterLanguageElixir,
	".exs":    TreeSitterLanguageElixir,
	".elm":    TreeSitterLanguageElm,
	".go":     TreeSitterLanguageGo,
	".groovy": TreeSitterLanguageGroovy,
	".hcl":    TreeSitterLanguageHcl,
	".html":   TreeSitterLanguageHtml,
	".java":   TreeSitterLanguageJava,
	".js":     TreeSitterLanguageJavascript,
	".json":   TreeSitterLanguageJson,
	".jsx":    TreeSitterLanguageTsx,
	".kt":     TreeSitterLanguageKotlin,
	".lua":    TreeSitterLanguageLua,
	".ml":     TreeSitterLanguageOCaml,
	".php":    TreeSitterLanguagePhp,
	".proto":  TreeSitterLanguageProtobuf,
	".py":     TreeSitterLanguagePython,
	".rb":     TreeSitterLanguageRuby,
	".rs":     TreeSitterLanguageRust,
	".scala":  TreeSitterLanguageScala,
	".svelte": TreeSitterLanguageSvelte,
	".swift":  TreeSitterLanguageSwift,
	".toml":   TreeSitterLanguageToml,
	".ts":     TreeSitterLanguageTypescript,
	".tsx":    TreeSitterLanguageTsx,
	".yaml":   TreeSitterLanguageYaml,
	".yml":    TreeSitterLanguageYaml,
}

var TreeSitterLanguageFallbackByExtension = map[string]TreeSitterLanguage{
	".ts": TreeSitterLanguageTsx,
	".js": TreeSitterLanguageTsx,
}
