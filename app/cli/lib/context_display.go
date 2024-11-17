package lib

import "github.com/plandex/plandex/shared"

func GetContextLabelAndIcon(contextType shared.ContextType) (string, string) {
	var icon string
	var lbl string
	switch contextType {
	case shared.ContextFileType:
		icon = "📄"
		lbl = "file"
	case shared.ContextURLType:
		icon = "🌎"
		lbl = "url"
	case shared.ContextDirectoryTreeType:
		icon = "🗂 "
		lbl = "tree"
	case shared.ContextNoteType:
		icon = "✏️ "
		lbl = "note"
	case shared.ContextPipedDataType:
		icon = "↔️ "
		lbl = "piped"
	case shared.ContextImageType:
		icon = "🖼️ "
		lbl = "image"
	case shared.ContextMapType:
		icon = "🗺️ "
		lbl = "map"
	}

	return lbl, icon
}
