package plan

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"

	"github.com/plandex/plandex/shared"
)

func (state *activeTellStreamState) onError(streamErr error, storeDesc bool, convoMessageId, commitMsg string) {
	log.Printf("\nStream error: %v\n", streamErr)

	planId := state.plan.Id
	branch := state.branch
	currentOrgId := state.currentOrgId
	summarizedToMessageId := state.summarizedToMessageId

	active := GetActivePlan(planId, branch)

	if active == nil {
		log.Printf("tellStream onError - Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return
	}

	storeDescAndReply := func() error {
		ctx, cancelFn := context.WithCancel(context.Background())

		repoLockId, err := db.LockRepo(
			db.LockRepoParams{
				UserId:   state.currentUserId,
				OrgId:    state.currentOrgId,
				PlanId:   planId,
				Branch:   branch,
				Scope:    db.LockScopeWrite,
				Ctx:      ctx,
				CancelFn: cancelFn,
			},
		)

		if err != nil {
			log.Printf("Error locking repo for plan %s: %v\n", planId, err)
			return err
		} else {

			defer func() {
				err := db.DeleteRepoLock(repoLockId)
				if err != nil {
					log.Printf("Error unlocking repo for plan %s: %v\n", planId, err)
				}
			}()

			err := db.GitClearUncommittedChanges(state.currentOrgId, planId)
			if err != nil {
				log.Printf("Error clearing uncommitted changes for plan %s: %v\n", planId, err)
				return err
			}
		}

		storedMessage := false
		storedDesc := false

		if convoMessageId == "" {
			assistantMsg, msg, err := state.storeAssistantReply("")
			if err == nil {
				convoMessageId = assistantMsg.Id
				commitMsg = msg
				storedMessage = true
			} else {
				log.Printf("Error storing assistant message after stream error: %v\n", err)
				return err
			}
		}

		if storeDesc && convoMessageId != "" {
			err := db.StoreDescription(&db.ConvoMessageDescription{
				OrgId:                 currentOrgId,
				PlanId:                planId,
				SummarizedToMessageId: summarizedToMessageId,
				MadePlan:              false,
				ConvoMessageId:        convoMessageId,
				BuildPathsInvalidated: map[string]bool{},
				Error:                 streamErr.Error(),
			})
			if err == nil {
				storedDesc = true
			} else {
				log.Printf("Error storing description after stream error: %v\n", err)
				return err
			}
		}

		if storedMessage || storedDesc {
			err := db.GitAddAndCommit(currentOrgId, planId, branch, commitMsg)
			if err != nil {
				log.Printf("Error committing after stream error: %v\n", err)
				return err
			}
		}

		return nil
	}

	storeDescAndReply()

	active.StreamDoneCh <- &shared.ApiError{
		Type:   shared.ApiErrorTypeOther,
		Status: http.StatusInternalServerError,
		Msg:    "Stream error: " + streamErr.Error(),
	}
}

func (state *activeTellStreamState) onActivePlanMissingError() {
	planId := state.plan.Id
	branch := state.branch
	log.Printf("Active plan not found for plan ID %s on branch %s\n", planId, branch)
	state.onError(fmt.Errorf("active plan not found for plan ID %s on branch %s", planId, branch), true, "", "")
}
