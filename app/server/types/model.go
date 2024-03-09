package types

import (
	"github.com/plandex/plandex/shared"
)

type StreamedFile struct {
	Content string `json:"content"`
}

type StreamedChanges struct {
	Changes []*shared.StreamedChange `json:"changes"`
}
