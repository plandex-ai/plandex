package hooks

import (
	"net/http"
	"plandex-server/db"

	"github.com/jmoiron/sqlx"
	"github.com/plandex/plandex/shared"
)

const (
	CreateAccount  = "create_account"
	WillCreatePlan = "will_create_plan"
	WillTellPlan   = "will_tell_plan"
	WillExecPlan   = "will_exec_plan"

	CreateOrg = "create_org"
)

type HookParams struct {
	W            http.ResponseWriter
	User         *db.User
	OrgId        string
	Org          *db.Org
	Plan         *db.Plan
	StreamDoneCh chan *shared.ApiError
}

type Hook func(tx *sqlx.Tx, params HookParams) error

var hooks = make(map[string]Hook)

func RegisterHook(name string, hook Hook) {
	hooks[name] = hook
}

func ExecHook(tx *sqlx.Tx, name string, params HookParams) error {
	hook, ok := hooks[name]
	if !ok {
		return nil
	}
	return hook(tx, params)
}
