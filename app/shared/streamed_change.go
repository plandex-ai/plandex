package shared

type StreamedChangeSection struct {
	MaybeStartLine int    `json:"maybeStartLine"`
	MaybeEndLine   int    `json:"maybeEndLine"`
	Err            string `json:"err"`

	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine"`
}

// type StreamedChangeType int

// const (
// 	StreamedChangeTypeSimple  StreamedChangeType = 1
// 	StreamedChangeTypeComplex StreamedChangeType = 2
// )

type StreamedChange struct {
	Summary string `json:"summary"`
	Section string `json:"section"`
	// ChangeType     StreamedChangeType    `json:"changeType"`
	Old StreamedChangeSection `json:"old"`
	// New            StreamedChangeSection `json:"new"`
	New string `json:"new"`
}
