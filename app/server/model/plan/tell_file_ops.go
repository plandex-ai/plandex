package plan

import (
	"plandex-server/model/parse"
)

func (state *activeTellStreamState) checkMoveFileOps() []parse.FileMove {
	activePlan := GetActivePlan(state.plan.Id, state.branch)

	if activePlan == nil {
		return nil
	}

	moveFiles := parse.ParseMoveFiles(activePlan.CurrentReplyContent)

	return moveFiles
}

func (state *activeTellStreamState) checkRemoveFileOps() []string {
	activePlan := GetActivePlan(state.plan.Id, state.branch)

	if activePlan == nil {
		return nil
	}

	removeFiles := parse.ParseRemoveFiles(activePlan.CurrentReplyContent)

	return removeFiles
}

func (state *activeTellStreamState) checkResetFileOps() []string {
	activePlan := GetActivePlan(state.plan.Id, state.branch)

	if activePlan == nil {
		return nil
	}

	resetFiles := parse.ParseResetChanges(activePlan.CurrentReplyContent)

	return resetFiles
}
