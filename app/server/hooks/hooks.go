package hooks

import (
	"plandex-server/db"
	"plandex-server/types"

	"github.com/jmoiron/sqlx"
	"github.com/plandex/plandex/shared"
)

const (
	HealthCheck = "health_check"

	CreateAccount        = "create_account"
	WillCreatePlan       = "will_create_plan"
	WillTellPlan         = "will_tell_plan"
	WillExecPlan         = "will_exec_plan"
	WillSendModelRequest = "will_send_model_request"
	DidSendModelRequest  = "did_send_model_request"

	CreateOrg           = "create_org"
	Authenticate        = "authenticate"
	GetIntegratedModels = "get_integrated_models"
	GetApiOrgs          = "get_api_orgs"
)

type WillSendModelRequestParams struct {
	InputTokens  int
	OutputTokens int
	ModelName    string
}

type DidSendModelRequestParams struct {
	InputTokens   int
	OutputTokens  int
	ModelName     string
	ModelProvider shared.ModelProvider
	ModelRole     shared.ModelRole
	ModelPackName string
	Purpose       string
}

type CreateOrgHookRequestParams struct {
	Org *db.Org
}

type AuthenticateHookRequestParams struct {
	Path string
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
}

type GetIntegratedModelsResult struct {
	IntegratedModelsMode bool
	ApiKeys              map[string]string
}

type HookResult struct {
	GetIntegratedModelsResult *GetIntegratedModelsResult
	ApiOrgsById               map[string]*shared.Org
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
