package types

import (
	"github.com/plandex/plandex/shared"
)

type StreamedFile struct {
	Content string `json:"content"`
}

type StreamedChanges struct {
	References string                   `json:"references"`
	Changes    []*shared.StreamedChange `json:"changes"`
}
