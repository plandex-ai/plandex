package lib

import "github.com/plandex/plandex/shared"

var buildPlanInlineFn func(autoConfirm bool, maybeContexts []*shared.Context) (bool, error)

func SetBuildPlanInlineFn(fn func(autoConfirm bool, maybeContexts []*shared.Context) (bool, error)) {
	buildPlanInlineFn = fn
}
