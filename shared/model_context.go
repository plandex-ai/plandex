package shared

type ModelContextPartType string

const (
	ContextFileType          ModelContextPartType = "file"
	ContextURLType           ModelContextPartType = "url"
	ContextNoteType          ModelContextPartType = "note"
	ContextDirectoryTreeType ModelContextPartType = "directory tree"
	ContextPipedDataType     ModelContextPartType = "piped data"
)

type ModelContextPart struct {
	Type      ModelContextPartType `json:"type"`
	Name      string               `json:"name"`
	Body      string               `json:"body"`
	Url       string               `json:"url"`
	FilePath  string               `json:"filePath"`
	Sha       string               `json:"sha"`
	NumTokens int                  `json:"numTokens"`
	AddedAt   string               `json:"addedAt"`
	UpdatedAt string               `json:"updatedAt"`
}
type ModelContext []*ModelContextPart

func (c ModelContext) ByPath() map[string]*ModelContextPart {
	byPath := make(map[string]*ModelContextPart)
	for _, part := range c {
		if part.FilePath != "" {
			byPath[part.FilePath] = part
		}
	}
	return byPath
}
