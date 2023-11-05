package types

import (
	"github.com/looplab/fsm"
	"github.com/plandex/plandex/shared"
)

type OnStreamPlanParams struct {
	Content string
	State   *fsm.FSM
	Err     error
}

type OnStreamPlan func(params OnStreamPlanParams)

// APIHandler is an interface that represents the public API functions
type APIHandler interface {
	Propose(prompt, parentProposalId, rootId string, onStream OnStreamPlan) (*shared.PromptRequest, error)
	ShortSummary(text string) (*shared.ShortSummaryResponse, error)
	Abort(proposalId string) error
	FileName(text string) (*shared.FileNameResponse, error)
	ConvoSummary(rootId, latestTimestamp string) (*shared.ConversationSummary, error)
}

type AppendConversationParams struct {
	Timestamp         string
	ResponseTimestamp string
	Prompt            string
	Reply             string
	PromptTokens      int
	ReplyTokens       int
	PlanState         *PlanState
}

type PlanState struct {
	Name                  string                  `json:"name"`
	ProposalId            string                  `json:"proposalId"`
	RootId                string                  `json:"rootId"`
	CreatedAt             string                  `json:"createdAt"`
	UpdatedAt             string                  `json:"updatedAt"`
	Description           *shared.PlanDescription `json:"description"`
	ContextTokens         int                     `json:"contextTokens"`
	ConvoTokens           int                     `json:"convoTokens"`
	ConvoSummarizedTokens int                     `json:"convoSummarizedTokens"`
}

type PlanSettings struct {
	Name string
}

type LoadContextParams struct {
	Note      string
	Recursive bool
	NamesOnly bool
	// Truncate  bool
	// MaxDepth  int
	// MaxTokens int
}
