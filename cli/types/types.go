package types

import (
	"github.com/looplab/fsm"
	"github.com/plandex/plandex/shared"
)

type LoadContextParams struct {
	Note      string
	MaxTokens int
	Recursive bool
	MaxDepth  int16
	NamesOnly bool
	Truncate  bool
	Resources []string
}

type OnStreamPlanParams struct {
	Content string
	State   *fsm.FSM
	Err     error
}

type OnStreamPlan func(params OnStreamPlanParams)

// APIHandler is an interface that represents the public API functions
type APIHandler interface {
	Propose(prompt string, onStream OnStreamPlan) (*shared.PromptRequest, error)
	Summarize(text string) (*shared.SummarizeResponse, error)
	Abort(proposalId string) error
}
