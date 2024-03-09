package shared

type StreamedChangeSection struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine"`
}

type StreamedChangeType int

const (
	StreamedChangeTypeReplace StreamedChangeType = 1
	StreamedChangeTypeAppend  StreamedChangeType = 2
	StreamedChangeTypePrepend StreamedChangeType = 3
)

type StreamedChange struct {
	ShortSummary   string                `json:"shortSummary"`
	ChangeSections string                `json:"changeSections"`
	ChangeType     StreamedChangeType    `json:"changeType"`
	Old            StreamedChangeSection `json:"old"`
	New            StreamedChangeSection `json:"new"`
}
