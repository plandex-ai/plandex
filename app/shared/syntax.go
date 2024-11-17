package shared

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

var TreeSitterLanguageSet = map[TreeSitterLanguage]bool{}

func init() {
	for _, lang := range TreeSitterLanguages {
		TreeSitterLanguageSet[lang] = true
	}
}

func IsTreeSitterLanguage(lang TreeSitterLanguage) bool {
	return TreeSitterLanguageSet[lang]
}

func IsTreeSitterExtension(ext string) bool {
	return TreeSitterLanguageByExtension[ext] != ""
}

var TreeSitterLanguageByExtension = map[string]TreeSitterLanguage{
	".sh":         TreeSitterLanguageBash,
	".bash":       TreeSitterLanguageBash,
	".c":          TreeSitterLanguageC,
	".h":          TreeSitterLanguageC,
	".cpp":        TreeSitterLanguageCpp,
	".cc":         TreeSitterLanguageCpp,
	".cs":         TreeSitterLanguageCsharp,
	".css":        TreeSitterLanguageCss,
	".cue":        TreeSitterLanguageCue,
	".dockerfile": TreeSitterLanguageDockerfile,
	".ex":         TreeSitterLanguageElixir,
	".exs":        TreeSitterLanguageElixir,
	".elm":        TreeSitterLanguageElm,
	".go":         TreeSitterLanguageGo,
	".groovy":     TreeSitterLanguageGroovy,
	".hcl":        TreeSitterLanguageHcl,
	".html":       TreeSitterLanguageHtml,
	".java":       TreeSitterLanguageJava,
	".js":         TreeSitterLanguageJavascript,
	".json":       TreeSitterLanguageJson,
	".jsx":        TreeSitterLanguageTsx,
	".kt":         TreeSitterLanguageKotlin,
	".lua":        TreeSitterLanguageLua,
	".ml":         TreeSitterLanguageOCaml,
	".php":        TreeSitterLanguagePhp,
	".proto":      TreeSitterLanguageProtobuf,
	".py":         TreeSitterLanguagePython,
	".rb":         TreeSitterLanguageRuby,
	".rs":         TreeSitterLanguageRust,
	".scala":      TreeSitterLanguageScala,
	".svelte":     TreeSitterLanguageSvelte,
	".swift":      TreeSitterLanguageSwift,
	".toml":       TreeSitterLanguageToml,
	".ts":         TreeSitterLanguageTypescript,
	".tsx":        TreeSitterLanguageTsx,
	".yaml":       TreeSitterLanguageYaml,
	".yml":        TreeSitterLanguageYaml,
}

var TreeSitterLanguageFallbackByExtension = map[string]TreeSitterLanguage{
	".ts": TreeSitterLanguageTsx,
	".js": TreeSitterLanguageTsx,
}
