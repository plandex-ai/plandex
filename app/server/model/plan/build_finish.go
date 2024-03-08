package plan

import (
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
)

func (state *activeBuildStreamFileState) onFinishBuild() {
	log.Println("Build finished")

	planId := state.plan.Id
	branch := state.branch
	currentOrgId := state.currentOrgId
	currentUserId := state.currentUserId
	convoMessageId := state.convoMessageId
	build := state.build

	activePlan := GetActivePlan(planId, branch)

	// first check if any of the messages we're building hasen't finished streaming yet
	stillStreaming := false
	var doneCh chan bool
	ap := GetActivePlan(planId, branch)
	if ap.CurrentStreamingReplyId == convoMessageId {
		stillStreaming = true
		doneCh = ap.CurrentReplyDoneCh
	}
	if stillStreaming {
		log.Println("Reply is still streaming, waiting for it to finish before finishing build")
		<-doneCh
	}

	// Check again if build is finished
	// (more builds could have been queued while we were waiting for the reply to finish streaming)
	ap = GetActivePlan(planId, branch)
	if !ap.BuildFinished() {
		log.Println("Build not finished after waiting for reply to finish streaming")
		return
	}

	log.Println("Locking repo for finished build")

	repoLockId, err := db.LockRepo(
		db.LockRepoParams{
			OrgId:       currentOrgId,
			UserId:      currentUserId,
			PlanId:      planId,
			Branch:      branch,
			PlanBuildId: build.Id,
			Scope:       db.LockScopeWrite,
			Ctx:         activePlan.Ctx,
			CancelFn:    activePlan.CancelFn,
		},
	)

	if err != nil {
		log.Printf("Error locking repo for finished build: %v\n", err)
		activePlan.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error locking repo for finished build: " + err.Error(),
		}
		return
	}

	log.Println("Locked repo for finished build")

	err = func() error {
		var err error
		defer func() {
			if err != nil {
				log.Printf("Finish build error: %v\n", err)
				err = db.GitClearUncommittedChanges(currentOrgId, planId)
				if err != nil {
					log.Printf("Error clearing uncommitted changes: %v\n", err)
				}
				log.Println("Cleared uncommitted changes")
			}

			err := db.UnlockRepo(repoLockId)
			if err != nil {
				log.Printf("Error unlocking repo: %v\n", err)
			}

			log.Println("Unlocked repo")
		}()

		// get plan descriptions
		var planDescs []*db.ConvoMessageDescription
		planDescs, err = db.GetConvoMessageDescriptions(currentOrgId, planId)
		if err != nil {
			log.Printf("Error getting pending build descriptions: %v\n", err)
			return fmt.Errorf("error getting pending build descriptions: %v", err)
		}

		var unbuiltDescs []*db.ConvoMessageDescription
		for _, desc := range planDescs {
			if !desc.DidBuild || len(desc.BuildPathsInvalidated) > 0 {
				unbuiltDescs = append(unbuiltDescs, desc)
			}
		}

		// get fresh current plan state
		var currentPlan *shared.CurrentPlanState
		currentPlan, err = db.GetCurrentPlanState(db.CurrentPlanStateParams{
			OrgId:                    currentOrgId,
			PlanId:                   planId,
			ConvoMessageDescriptions: planDescs,
		})
		if err != nil {
			log.Printf("Error getting current plan state: %v\n", err)
			return fmt.Errorf("error getting current plan state: %v", err)
		}

		descErrCh := make(chan error)
		for _, desc := range unbuiltDescs {
			if len(desc.Files) > 0 {
				desc.DidBuild = true
				desc.BuildPathsInvalidated = map[string]bool{}
			}

			go func(desc *db.ConvoMessageDescription) {
				err := db.StoreDescription(desc)

				if err != nil {
					descErrCh <- fmt.Errorf("error storing description: %v", err)
					return
				}

				descErrCh <- nil
			}(desc)
		}

		for range unbuiltDescs {
			err = <-descErrCh
			if err != nil {
				log.Printf("Error storing description: %v\n", err)
				return err
			}
		}

		err = db.GitAddAndCommit(currentOrgId, planId, branch, currentPlan.PendingChangesSummaryForBuild())

		if err != nil {
			log.Printf("Error committing plan build: %v\n", err)
			activePlan.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Error committing plan build: " + err.Error(),
			}
			return err
		}

		log.Println("Plan build committed")

		return nil

	}()

	if err != nil {
		return
	}

	active := GetActivePlan(planId, branch)

	if active != nil && (active.RepliesFinished || active.BuildOnly) {
		activePlan.Stream(shared.StreamMessage{
			Type: shared.StreamMessageFinished,
		})
	}
}

