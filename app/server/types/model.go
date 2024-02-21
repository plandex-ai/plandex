package types

type StreamedFile struct {
	Content string `json:"content"`
}

type Section struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine"`
}

type StreamedReplacement struct {
	ShortSummary   string  `json:"shortSummary"`
	ChangeSections string  `json:"changeSections"`
	Old            Section `json:"old"`
	New            Section `json:"new"`
}

type StreamedReplacements struct {
	Replacements []*StreamedReplacement `json:"replacements"`
}
