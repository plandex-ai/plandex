package shared

import (
	openai "github.com/sashabaranov/go-openai"
)

type CurrentPlanFiles struct {
	Files map[string]string `json:"files"`
	// Exec  string            `json:"exec"`
}

type PromptRequest struct {
	Prompt       string       `json:"prompt"`
	ModelContext ModelContext `json:"modelContext"`
	Conversation []openai.ChatCompletionMessage
	CurrentPlan  CurrentPlanFiles `json:"currentPlan"`
}

type SummarizeRequest struct {
	Text string `json:"text"`
}

type SummarizeResponse struct {
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	FileName string `json:"fileName"`
}

type ModelContextPart struct {
	Name      string `json:"name"`
	Summary   string `json:"summary"`
	Body      string `json:"body"`
	Url       string `json:"url"`
	FilePath  string `json:"filePath"`
	Sha       string `json:"sha"`
	NumTokens uint32 `json:"numTokens"`
	UpdatedAt string `json:"updatedAt"`
}
type ModelContext []ModelContextPart

type ModelContextState struct {
	NumTokens    uint32 `json:"numTokens"`
	Counter      uint32 `json:"counter"`
	ActiveTokens uint32 `json:"activeTokens"`
	ChatFlexPct  uint8  `json:"chatFlexPct"`
	PlanFlexPct  uint8  `json:"planFlexPct"`
}

type PlanSettings struct {
	Name string `json:"name"`
}

type PlanChunk struct {
	Type     string `json:"type"`
	FilePath string `json:"filePath"`
	Content  string `json:"content"`
	// IsExec   bool   `json:"isExec"`
}

type PlanDescription struct {
	MadePlan  bool     `json:"madePlan"`
	CommitMsg string   `json:"commitMsg"`
	Files     []string `json:"files"`
	// HasExec   bool     `json:"hasExec"`
}
