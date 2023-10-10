package types

import (
	"github.com/plandex/plandex/shared"
)

type Proposal struct {
	Id      string
	Request *shared.PromptRequest
	Content string
	ProposalStage
	PlanDescription *shared.PlanDescription
}

type OnStreamFunc func(content string, err error)

func (s *Proposal) Finish(desc *shared.PlanDescription) bool {
	if desc == nil {
		return false
	}

	stopped := s.ProposalStage.Finish()
	if stopped {
		s.PlanDescription = desc
	}
	return stopped
}

func (s *Proposal) IsFinished() bool {
	return s.ProposalStage.Finished && s.PlanDescription != nil
}
