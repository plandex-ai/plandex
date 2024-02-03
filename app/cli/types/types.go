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
	ClientAccount
	OrgId   string `json:"orgId"`
	OrgName string `json:"orgName"`
}

type LoadContextParams struct {
	Note            string
	Recursive       bool
	NamesOnly       bool
	ForceSkipIgnore bool
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

type CurrentPlanSettings struct {
	Id string `json:"id"`
}

type PlanSettings struct {
	Branch string `json:"branch"`
}

type CurrentProjectSettings struct {
	Id string `json:"id"`
}

type ChangesUIScrollReplacement struct {
	OldContent        string
	NewContent        string
	NumLinesPrepended int
}

type ChangesUIViewportsUpdate struct {
	ScrollReplacement *ChangesUIScrollReplacement
}
