package plan

import (
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
)

func (state *activeBuildStreamState) loadPendingBuilds() (map[string][]*types.ActiveBuild, error) {
	client := state.client
	plan := state.plan
	branch := state.branch
	auth := state.auth

	active, err := activatePlan(client, plan, branch, auth, "", true)

	if err != nil {
		log.Printf("Error activating plan: %v\n", err)
	}

	repoLockId, err := db.LockRepo(
		db.LockRepoParams{
			OrgId:    auth.OrgId,
			UserId:   auth.User.Id,
			PlanId:   plan.Id,
			Branch:   branch,
			Scope:    db.LockScopeRead,
			Ctx:      active.Ctx,
			CancelFn: active.CancelFn,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error locking repo for build: %v", err)
	}

	var modelContext []*db.Context
	var pendingBuildsByPath map[string][]*types.ActiveBuild
	var settings *shared.PlanSettings

	err = func() error {
		defer func() {
			err := db.UnlockRepo(repoLockId)
			if err != nil {
				log.Printf("Error unlocking repo: %v\n", err)
			}
		}()

		errCh := make(chan error)

		go func() {
			res, err := db.GetPlanContexts(auth.OrgId, plan.Id, true)
			if err != nil {
				log.Printf("Error getting plan modelContext: %v\n", err)
				errCh <- fmt.Errorf("error getting plan modelContext: %v", err)
				return
			}
			modelContext = res

			errCh <- nil
		}()

		go func() {
			res, err := active.PendingBuildsByPath(auth.OrgId, auth.User.Id, nil)

			if err != nil {
				log.Printf("Error getting pending builds by path: %v\n", err)
				errCh <- fmt.Errorf("error getting pending builds by path: %v", err)
				return
			}

			pendingBuildsByPath = res

			errCh <- nil
		}()

		go func() {
			res, err := db.GetPlanSettings(plan, true)
			if err != nil {
				log.Printf("Error getting plan settings: %v\n", err)
				errCh <- fmt.Errorf("error getting plan settings: %v", err)
				return
			}

			settings = res
			errCh <- nil
		}()

		for i := 0; i < 3; i++ {
			err = <-errCh
			if err != nil {
				log.Printf("Error getting plan data: %v\n", err)
				return err
			}
		}
		return nil
	}()

	if err != nil {
		return nil, err
	}

	UpdateActivePlan(plan.Id, branch, func(ap *types.ActivePlan) {
		ap.Contexts = modelContext
		for _, context := range modelContext {
			if context.FilePath != "" {
				ap.ContextsByPath[context.FilePath] = context
			}
		}
	})

	state.modelContext = modelContext
	state.settings = settings

	return pendingBuildsByPath, nil
}

func (state *activeBuildStreamFileState) loadBuildFile(activeBuild *types.ActiveBuild) error {

	currentOrgId := state.currentOrgId
	currentUserId := state.currentUserId
	planId := state.plan.Id
	branch := state.branch
	filePath := state.filePath

	activePlan := GetActivePlan(planId, branch)

	convoMessageId := activeBuild.ReplyId

	build := &db.PlanBuild{
		OrgId:          currentOrgId,
		PlanId:         planId,
		ConvoMessageId: convoMessageId,
		FilePath:       filePath,
	}
	err := db.StorePlanBuild(build)

	if err != nil {
		log.Printf("Error storing plan build: %v\n", err)
		UpdateActivePlan(activePlan.Id, activePlan.Branch, func(ap *types.ActivePlan) {
			ap.IsBuildingByPath[filePath] = false
		})
		activePlan.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error storing plan build: " + err.Error(),
		}
		return err
	}

	var currentPlan *shared.CurrentPlanState

	repoLockId, err := db.LockRepo(
		db.LockRepoParams{
			OrgId:       currentOrgId,
			UserId:      currentUserId,
			PlanId:      planId,
			Branch:      branch,
			PlanBuildId: build.Id,
			Scope:       db.LockScopeRead,
			Ctx:         activePlan.Ctx,
			CancelFn:    activePlan.CancelFn,
		},
	)
	if err != nil {
		log.Printf("Error locking repo for build file: %v\n", err)
		UpdateActivePlan(activePlan.Id, activePlan.Branch, func(ap *types.ActivePlan) {
			ap.IsBuildingByPath[filePath] = false
		})
		activePlan.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error locking repo for build file: " + err.Error(),
		}
		return err
	}

	err = func() error {
		defer func() {
			err := db.UnlockRepo(repoLockId)
			if err != nil {
				log.Printf("Error unlocking repo: %v\n", err)
			}
		}()

		res, err := db.GetCurrentPlanState(db.CurrentPlanStateParams{
			OrgId:  currentOrgId,
			PlanId: planId,
		})
		if err != nil {
			log.Printf("Error getting current plan state: %v\n", err)
			UpdateActivePlan(activePlan.Id, activePlan.Branch, func(ap *types.ActivePlan) {
				ap.IsBuildingByPath[filePath] = false
			})
			activePlan.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Error getting current plan state: " + err.Error(),
			}
			return err
		}
		currentPlan = res

		log.Println("Got current plan state")
		return nil
	}()

	if err != nil {
		return err
	}

	state.filePath = filePath
	state.convoMessageId = convoMessageId
	state.build = build
	state.currentPlanState = currentPlan

	return nil

}
