package types

import "github.com/plandex/plandex/shared"

type StreamedFile struct {
	Content string `json:"content"`
}

type StreamedReplacements struct {
	Replacements []*shared.Replacement `json:"replacements"`
}
