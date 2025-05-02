package plan

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"plandex-server/db"
	"plandex-server/model"
	"plandex-server/notify"
	"plandex-server/shutdown"
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
	modelErr       *shared.ModelError
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
	modelErr := params.modelErr

	planId := state.plan.Id
	branch := state.branch
	currentOrgId := state.currentOrgId
	summarizedToMessageId := state.summarizedToMessageId

	active := GetActivePlan(planId, branch)

	if active == nil {
		log.Printf("tellStream onError - Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return onErrorResult{
			shouldReturn: true,
		}
	}

	canRetry := params.canRetry
	hasFallback := state.fallbackRes.HasErrorFallback
	isFallback := state.fallbackRes.IsFallback

	maxRetries := model.MAX_RETRIES_WITHOUT_FALLBACK
	if hasFallback {
		maxRetries = model.MAX_ADDITIONAL_RETRIES_WITH_FALLBACK
	}

	compareRetries := state.numErrorRetry
	if isFallback {
		compareRetries = state.numFallbackRetry
	}

	newFallback := false
	if modelErr != nil {
		if !modelErr.Retriable {
			log.Printf("tellStream onError - operation returned non-retriable error: %v", modelErr)
			if modelErr.Kind == shared.ErrContextTooLong && state.fallbackRes.ModelRoleConfig.LargeContextFallback == nil {
				log.Printf("tellStream onError - non-retriable context too long error and no large context fallback is defined, no retry")
				// if it's a context too long error and no large context fallback is defined, no retry
				canRetry = false

			} else if modelErr.Kind != shared.ErrContextTooLong && state.fallbackRes.ModelRoleConfig.ErrorFallback == nil {
				log.Printf("tellStream onError - non-retriable error and no error fallback is defined, no retry")
				// if it's any other error and no error fallback is defined, no retry
				canRetry = false
			} else {
				log.Printf("tellStream onError - operation returned non-retriable error, but has fallback - resetting numFallbackRetry to 0 and continuing to retry")
				state.numFallbackRetry = 0
				// otherwise, continue to retry logic
				canRetry = true
				newFallback = true
			}
		}
	}

	if canRetry {
		log.Println("tellStream onError - canRetry", canRetry)

		if compareRetries >= maxRetries {
			log.Printf("tellStream onError - Max retries reached for plan ID %s on branch %s\n", planId, branch)

			canRetry = false
		}
	}

	if canRetry {
		log.Println("tellStream onError - retrying stream")
		// stop stream via context (ensures we stop child streams too)
		active.CancelModelStreamFn()

		active.ResetModelCtx()

		var retryDelay time.Duration
		if modelErr != nil && modelErr.RetryAfterSeconds > 0 {
			// if the model err has a retry after, then use that with a bit of padding
			retryDelay = time.Duration(int(float64(modelErr.RetryAfterSeconds)*1.1)) * time.Second
		} else {
			// otherwise, use some jitter
			retryDelay = time.Duration(1000+rand.Intn(200)) * time.Millisecond
		}

		cacheSupportErr := false
		numErrorRetry := state.numErrorRetry
		if !(modelErr != nil && modelErr.Kind == shared.ErrCacheSupport) {
			numErrorRetry = numErrorRetry + 1
		} else {
			cacheSupportErr = true
		}

		log.Printf("tellStream onError - Retry %d/%d - Retrying stream in %v", numErrorRetry, maxRetries, retryDelay)
		time.Sleep(retryDelay)

		state.numErrorRetry = numErrorRetry
		if isFallback && !newFallback && !cacheSupportErr {
			state.numFallbackRetry = state.numFallbackRetry + 1
		}

		// if we got a cache support error, keep everything the same, including the modelErr (if we're already retrying) so we can make the exact same request again without cache control breakpoints
		if cacheSupportErr {
			state.noCacheSupportErr = true
		} else {
			state.modelErr = modelErr

			if newFallback {
				// if we got a new fallback, we need to reset the noCacheSupportErr flag since we're using a different model now
				state.noCacheSupportErr = false
			}
		}

		// retry the request
		state.doTellRequest()
		return onErrorResult{
			shouldReturn: true,
		}
	}

	storeDescAndReply := func() error {
		log.Println("tellStream onError - storing desc and reply")
		ctx, cancelFn := context.WithTimeout(shutdown.ShutdownCtx, 5*time.Second)

		err := db.ExecRepoOperation(db.ExecRepoOperationParams{
			OrgId:    currentOrgId,
			UserId:   state.currentUserId,
			PlanId:   planId,
			Branch:   branch,
			Scope:    db.LockScopeWrite,
			Ctx:      ctx,
			CancelFn: cancelFn,
			Reason:   "store desc and reply",
		}, func(repo *db.GitRepo) error {
			storedMessage := false
			storedDesc := false

			if convoMessageId == "" {
				hasUnfinishedSubtasks := false
				for _, subtask := range state.subtasks {
					if !subtask.IsFinished {
						hasUnfinishedSubtasks = true
						break
					}
				}

				assistantMsg, msg, err := state.storeAssistantReply(repo, storeAssistantReplyParams{
					flags: shared.ConvoMessageFlags{
						CurrentStage:          state.currentStage,
						HasUnfinishedSubtasks: hasUnfinishedSubtasks,
						HasError:              true,
					},
					subtask:       nil,
					addedSubtasks: nil,
				})
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
					WroteFiles:            false,
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
				err := repo.GitAddAndCommit(branch, commitMsg)
				if err != nil {
					log.Printf("Error committing after stream error: %v\n", err)
					return err
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("Error storing description and reply after stream error: %v\n", err)
			return err
		}

		return nil
	}

	if active.CurrentReplyContent != "" {
		storeDescAndReply() // best effort to store description and reply, ignore errors
	}

	if params.streamApiErr != nil {
		active.StreamDoneCh <- params.streamApiErr
	} else {
		msg := "Stream error: " + streamErr.Error()
		if params.canRetry && state.numErrorRetry >= maxRetries {
			msg += " | Failed after " + strconv.Itoa(state.numErrorRetry) + " retries"
		}

		go notify.NotifyErr(notify.SeverityInfo, fmt.Sprintf("tellStream stream error after %d retries: %v", state.numErrorRetry, streamErr))

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
