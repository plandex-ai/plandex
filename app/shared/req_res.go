package shared

import "time"

type StartTrialResponse struct {
	UserId   string `json:"userId"`
	Token    string `json:"token"`
	OrgId    string `json:"orgId"`
	Email    string `json:"email"`
	UserName string `json:"userName"`
	OrgName  string `json:"orgName"`
}

type CreateEmailVerificationRequest struct {
	Email  string `json:"email"`
	UserId string `json:"userId"`
}

type CreateEmailVerificationResponse struct {
	HasAccount bool `json:"hasAccount"`
}

type SignInRequest struct {
	Email string `json:"email"`
	Pin   string `json:"pin"`
}

type CreateAccountRequest struct {
	Email    string `json:"email"`
	Pin      string `json:"pin"`
	UserName string `json:"userName"`
}

type SessionResponse struct {
	UserId   string `json:"userId"`
	Token    string `json:"token"`
	Email    string `json:"email"`
	UserName string `json:"userName"`
	Orgs     []*Org `json:"orgs"`
}

type CreateOrgRequest struct {
	Name               string `json:"name"`
	AutoAddDomainUsers bool   `json:"autoAddDomainUsers"`
}

type ConvertTrialRequest struct {
	Email                 string `json:"email"`
	Pin                   string `json:"pin"`
	UserName              string `json:"userName"`
	OrgName               string `json:"orgName"`
	OrgAutoAddDomainUsers bool   `json:"orgAutoAddDomainUsers"`
}

type CreateOrgResponse struct {
	Id string `json:"id"`
}

type InviteRequest struct {
	Email     string `json:"email"`
	Name      string `json:"name"`
	OrgRoleId string `json:"orgRoleId"`
}

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

type GetCurrentBranchByPlanIdRequest struct {
	CurrentBranchByPlanId map[string]string `json:"currentBranchByPlanId"`
}

type ListPlansRunningResponse struct {
	Branches                   []*Branch            `json:"branches"`
	StreamStartedAtByBranchId  map[string]time.Time `json:"streamStartedAtByBranchId"`
	StreamFinishedAtByBranchId map[string]time.Time `json:"streamFinishedAtByBranchId"`
	StreamIdByBranchId         map[string]string    `json:"streamIdByBranchId"`
	PlansById                  map[string]*Plan     `json:"plansById"`
}

type BuildMode string

const (
	BuildModeAuto BuildMode = "auto"
	BuildModeNone BuildMode = "none"
)

type TellPlanRequest struct {
	Prompt         string          `json:"prompt"`
	BuildMode      BuildMode       `json:"buildMode"`
	ConnectStream  bool            `json:"connectStream"`
	AutoContinue   bool            `json:"autoContinue"`
	IsUserContinue bool            `json:"isUserContinue"`
	ApiKey         string          `json:"apiKey"`
	ProjectPaths   map[string]bool `json:"projectPaths"`
}

type BuildPlanRequest struct {
	ConnectStream bool            `json:"connectStream"`
	ApiKey        string          `json:"apiKey"`
	ProjectPaths  map[string]bool `json:"projectPaths"`
}

const NoBuildsErr string = "No builds"

type RespondMissingFileChoice string

const (
	RespondMissingFileChoiceLoad      RespondMissingFileChoice = "load"
	RespondMissingFileChoiceSkip      RespondMissingFileChoice = "skip"
	RespondMissingFileChoiceOverwrite RespondMissingFileChoice = "overwrite"
)

type RespondMissingFileRequest struct {
	Choice   RespondMissingFileChoice `json:"choice"`
	FilePath string                   `json:"filePath"`
	Body     string                   `json:"body"`
}

type LoadContextParams struct {
	ContextType     ContextType `json:"contextType"`
	Name            string      `json:"name"`
	Url             string      `json:"url"`
	FilePath        string      `json:"file_path"`
	Body            string      `json:"body"`
	ForceSkipIgnore bool        `json:"forceSkipIgnore"`
}

type LoadContextRequest []*LoadContextParams

type LoadContextResponse struct {
	TokensAdded       int    `json:"tokensAdded"`
	TotalTokens       int    `json:"totalTokens"`
	MaxTokensExceeded bool   `json:"maxTokensExceeded"`
	MaxTokens         int    `json:"maxTokens"`
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

type RejectFileRequest struct {
	FilePath string `json:"filePath"`
}

type RewindPlanRequest struct {
	Sha string `json:"sha"`
}

type RewindPlanResponse struct {
	LatestSha    string `json:"latestSha"`
	LatestCommit string `json:"latestCommit"`
}

type LogResponse struct {
	Shas []string `json:"shas"`
	Body string   `json:"body"`
}

type CreateBranchRequest struct {
	Name string `json:"name"`
}

type UpdateSettingsRequest struct {
	Settings *PlanSettings `json:"settings"`
}

type UpdateSettingsResponse struct {
	Msg string `json:"msg"`
}

type ListUsersResponse struct {
	Users            []*User             `json:"users"`
	OrgUsersByUserId map[string]*OrgUser `json:"orgUsersByUserId"`
}
