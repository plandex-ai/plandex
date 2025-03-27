package shared

func (state *CurrentPlanState) ExecHistory() string {
	execHistory := ""

	if state.PlanResult == nil {
		return execHistory
	}

	for _, result := range state.PlanResult.Results {
		if result.Path == "_apply.sh" && result.AppliedAt != nil {
			execHistory += "Previously executed _apply.sh:\n\n```\n" + result.Content + "\n```\n\n"
		}
	}

	return execHistory
}
