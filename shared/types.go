package shared

import (
	openai "github.com/sashabaranov/go-openai"
)

type CurrentPlanFiles struct {
	Files map[string]string `json:"files"`
}

type ConversationMessage struct {
	Message openai.ChatCompletionMessage `json:"message"`
	Tokens  int                          `json:"tokens"`
}

type PromptRequest struct {
	Prompt           string                `json:"prompt"`
	ModelContext     ModelContext          `json:"modelContext"`
	CurrentPlan      CurrentPlanFiles      `json:"currentPlan"`
	ProjectPaths     map[string]bool       `json:"projectPaths"`
	Conversation     []ConversationMessage `json:"conversation"`
	ParentProposalId string                `json:"parentProposalId"`
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

type ModelContextPart struct {
	Name      string `json:"name"`
	Body      string `json:"body"`
	Url       string `json:"url"`
	FilePath  string `json:"filePath"`
	Sha       string `json:"sha"`
	NumTokens int    `json:"numTokens"`
	UpdatedAt string `json:"updatedAt"`
}
type ModelContext []ModelContextPart

type PlanChunk struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type PlanDescription struct {
	MadePlan  bool     `json:"madePlan"`
	CommitMsg string   `json:"commitMsg"`
	Files     []string `json:"files"`
}
