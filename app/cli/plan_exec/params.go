package plan_exec

import "github.com/plandex/plandex/shared"

type ExecParams struct {
	CurrentPlanId        string
	CurrentBranch        string
	CheckOutdatedContext func(maybeContexts []*shared.Context) (bool, bool)
}
