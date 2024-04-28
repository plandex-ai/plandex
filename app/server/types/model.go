package types

import (
	"github.com/plandex/plandex/shared"
)

type StreamedFile struct {
	Content string `json:"content"`
}

type StreamedChangesWithLineNums struct {
	Changes []*shared.StreamedChangeWithLineNums `json:"changes"`
}

// type StreamedChangesFull struct {
// 	Changes []*shared.StreamedChangeFull `json:"changes"`
// }

type StreamedVerifyResult struct {
	Reasoning string `json:"reasoning"`
	IsCorrect bool   `json:"isCorrect"`
}
