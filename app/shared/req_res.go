package shared

import (
	"time"

	"github.com/sashabaranov/go-openai"
)

type CreateEmailVerificationRequest struct {
	Email         string `json:"email"`
	UserId        string `json:"userId"`
	RequireUser   bool   `json:"requireUser"`
	RequireNoUser bool   `json:"requireNoUser"`
}

type CreateEmailVerificationResponse struct {
	HasAccount  bool `json:"hasAccount"`
	IsLocalMode bool `json:"isLocalMode"`
}

type VerifyEmailPinRequest struct {
	Email string `json:"email"`
	Pin   string `json:"pin"`
}

type SignInRequest struct {
	Email        string `json:"email"`
	Pin          string `json:"pin"`
	IsSignInCode bool   `json:"isSignInCode"`
}

type UiSignInToken struct {
	Pin        string `json:"pin"`
	RedirectTo string `json:"redirectTo"`
}

type CreateAccountRequest struct {
	Email    string `json:"email"`
	Pin      string `json:"pin"`
	UserName string `json:"userName"`
}

type SessionResponse struct {
	UserId      string `json:"userId"`
	Token       string `json:"token"`
	Email       string `json:"email"`
	UserName    string `json:"userName"`
	Orgs        []*Org `json:"orgs"`
	IsLocalMode bool   `json:"isLocalMode"`
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
	Prompt                 string            `json:"prompt"`
	BuildMode              BuildMode         `json:"buildMode"`
	ConnectStream          bool              `json:"connectStream"`
	AutoContinue           bool              `json:"autoContinue"`
	IsUserContinue         bool              `json:"isUserContinue"`
	IsUserDebug            bool              `json:"isUserDebug"`
	IsApplyDebug           bool              `json:"isApplyDebug"`
	IsChatOnly             bool              `json:"isChatOnly"`
	AutoContext            bool              `json:"autoContext"`
	SmartContext           bool              `json:"smartContext"`
	ExecEnabled            bool              `json:"execEnabled"`
	OsDetails              string            `json:"osDetails"`
	ApiKey                 string            `json:"apiKey"`   // deprecated
	Endpoint               string            `json:"endpoint"` // deprecated
	ApiKeys                map[string]string `json:"apiKeys"`
	OpenAIBase             string            `json:"openAIBase"`
	OpenAIOrgId            string            `json:"openAIOrgId"`
	ProjectPaths           map[string]bool   `json:"projectPaths"`
	IsImplementationOfChat bool              `json:"isImplementationOfChat"`
}

type BuildPlanRequest struct {
	ConnectStream bool              `json:"connectStream"`
	ApiKey        string            `json:"apiKey"`   // deprecated
	Endpoint      string            `json:"endpoint"` // deprecated
	ApiKeys       map[string]string `json:"apiKeys"`
	OpenAIBase    string            `json:"openAIBase"`
	OpenAIOrgId   string            `json:"openAIOrgId"`
	ProjectPaths  map[string]bool   `json:"projectPaths"`
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

type FileMapInputs map[string]string

type LoadContextParams struct {
	ContextType     ContextType           `json:"contextType"`
	Name            string                `json:"name"`
	Url             string                `json:"url"`
	FilePath        string                `json:"file_path"`
	Body            string                `json:"body"`
	ForceSkipIgnore bool                  `json:"forceSkipIgnore"`
	ImageDetail     openai.ImageURLDetail `json:"imageDetail"`
	AutoLoaded      bool                  `json:"autoLoaded"`
	MapInputs       FileMapInputs         `json:"mapInputs"`

	// For naming piped data
	ApiKeys     map[string]string `json:"apiKeys"`
	OpenAIBase  string            `json:"openAIBase"`
	OpenAIOrgId string            `json:"openAIOrgId"`
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
	Body            string            `json:"body"`
	InputShas       map[string]string `json:"inputShas"`
	MapBodies       FileMapBodies     `json:"mapBodies"`
	RemovedMapPaths []string          `json:"removedMapPaths"`
}

type GetFileMapRequest struct {
	MapInputs FileMapInputs `json:"mapInputs"`
}

type GetFileMapResponse struct {
	MapBodies FileMapBodies `json:"mapBodies"`
}

type LoadCachedFileMapRequest struct {
	FilePaths []string `json:"filePaths"`
}

type LoadCachedFileMapResponse struct {
	LoadRes      *LoadContextResponse `json:"loadRes"`
	CachedByPath map[string]bool      `json:"cachedByPath"`
}

type GetContextBodyRequest struct {
	ContextId string `json:"contextId"`
}

type GetContextBodyResponse struct {
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

type RejectFilesRequest struct {
	Paths []string `json:"paths"`
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

type UpdatePlanConfigRequest struct {
	Config *PlanConfig `json:"config"`
}

type UpdateDefaultPlanConfigRequest struct {
	Config *PlanConfig `json:"config"`
}

type GetPlanConfigResponse struct {
	Config *PlanConfig `json:"config"`
}

type GetDefaultPlanConfigResponse struct {
	Config *PlanConfig `json:"config"`
}

type ListUsersResponse struct {
	Users            []*User             `json:"users"`
	OrgUsersByUserId map[string]*OrgUser `json:"orgUsersByUserId"`
}

type ApplyPlanRequest struct {
	ApiKeys     map[string]string `json:"apiKeys"`
	OpenAIBase  string            `json:"openAIBase"`
	OpenAIOrgId string            `json:"openAIOrgId"`
}

type RenamePlanRequest struct {
	Name string `json:"name"`
}

// Cloud requests and responses
type CreditsLogRequest struct {
	PageSize int `json:"pageSize"`
	Page     int `json:"page"`
}

type CreditsLogResponse struct {
	Transactions []*CreditsTransaction `json:"transactions"`
	NumPages     int                   `json:"numPages"`
	NumPagesMax  bool                  `json:"numPagesMax"`
}
