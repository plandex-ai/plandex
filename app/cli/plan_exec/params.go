package plan_exec

type ExecParams struct {
	CurrentPlanId        string
	CurrentBranch        string
	CheckOutdatedContext func()
}
