package plan

import (
	"log"
	"net/http"
	"plandex-server/db"
	"strings"

	"github.com/plandex/plandex/shared"
)

func (state *activeTellStreamState) verifyOrFinish() {
	plan := state.plan
	planId := plan.Id
	branch := state.branch
	currentOrgId := state.currentOrgId
	currentUserId := state.currentUserId

	active := GetActivePlan(planId, branch)

	if active == nil {
		return
	}

	if active.ShouldVerifyDiff() {

		repoLockId, err := db.LockRepo(
			db.LockRepoParams{
				OrgId:    currentOrgId,
				UserId:   currentUserId,
				PlanId:   planId,
				Branch:   branch,
				Scope:    db.LockScopeRead,
				Ctx:      active.Ctx,
				CancelFn: active.CancelFn,
			},
		)
		if err != nil {
			log.Printf("Error locking repo for verify diff: %v\n", err)
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Error locking repo for verify diff: " + err.Error(),
			}
			return
		}

		diffs, err := func() (string, error) {
			defer func() {
				err := db.DeleteRepoLock(repoLockId)
				if err != nil {
					log.Printf("Error unlocking repo after verify diff: %v\n", err)
				}
			}()

			diffs, err := db.GetPlanDiffs(currentOrgId, planId, true)
			if err != nil {
				log.Printf("Error getting plan diffs for verify diff: %v\n", err)
				return "", err
			}

			return diffs, nil
		}()

		if err != nil {
			log.Printf("Error getting plan diffs for verify diff: %v\n", err)
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Error getting plan diffs for verify diff: " + err.Error(),
			}
			return
		}

		if len(strings.TrimSpace(diffs)) == 0 {
			active.Finish()
		} else {
			state.verifyDiffs(diffs)
		}
	} else {
		active.Finish()
	}
}

func (state *activeTellStreamState) verifyDiffs(diffs string) {
	plan := state.plan
	planId := plan.Id
	branch := state.branch

	active := GetActivePlan(planId, branch)

	if active == nil {
		return
	}

	defer func() {
		active.DidVerifyDiff = true
	}()

	execTellPlan(
		state.clients,
		plan,
		branch,
		state.auth,
		state.req,
		state.iteration+1,
		"",
		false,
		"",
		diffs,
		0,
	)
}
