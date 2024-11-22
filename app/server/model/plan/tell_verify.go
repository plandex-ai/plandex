package plan

import (
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/types"
	"strings"

	"github.com/plandex/plandex/shared"
)

func (state *activeTellStreamState) verifyOrFinish() {
	log.Println("Verifying or finishing")

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
		log.Println("Should verify diff")
		UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
			ap.IsVerifyingDiff = true
		})

		log.Println("Locking repo and getting diffs")
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
				log.Println("Unlocking repo after verify diff")
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
			UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
				ap.IsVerifyingDiff = false
			})
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Error getting plan diffs for verify diff: " + err.Error(),
			}
			return
		}

		if len(strings.TrimSpace(diffs)) == 0 {
			log.Println("No diffs, finishing")
			UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
				ap.IsVerifyingDiff = false
			})
			active.Finish()
		} else {
			log.Println("Got diffs, verifying")
			state.verifyDiffs(diffs)
		}
	} else {
		log.Println("Not verifying, finishing unless another verification is already in progress")
		if !active.IsVerifyingDiff {
			log.Println("No verification in progress, finishing")
			active.Finish()
		} else {
			log.Println("Another verification is already in progress, skipping")
		}
	}
}

func (state *activeTellStreamState) verifyDiffs(diffs string) {
	log.Println("Verifying diffs")

	plan := state.plan
	planId := plan.Id
	branch := state.branch

	active := GetActivePlan(planId, branch)

	if active == nil {
		return
	}

	defer func() {
		log.Println("Setting DidVerifyDiff to true")
		UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
			ap.DidVerifyDiff = true
			ap.IsVerifyingDiff = false
		})
	}()

	log.Println("Executing tell plan")

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
