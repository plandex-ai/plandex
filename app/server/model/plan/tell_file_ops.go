package plan

import (
	"log"
	"plandex-server/model/parse"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
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

func (state *activeTellStreamState) handleFileOps() {
	req := state.req
	clients := state.clients
	auth := state.auth
	currentOrgId := state.currentOrgId
	currentUserId := state.currentUserId
	plan := state.plan
	branch := state.branch
	settings := state.settings
	replyId := state.replyId

	moveFiles := state.checkMoveFileOps()
	removeFiles := state.checkRemoveFileOps()
	resetFiles := state.checkResetFileOps()

	if req.BuildMode == shared.BuildModeAuto {
		getBuildState := func() *activeBuildStreamState {
			return &activeBuildStreamState{
				tellState:     state,
				clients:       clients,
				auth:          auth,
				currentOrgId:  currentOrgId,
				currentUserId: currentUserId,
				plan:          plan,
				branch:        branch,
				settings:      settings,
				modelContext:  state.modelContext,
			}
		}

		if len(moveFiles) > 0 {
			log.Println("Detected move files, queuing builds")
			for i, moveFile := range moveFiles {
				getBuildState().queueBuilds([]*types.ActiveBuild{
					{
						ReplyId:         replyId,
						Idx:             i,
						Path:            moveFile.Source,
						IsMoveOp:        true,
						MoveDestination: moveFile.Destination,
					},
				})
			}
		}

		if len(removeFiles) > 0 {
			log.Println("Detected remove files, queuing builds")
			for i, removeFile := range removeFiles {
				getBuildState().queueBuilds([]*types.ActiveBuild{
					{
						ReplyId:    replyId,
						Idx:        i,
						Path:       removeFile,
						IsRemoveOp: true,
					},
				})
			}
		}

		if len(resetFiles) > 0 {
			log.Println("Detected reset files, queuing builds")

			for i, resetFile := range resetFiles {
				getBuildState().queueBuilds([]*types.ActiveBuild{
					{
						ReplyId:   replyId,
						Idx:       i,
						Path:      resetFile,
						IsResetOp: true,
					},
				})
			}
		}
	}
}
