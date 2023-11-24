package shared

import (
	openai "github.com/sashabaranov/go-openai"
)

type CurrentPlanFiles struct {
	Files       map[string]string `json:"files"`
	ContextShas map[string]string `json:"contextShas"`
}

type ConversationMessage struct {
	Message   openai.ChatCompletionMessage `json:"message"`
	Tokens    int                          `json:"tokens"`
	Timestamp string                       `json:"timestamp"`
}

type PromptRequest struct {
	Timestamp             string                 `json:"timestamp"`
	Prompt                string                 `json:"prompt"`
	ModelContext          ModelContext           `json:"modelContext"`
	CurrentPlan           *CurrentPlanFiles      `json:"currentPlan"`
	Conversation          []*ConversationMessage `json:"conversation"`
	ConversationSummaries []*ConversationSummary `json:"conversationSummaries"`
	ParentProposalId      string                 `json:"parentProposalId"`
	RootProposalId        string                 `json:"rootProposalId"`
}

type ShortSummaryRequest struct {
	Text string `json:"text"`
}

type ShortSummaryResponse struct {
	Summary string `json:"summary"`
}

type FileNameRequest struct {
	Text string `json:"text"`
}

type FileNameResponse struct {
	FileName string `json:"fileName"`
}

type BuildRequest struct {
	ProposalIds []string `json:"proposalIds"`
}

type PlanTokenCount struct {
	Path      string `json:"path"`
	NumTokens int    `json:"numTokens"`
	Finished  bool   `json:"finished"`
}

type PlanDescription struct {
	MadePlan              bool     `json:"madePlan"`
	CommitMsg             string   `json:"commitMsg"`
	Files                 []string `json:"files"`
	ResponseTimestamp     string   `json:"responseTimestamp"`
	SummarizedToTimestamp string   `json:"summarizedToTimestamp"`
}

type ConversationSummary struct {
	Summary              string
	Tokens               int
	LastMessageTimestamp string
	NumMessages          int
}
