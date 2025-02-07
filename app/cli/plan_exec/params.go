package plan_exec

import shared "plandex-shared"

type ExecParams struct {
	CurrentPlanId        string
	CurrentBranch        string
	ApiKeys              map[string]string
	CheckOutdatedContext func(maybeContexts []*shared.Context) (bool, bool, error)
}

type TellFlags struct {
	TellBg               bool
	TellStop             bool
	TellNoBuild          bool
	IsUserContinue       bool
	IsUserDebug          bool
	IsApplyDebug         bool
	IsChatOnly           bool
	AutoContext          bool
	ContinuedAfterAction bool
	ExecEnabled          bool
}
