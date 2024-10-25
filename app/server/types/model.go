package types

import (
	"fmt"
	"strings"

	"github.com/plandex/plandex/shared"
)

type StreamedFile struct {
	Content string `json:"content"`
}

type ChangesWithLineNums struct {
	Comments []struct {
		Txt       string `json:"txt"`
		Reference bool   `json:"reference"`
	}
	Problems string                               `json:"problems"`
	Changes  []*shared.StreamedChangeWithLineNums `json:"changes"`
}

type UpdatedChangesWithLineNums struct {
	EntireFileReasoning string `json:"entireFileReasoning"`
	EntireFile          bool   `json:"entireFile"`
	Problems            string `json:"problems"`
	OriginalSections    []struct {
		Description      string           `json:"description"`
		Reasoning        string           `json:"reasoning"`
		SectionStartLine string           `json:"sectionStartLine"`
		SectionEndLine   string           `json:"sectionEndLine"`
		ShouldChange     bool             `json:"shouldChange"`
		ShouldRemove     bool             `json:"shouldRemove"`
		Structure        shared.Structure `json:"structure"`
	} `json:"originalSections"`
	Changes []*shared.StreamedChangeWithLineNumsUpdated `json:"changes"`
}

// type StreamedChangesFull struct {
// 	Changes []*shared.StreamedChangeFull `json:"changes"`
// }

type VerifyResult struct {
	SyntaxErrorsReasoning string `json:"syntaxErrorsReasoning"`
	HasSyntaxErrors       bool   `json:"hasSyntaxErrors"`
	Removed               []struct {
		Code      string `json:"code"`
		Reasoning string `json:"reasoning"`
		Correct   bool   `json:"correct"`
	} `json:"removed"`
	RemovedCodeErrorsReasoning string `json:"removedCodeErrorsReasoning"`
	HasRemovedCodeErrors       bool   `json:"hasRemovedCodeErrors"`
	DuplicationErrorsReasoning string `json:"duplicationErrorsReasoning"`
	HasDuplicationErrors       bool   `json:"hasDuplicationErrors"`
	Comments                   []struct {
		Txt       string `json:"txt"`
		Reference bool   `json:"reference"`
	} `json:"comments"`
	ReferenceErrorsReasoning string `json:"referenceErrorsReasoning"`
	HasReferenceErrors       bool   `json:"hasReferenceErrors"`
}

func (s *VerifyResult) IsCorrect() bool {
	return !s.HasRemovedCodeErrors && !s.HasDuplicationErrors && !s.HasReferenceErrors && !s.HasSyntaxErrors
}

func (s *VerifyResult) GetReasoning() string {
	res := []string{}

	if s.HasSyntaxErrors {
		res = append(res, s.SyntaxErrorsReasoning)
	}

	if s.HasRemovedCodeErrors {
		for _, removed := range s.Removed {
			if !removed.Correct {
				res = append(res, fmt.Sprintf("\nIncorrectly removed code:\n```\n%s```\n\nReason: %s", removed.Code, removed.Reasoning))
			}
		}

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
