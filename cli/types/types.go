package types

import (
	"github.com/looplab/fsm"
	"github.com/plandex/plandex/shared"
)

type LoadContextParams struct {
	Note      string
	MaxTokens int
	Recursive bool
	MaxDepth  int
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
	Propose(prompt, parentProposalId string, onStream OnStreamPlan) (*shared.PromptRequest, error)
	ShortSummary(text string) (*shared.ShortSummaryResponse, error)
	Abort(proposalId string) error
	FileName(text string) (*shared.FileNameResponse, error)
}

type AppendConversationParams struct {
	Timestamp    string
	Prompt       string
	Reply        string
	PromptTokens int
	ReplyTokens  int
}

type ConversationSummaryParams struct {
	CurrentTimestamp string
	MessageTimestamp string
	Summary          string
	SummaryTokens    int
}

type PlanState struct {
	ProposalId string
}

type PlanSettings struct {
	Name string
}
