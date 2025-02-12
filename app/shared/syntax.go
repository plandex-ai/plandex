package shared

import (
	"path/filepath"
	"strings"
)

type Language string

const (
	LanguageBash       Language = "bash"
	LanguageC          Language = "c"
	LanguageCpp        Language = "cpp"
	LanguageCsharp     Language = "csharp"
	LanguageCss        Language = "css"
	LanguageCue        Language = "cue"
	LanguageDockerfile Language = "dockerfile"
	LanguageElixir     Language = "elixir"
	LanguageElm        Language = "elm"
	LanguageGo         Language = "go"
	LanguageGroovy     Language = "groovy"
	LanguageHcl        Language = "hcl"
	LanguageHtml       Language = "html"
	LanguageJava       Language = "java"
	LanguageJavascript Language = "javascript"
	LanguageJson       Language = "json"
	LanguageKotlin     Language = "kotlin"
	LanguageLua        Language = "lua"
	LanguageOCaml      Language = "ocaml"
	LanguagePhp        Language = "php"
	LanguageProtobuf   Language = "protobuf"
	LanguagePython     Language = "python"
	LanguageRuby       Language = "ruby"
	LanguageRust       Language = "rust"
	LanguageScala      Language = "scala"
	LanguageSvelte     Language = "svelte"
	LanguageSwift      Language = "swift"
	LanguageToml       Language = "toml"
	LanguageTypescript Language = "typescript"
	LanguageJsx        Language = "jsx"
	LanguageTsx        Language = "tsx"
	LanguageYaml       Language = "yaml"
	LanguageMarkdown   Language = "markdown"
)

var Languages = []Language{
	LanguageBash,
	LanguageC,
	LanguageCpp,
	LanguageCsharp,
	LanguageCss,
	LanguageCue,
	LanguageDockerfile,
	LanguageElixir,
	LanguageElm,
	LanguageGo,
	LanguageGroovy,
	LanguageHcl,
	LanguageHtml,
	LanguageJava,
	LanguageJavascript,
	LanguageJson,
	LanguageKotlin,
	LanguageLua,
	LanguageMarkdown,
	LanguageOCaml,
	LanguagePhp,
	LanguageProtobuf,
	LanguagePython,
	LanguageRuby,
	LanguageRust,
	LanguageScala,
	LanguageSvelte,
	LanguageSwift,
	LanguageToml,
	LanguageTypescript,
	LanguageJsx,
	LanguageTsx,
	LanguageYaml,
}

var lacksFileMapSupport = []Language{
	// config languages aren't mapped, model decides whether to load them based on file name
	LanguageHcl,
	LanguageYaml,
	LanguageToml,
	LanguageCue,
	LanguageJson,
	LanguageProtobuf,

	// these just need more work for mapping
	LanguageGroovy,
	LanguageOCaml,
}

var SkipTreeSitter = map[Language]bool{
	LanguageMarkdown: true,
}

var LanguageSet = map[Language]bool{}

var FileMapSupportSet = map[Language]bool{}

func init() {
	for _, lang := range Languages {
		LanguageSet[lang] = true
		FileMapSupportSet[lang] = true
	}
	for _, lang := range lacksFileMapSupport {
		FileMapSupportSet[lang] = false
	}
}

func IsTreeSitterLanguage(lang Language) bool {
	return LanguageSet[lang] && !SkipTreeSitter[lang]
}

func HasTreeSitterSupport(path string) bool {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	isDockerfile := strings.Contains(strings.ToLower(base), "dockerfile")
	return (isDockerfile || LanguageByExtension[ext] != "") && IsTreeSitterLanguage(LanguageByExtension[ext])
}

func HasFileMapSupport(path string) bool {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	isDockerfile := strings.Contains(strings.ToLower(base), "dockerfile")
	lang := LanguageByExtension[ext]
	return isDockerfile || (lang != "" && FileMapSupportSet[lang])
}

var LanguageByExtension = map[string]Language{
	".sh":     LanguageBash,
	".bash":   LanguageBash,
	".c":      LanguageC,
	".h":      LanguageC,
	".cpp":    LanguageCpp,
	".cc":     LanguageCpp,
	".cs":     LanguageCsharp,
	".css":    LanguageCss,
	".cue":    LanguageCue,
	".ex":     LanguageElixir,
	".exs":    LanguageElixir,
	".elm":    LanguageElm,
	".go":     LanguageGo,
	".groovy": LanguageGroovy,
	".hcl":    LanguageHcl,
	".html":   LanguageHtml,
	".java":   LanguageJava,
	".js":     LanguageJavascript,
	".json":   LanguageJson,
	".jsx":    LanguageTsx,
	".kt":     LanguageKotlin,
	".lua":    LanguageLua,
	".ml":     LanguageOCaml,
	".php":    LanguagePhp,
	".proto":  LanguageProtobuf,
	".py":     LanguagePython,
	".rb":     LanguageRuby,
	".rs":     LanguageRust,
	".scala":  LanguageScala,
	".svelte": LanguageSvelte,
	".swift":  LanguageSwift,
	".toml":   LanguageToml,
	".ts":     LanguageTypescript,
	".tsx":    LanguageTsx,
	".yaml":   LanguageYaml,
	".yml":    LanguageYaml,
	".md":     LanguageMarkdown,
}

var LanguageFallbackByExtension = map[string]Language{
	".ts": LanguageTsx,
	".js": LanguageTsx,
}