func (fileState *activeBuildStreamFileState) onFinishBuildFile(planRes *db.PlanFileResult) {
	planId := fileState.plan.Id
	branch := fileState.branch
	currentOrgId := fileState.currentOrgId
	currentUserId := fileState.currentUserId
	build := fileState.build
	activeBuild := fileState.activeBuild

	activePlan := GetActivePlan(planId, branch)
	filePath := fileState.filePath

	finished := false
	log.Println("onFinishBuildFile: " + filePath)

	repoLockId, err := db.LockRepo(
		db.LockRepoParams{
			OrgId:       currentOrgId,
			UserId:      currentUserId,
			PlanId:      planId,
			Branch:      branch,
			PlanBuildId: build.Id,
			Scope:       db.LockScopeWrite,
			Ctx:         activePlan.Ctx,
			CancelFn:    activePlan.CancelFn,
		},
	)
	if err != nil {
		log.Printf("Error locking repo for build file: %v\n", err)
		activePlan.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error locking repo for build file: " + err.Error(),
		}
		return
	}

	err = func() error {
		var err error
		defer func() {
			if err != nil {
				log.Printf("Error: %v\n", err)
				err = db.GitClearUncommittedChanges(currentOrgId, planId)
				if err != nil {
					log.Printf("Error clearing uncommitted changes: %v\n", err)
				}
			}

			err := db.UnlockRepo(repoLockId)
			if err != nil {
				log.Printf("Error unlocking repo: %v\n", err)
			}
		}()

		err = db.StorePlanResult(planRes)
		if err != nil {
			log.Printf("Error storing plan result: %v\n", err)
			activePlan.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Error storing plan result: " + err.Error(),
			}
			return err
		}
		return nil
	}()

	if err != nil {
		return
	}

	activeBuild.Success = true

	UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
		ap.BuiltFiles[filePath] = true
		if ap.BuildFinished() {
			finished = true
		}
	})

	log.Printf("Finished building file %s\n", filePath)

	if finished {
		UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
			ap.IsBuildingByPath[filePath] = false
		})
		log.Println("Finished building plan, calling onFinishBuild")
		fileState.onFinishBuild()
	} else {
		if activePlan.PathFinished(filePath) {
			UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
				ap.IsBuildingByPath[filePath] = false
			})
			log.Printf("File %s finished, but not all builds finished\n", filePath)
		} else {
			log.Printf("Processing next build for file %s\n", filePath)
			queue := activePlan.BuildQueuesByPath[filePath]
			var nextBuild *types.ActiveBuild
			for _, build := range queue {
				if !build.BuildFinished() {
					nextBuild = build
					break
				}
			}

			if nextBuild != nil {
				log.Println("Calling execPlanBuild for next build in queue")
				go fileState.execPlanBuild(nextBuild)
			}
			return
		}
	}

}

func (fileState *activeBuildStreamFileState) onBuildFileError(err error) {
	planId := fileState.plan.Id
	branch := fileState.branch
	filePath := fileState.filePath
	build := fileState.build
	activeBuild := fileState.activeBuild

	activePlan := GetActivePlan(planId, branch)

	log.Printf("Error for file %s: %v\n", filePath, err)

	activeBuild.Success = false
	activeBuild.Error = err

	activePlan.StreamDoneCh <- &shared.ApiError{
		Type:   shared.ApiErrorTypeOther,
		Status: http.StatusInternalServerError,
		Msg:    err.Error(),
	}

	if err != nil {
		log.Printf("Error storing plan error result: %v\n", err)
	}

	build.Error = err.Error()

	err = db.SetBuildError(build)
	if err != nil {
		log.Printf("Error setting build error: %v\n", err)
	}
}
