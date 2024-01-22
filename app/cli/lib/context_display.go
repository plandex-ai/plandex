package lib

import "github.com/plandex/plandex/shared"

func GetContextTypeAndIcon(context *shared.Context) (string, string) {
	var icon string
	var t string
	switch context.ContextType {
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
