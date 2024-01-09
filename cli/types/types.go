package types

import "github.com/plandex/plandex/shared"

type ClientAccount struct {
	IsCloud  bool   `json:"isCloud"`
	Host     string `json:"host"`
	Email    string `json:"email"`
	UserName string `json:"userName"`
	UserId   string `json:"userId"`
	Token    string `json:"token"`
	IsTrial  bool   `json:"isTrial"`
}

type ClientAuth struct {
	IsCloud  bool   `json:"isCloud"`
	Host     string `json:"host"`
	Email    string `json:"email"`
	OrgId    string `json:"orgId"`
	OrgName  string `json:"orgName"`
	UserId   string `json:"userId"`
	UserName string `json:"userName"`
	Token    string `json:"token"`
	IsTrial  bool   `json:"isTrial"`
}

type LoadContextParams struct {
	Note      string
	Recursive bool
	NamesOnly bool
}

type ContextOutdatedResult struct {
	Msg             string
	UpdatedContexts []*shared.Context
	TokenDiffsById  map[string]int
	NumFiles        int
	NumUrls         int
	NumTrees        int
}

const (
	PlanOutdatedStrategyOverwrite        string = "Clear the modifications and then apply"
	PlanOutdatedStrategyApplyUnmodified  string = "Apply only new and unmodified files"
	PlanOutdatedStrategyApplyNoConflicts string = "Apply anyway since there are no conflicts"
	PlanOutdatedStrategyRebuild          string = "Rebuild the plan with updated context"
	PlanOutdatedStrategyCancel           string = "Cancel"
)

type PlanSettings struct {
	Id string `json:"id"`
}

type ProjectSettings struct {
	Id string `json:"id"`
}

type StreamTUIUpdate struct {
	ReplyChunk     string
	Processing     bool
	PlanTokenCount *shared.PlanTokenCount
}
