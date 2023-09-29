package types

import (
	"context"

	"github.com/plandex/plandex/shared"
)

type Proposal struct {
	Id               string
	Cancel           *context.CancelFunc
	ModelContext     *shared.ModelContext
	Content          string
	FinishedProposal bool
	ProposalError    error
	Aborted          bool
	ConvertPlanError error
}

type OnStreamProposalFunc func(
	content string,
	finished bool,
	err error,
)

type OnStreamPlanFunc func(
	chunk *shared.PlanChunk,
	finished bool,
	err error,
)
