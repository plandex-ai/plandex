package types

import (
	shared "plandex-shared"
	"time"

	"github.com/sashabaranov/go-openai"
)

type LoadContextParams struct {
	Note              string
	Recursive         bool
	NamesOnly         bool
	ForceSkipIgnore   bool
	ImageDetail       openai.ImageURLDetail
	DefsOnly          bool
	SkipIgnoreWarning bool
	AutoLoaded        bool
	SessionId         string
}

type ContextOutdatedResult struct {
	Msg             string
	UpdatedContexts []*shared.Context
	RemovedContexts []*shared.Context
	TokenDiffsById  map[string]int
	NumFiles        int
	NumUrls         int
	NumTrees        int
	NumMaps         int
	NumFilesRemoved int
	NumTreesRemoved int
	ReqFn           func() (map[string]*shared.UpdateContextParams, error)
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

type CurrentPlanSettingsByAccount map[string]*CurrentPlanSettings
type PlanSettingsByAccount map[string]*PlanSettings
type CurrentProjectSettingsByAccount map[string]*CurrentProjectSettings

type ChangesUIScrollReplacement struct {
	OldContent        string
	NewContent        string
	NumLinesPrepended int
}

type ChangesUIViewportsUpdate struct {
	ScrollReplacement *ChangesUIScrollReplacement
}

type OnErrFn func(errMsg string, errArgs ...interface{})

type OauthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type OauthCreds struct {
	OauthResponse
	ExpiresAt time.Time `json:"expires_at"`
}

type AccountCredentials struct {
	ClaudeMax *OauthCreds `json:"claudeMax"`
}
