package hooks

import (
	"context"
	"plandex-server/db"
	"plandex-server/types"
	"time"

	shared "plandex-shared"

	"github.com/jmoiron/sqlx"
	"github.com/sashabaranov/go-openai"
)

const (
	HealthCheck = "health_check"

	CreateAccount        = "create_account"
	WillCreatePlan       = "will_create_plan"
	WillTellPlan         = "will_tell_plan"
	WillExecPlan         = "will_exec_plan"
	WillSendModelRequest = "will_send_model_request"
	DidSendModelRequest  = "did_send_model_request"
	DidFinishBuilderRun  = "did_finish_builder_run"
	CreateOrg            = "create_org"
	Authenticate         = "authenticate"
	GetIntegratedModels  = "get_integrated_models"
	GetApiOrgs           = "get_api_orgs"
	CallFastApply        = "call_fast_apply"
)

type WillSendModelRequestParams struct {
	InputTokens  int
	OutputTokens int
	ModelName    shared.ModelName
	IsUserPrompt bool
}

type DidSendModelRequestParams struct {
	InputTokens     int
	OutputTokens    int
	CachedTokens    int
	ModelId         shared.ModelId
	ModelName       shared.ModelName
	ModelProvider   shared.ModelProvider
	ModelRole       shared.ModelRole
	ModelPackName   string
	Purpose         string
	GenerationId    string
	PlanId          string
	ModelStreamId   string
	ConvoMessageId  string
	BuildId         string
	StoppedEarly    bool
	UserCancelled   bool
	HadError        bool
	NoReportedUsage bool
	SessionId       string

	RequestStartedAt time.Time
	Streaming        bool
	StreamResult     string
	FirstTokenAt     time.Time
	Req              *types.ExtendedChatCompletionRequest
	Res              *openai.ChatCompletionResponse
	ModelConfig      *shared.ModelRoleConfig
}

type DidFinishBuilderRunParams struct {
	PlanId        string
	FilePath      string
	FileExt       string
	Lang          string
	GenerationIds []string

	ValidateModelConfig  *shared.ModelRoleConfig
	FastApplyModelConfig *shared.ModelRoleConfig
	WholeFileModelConfig *shared.ModelRoleConfig

	AutoApplySuccess                   bool
	AutoApplyValidationReasons         []string
	AutoApplyValidationSyntaxErrors    []string
	AutoApplyValidationPassed          bool
	AutoApplyValidationFailureResponse string
	AutoApplyValidationStartedAt       time.Time
	AutoApplyValidationFinishedAt      time.Time

	DidReplacement             bool
	ReplacementSuccess         bool
	ReplacementSyntaxErrors    []string
	ReplacementFailureResponse string
	ReplacementStartedAt       time.Time
	ReplacementFinishedAt      time.Time

	DidRewriteProposed             bool
	RewriteProposedSuccess         bool
	RewriteProposedSyntaxErrors    []string
	RewriteProposedFailureResponse string
	RewriteProposedStartedAt       time.Time
	RewriteProposedFinishedAt      time.Time

	DidFastApply             bool
	FastApplySuccess         bool
	FastApplySyntaxErrors    []string
	FastApplyFailureResponse string
	FastApplyStartedAt       time.Time
	FastApplyFinishedAt      time.Time

	BuiltWholeFile           bool
	BuildWholeFileStartedAt  time.Time
	BuildWholeFileFinishedAt time.Time

	StartedAt  time.Time
	FinishedAt time.Time
}

type CreateOrgHookRequestParams struct {
	Org *db.Org
}

type AuthenticateHookRequestParams struct {
	Path string
	Hash string
}

type FastApplyParams struct {
	InitialCode string `json:"initialCode"`
	EditSnippet string `json:"editSnippet"`

	InitialCodeTokens int
	EditSnippetTokens int

	Language shared.Language

	Ctx context.Context
}

type HookParams struct {
	Auth *types.ServerAuth
	Plan *db.Plan
	Tx   *sqlx.Tx

	WillSendModelRequestParams    *WillSendModelRequestParams
	DidSendModelRequestParams     *DidSendModelRequestParams
	CreateOrgHookRequestParams    *CreateOrgHookRequestParams
	GetApiOrgIds                  []string
	AuthenticateHookRequestParams *AuthenticateHookRequestParams
	DidFinishBuilderRunParams     *DidFinishBuilderRunParams
	FastApplyParams               *FastApplyParams
}

type GetIntegratedModelsResult struct {
	IntegratedModelsMode bool
	ApiKeys              map[string]string
}

type FastApplyResult struct {
	MergedCode string
}

type HookResult struct {
	GetIntegratedModelsResult *GetIntegratedModelsResult
	ApiOrgsById               map[string]*shared.Org
	FastApplyResult           *FastApplyResult
}

type Hook func(params HookParams) (HookResult, *shared.ApiError)

var hooks = make(map[string]Hook)

func RegisterHook(name string, hook Hook) {
	hooks[name] = hook
}

func ExecHook(name string, params HookParams) (HookResult, *shared.ApiError) {
	hook, ok := hooks[name]
	if !ok {
		return HookResult{}, nil
	}
	return hook(params)
}

func TestUpdate() {

}
