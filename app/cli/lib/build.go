package lib

var buildPlanInlineFn func() (bool, error)

func SetBuildPlanInlineFn(fn func() (bool, error)) {
	buildPlanInlineFn = fn
}
