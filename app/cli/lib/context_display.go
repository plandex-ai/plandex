package lib

import shared "plandex-shared"

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

func FindContextByIndex(contexts []*shared.Context, index int) *shared.Context {
	// Convert to 0-based index
	index--
	if index < 0 || index >= len(contexts) {
		return nil
	}
	return contexts[index]
}

func FindContextByName(contexts []*shared.Context, name string) *shared.Context {
	for _, ctx := range contexts {
		if ctx.Name == name || ctx.FilePath == name {
			return ctx
		}
	}
	return nil
}
