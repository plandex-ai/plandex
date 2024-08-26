package hooks

import (
	"plandex-server/db"

	"github.com/plandex/plandex/shared"
)

const (
	CreateAccount        = "create_account"
	WillCreatePlan       = "will_create_plan"
	WillTellPlan         = "will_tell_plan"
	WillExecPlan         = "will_exec_plan"
	WillSendModelRequest = "will_send_model_request"
	DidSendModelRequest  = "did_send_model_request"
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

type HookParams struct {
	User  *db.User
	OrgId string
	Plan  *db.Plan

	WillSendModelRequestParams *WillSendModelRequestParams
	DidSendModelRequestParams  *DidSendModelRequestParams
}

type Hook func(params HookParams) *shared.ApiError

var hooks = make(map[string]Hook)

func RegisterHook(name string, hook Hook) {
	hooks[name] = hook
}

func ExecHook(name string, params HookParams) *shared.ApiError {
	hook, ok := hooks[name]
	if !ok {
		return nil
	}
	return hook(params)
}
