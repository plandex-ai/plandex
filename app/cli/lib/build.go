package lib

import "github.com/plandex/plandex/shared"

var buildPlanInlineFn func(maybeContexts []*shared.Context) (bool, error)

func SetBuildPlanInlineFn(fn func(maybeContexts []*shared.Context) (bool, error)) {
	buildPlanInlineFn = fn
}
