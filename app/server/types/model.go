package types

import (
	"strings"

	"github.com/plandex/plandex/shared"
)

type StreamedFile struct {
	Content string `json:"content"`
}

type StreamedChangesWithLineNums struct {
	Comments []struct {
		Txt       string `json:"txt"`
		Reference bool   `json:"reference"`
	}
	Problems string                               `json:"problems"`
	Changes  []*shared.StreamedChangeWithLineNums `json:"changes"`
}

// type StreamedChangesFull struct {
// 	Changes []*shared.StreamedChangeFull `json:"changes"`
// }

type StreamedVerifyResult struct {
	SyntaxErrorsReasoning      string `json:"syntaxErrorsReasoning"`
	HasSyntaxErrors            bool   `json:"hasSyntaxErrors"`
	RemovedCodeErrorsReasoning string `json:"removedCodeErrorsReasoning"`
	HasRemovedCodeErrors       bool   `json:"hasRemovedCodeErrors"`
	DuplicationErrorsReasoning string `json:"duplicationErrorsReasoning"`
	HasDuplicationErrors       bool   `json:"hasDuplicationErrors"`
	ReferenceErrorsReasoning   string `json:"referenceErrorsReasoning"`
	HasReferenceErrors         bool   `json:"hasReferenceErrors"`
}

func (s *StreamedVerifyResult) IsCorrect() bool {
	return !s.HasRemovedCodeErrors && !s.HasDuplicationErrors && !s.HasReferenceErrors && !s.HasSyntaxErrors
}

func (s *StreamedVerifyResult) GetReasoning() string {
	res := []string{}

	if s.HasSyntaxErrors {
		res = append(res, s.SyntaxErrorsReasoning)
	}

	if s.HasRemovedCodeErrors {
		res = append(res, s.RemovedCodeErrorsReasoning)
	}

	if s.HasDuplicationErrors {
		res = append(res, s.DuplicationErrorsReasoning)
	}

	if s.HasReferenceErrors {
		res = append(res, s.ReferenceErrorsReasoning)
	}

	return strings.Join(res, "\n")
}
