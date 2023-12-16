package shared

type CreateProjectRequest struct {
	Name string `json:"name"`
}

type CreateProjectResponse struct {
	Id string `json:"id"`
}

type SetProjectPlanRequest struct {
	PlanId string `json:"planId"`
}

type RenameProjectRequest struct {
	Name string `json:"name"`
}

type CreatePlanRequest struct {
	Name string `json:"name"`
}

type CreatePlanResponse struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type TellPlanRequest struct {
	Prompt        string `json:"prompt"`
	ConnectStream bool   `json:"connectStream"`
}

type LoadContextParams struct {
	ContextType ContextType `json:"contextType"`
	Name        string      `json:"name"`
	Url         string      `json:"url"`
	FilePath    string      `json:"file_path"`
	Body        string      `json:"body"`
}

type LoadContextRequest []*LoadContextParams

type LoadContextResponse struct {
	TokensAdded       int    `json:"tokensAdded"`
	TotalTokens       int    `json:"totalTokens"`
	MaxTokensExceeded bool   `json:"maxTokensExceeded"`
	Msg               string `json:"msg"`
}

type UpdateContextParams struct {
	Body string `json:"body"`
}

type UpdateContextRequest map[string]*UpdateContextParams

type UpdateContextResponse = LoadContextResponse

type DeleteContextRequest struct {
	Ids map[string]bool `json:"ids"`
}

type DeleteContextResponse struct {
	TokensRemoved int    `json:"tokensRemoved"`
	TotalTokens   int    `json:"totalTokens"`
	Msg           string `json:"msg"`
}

type RewindPlanRequest struct {
	Sha string `json:"sha"`
}

type LogResponse struct {
	Body string `json:"body"`
}

type PlanTokenCount struct {
	Path      string `json:"path"`
	NumTokens int    `json:"numTokens"`
	Finished  bool   `json:"finished"`
}
