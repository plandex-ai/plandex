package hooks

import (
	"net/http"
	"plandex-server/db"

	"github.com/plandex/plandex/shared"
)

const (
	CreateAccount  = "create_account"
	WillCreatePlan = "will_create_plan"
	WillTellPlan   = "will_tell_plan"
	WillExecPlan   = "will_exec_plan"
)

type HookParams struct {
	W            http.ResponseWriter
	User         *db.User
	OrgId        string
	Plan         *db.Plan
	StreamDoneCh chan *shared.ApiError
}

type Hook func(params HookParams) error

var hooks = make(map[string]Hook)

func RegisterHook(name string, hook Hook) {
	hooks[name] = hook
}

func ExecHook(name string, params HookParams) error {
	hook, ok := hooks[name]
	if !ok {
		return nil
	}
	return hook(params)
}
