package lib

import "github.com/plandex/plandex/shared"

func GetContextTypeAndIcon(part *shared.ModelContextPart) (string, string) {
	var icon string
	var t string
	switch part.Type {
	case shared.ContextFileType:
		icon = "📄"
		t = "file"
	case shared.ContextURLType:
		icon = "🌎"
		t = "url"
	case shared.ContextDirectoryTreeType:
		icon = "🗂 "
		t = "tree"
	case shared.ContextNoteType:
		icon = "✏️ "
		t = "note"
	case shared.ContextPipedDataType:
		icon = "↔️ "
		t = "piped"
	}

	return t, icon
}
