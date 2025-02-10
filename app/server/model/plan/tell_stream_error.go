package plan

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"strconv"
	"time"

	shared "plandex-shared"
)

type onErrorParams struct {
	streamErr      error
	streamApiErr   *shared.ApiError
	storeDesc      bool
	convoMessageId string
	commitMsg      string
	canRetry       bool
}

type onErrorResult struct {
	shouldContinueMainLoop bool
	shouldReturn           bool
}

func (state *activeTellStreamState) onError(params onErrorParams) onErrorResult {
	log.Printf("\nStream error: %v\n", params.streamErr)
	streamErr := params.streamErr
	storeDesc := params.storeDesc
	convoMessageId := params.convoMessageId
	commitMsg := params.commitMsg

	planId := state.plan.Id
	branch := state.branch
	currentOrgId := state.currentOrgId
	summarizedToMessageId := state.summarizedToMessageId

	active := GetActivePlan(planId, branch)
	numRetries := state.execTellPlanParams.numErrorRetry

	if active == nil {
		log.Printf("tellStream onError - Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return onErrorResult{
			shouldReturn: true,
		}
	}

	canRetry := params.canRetry

	if canRetry {
		if numRetries >= NumTellStreamRetries {
			log.Printf("tellStream onError - Max retries reached for plan ID %s on branch %s\n", planId, branch)

			canRetry = false
		}
	}

	if canRetry {
		// stop stream via context (ensures we stop child streams too)
		active.CancelModelStreamFn()

		active.ResetModelCtx()

		retryDelaySeconds := 1 * numRetries * (numRetries / 2)

		log.Printf("tellStream onError - Retry %d/%d - Retrying stream in %d seconds", numRetries+1, NumTellStreamRetries, retryDelaySeconds)
		time.Sleep(time.Duration(retryDelaySeconds) * time.Second)

		params := state.execTellPlanParams
		params.numErrorRetry = numRetries + 1

		execTellPlan(params)
		return onErrorResult{
			shouldReturn: true,
		}
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

	if params.streamApiErr != nil {
		active.StreamDoneCh <- params.streamApiErr
	} else {
		msg := "Stream error: " + streamErr.Error()
		if params.canRetry && numRetries >= NumTellStreamRetries {
			msg += " | Failed after " + strconv.Itoa(numRetries) + " retries."
		}

		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    msg,
		}
	}

	return onErrorResult{
		shouldContinueMainLoop: true,
	}
}

func (state *activeTellStreamState) onActivePlanMissingError() {
	planId := state.plan.Id
	branch := state.branch
	log.Printf("Active plan not found for plan ID %s on branch %s\n", planId, branch)
	state.onError(onErrorParams{
		streamErr: fmt.Errorf("active plan not found for plan ID %s on branch %s", planId, branch),
		storeDesc: true,
	})
}
