package types

import "github.com/plandex/plandex/shared"

type LoadContextParams struct {
	Note      string
	MaxTokens uint32
	Recursive bool
	MaxDepth  int16
	NamesOnly bool
	Truncate  bool
	Resources []string
}

type OnStreamProposal func(content string, isFinished bool, err error)
type OnStreamPlan func(planChunk *shared.PlanChunk, isFinished bool, err error)
