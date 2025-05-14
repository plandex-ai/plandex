package plan_exec

import (
	"plandex-cli/types"
	shared "plandex-shared"
)

type ExecParams struct {
	CurrentPlanId        string
	CurrentBranch        string
	AuthVars             map[string]string
	CheckOutdatedContext func(maybeContexts []*shared.Context, projectPaths *types.ProjectPaths) (bool, bool, error)
}
