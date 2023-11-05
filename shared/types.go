package shared

import (
	openai "github.com/sashabaranov/go-openai"
)

type CurrentPlanFiles struct {
	Files map[string]string `json:"files"`
}

type ConversationMessage struct {
	Message   openai.ChatCompletionMessage `json:"message"`
	Tokens    int                          `json:"tokens"`
	Timestamp string                       `json:"timestamp"`
}

type PromptRequest struct {
	Timestamp             string                `json:"timestamp"`
	Prompt                string                `json:"prompt"`
	ModelContext          ModelContext          `json:"modelContext"`
	CurrentPlan           CurrentPlanFiles      `json:"currentPlan"`
	Conversation          []ConversationMessage `json:"conversation"`
	ConversationSummaries []ConversationSummary `json:"conversationSummaries"`
	ParentProposalId      string                `json:"parentProposalId"`
	RootProposalId        string                `json:"rootProposalId"`
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

type ModelContextPartType string

const (
	ContextFileType          ModelContextPartType = "file"
	ContextURLType           ModelContextPartType = "url"
	ContextNoteType          ModelContextPartType = "note"
	ContextDirectoryTreeType ModelContextPartType = "directory tree"
	ContextPipedDataType     ModelContextPartType = "piped data"
)

type ModelContextPart struct {
	Type      ModelContextPartType `json:"type"`
	Name      string               `json:"name"`
	Body      string               `json:"body"`
	Url       string               `json:"url"`
	FilePath  string               `json:"filePath"`
	Sha       string               `json:"sha"`
	NumTokens int                  `json:"numTokens"`
	AddedAt   string               `json:"addedAt"`
	UpdatedAt string               `json:"updatedAt"`
}
type ModelContext []ModelContextPart

type PlanChunk struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	NumTokens int    `json:"numTokens"`
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
}
