package hooks

import (
	"net/http"
	"plandex-server/db"

	"github.com/plandex/plandex/shared"
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
