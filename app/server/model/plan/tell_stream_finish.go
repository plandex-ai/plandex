package plan

import (
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/notify"
	"plandex-server/types"
	"runtime/debug"
	"time"

	shared "plandex-shared"

	"github.com/davecgh/go-spew/spew"
)

const MaxAutoContinueIterations = 200

type handleStreamFinishedResult struct {
	shouldContinueMainLoop bool
	shouldReturn           bool
}

func (state *activeTellStreamState) handleStreamFinished() handleStreamFinishedResult {
	planId := state.plan.Id
	branch := state.branch
	auth := state.auth
	plan := state.plan
	req := state.req
	clients := state.clients
	settings := state.settings
	currentOrgId := state.currentOrgId
	summaries := state.summaries
	convo := state.convo
	iteration := state.iteration
	replyOperations := state.chunkProcessor.replyOperations

	err := state.setActivePlan()
	if err != nil {
		state.onActivePlanMissingError()
		return handleStreamFinishedResult{
			shouldContinueMainLoop: true,
			shouldReturn:           false,
		}
	}

	active := state.activePlan

	time.Sleep(30 * time.Millisecond)
	active.FlushStreamBuffer()
	time.Sleep(100 * time.Millisecond)

	active.Stream(shared.StreamMessage{
		Type: shared.StreamMessageDescribing,
	})
	active.FlushStreamBuffer()

	err = db.SetPlanStatus(planId, branch, shared.PlanStatusDescribing, "")
	if err != nil {
		res := state.onError(onErrorParams{
			streamErr: fmt.Errorf("failed to set plan status to describing: %v", err),
			storeDesc: true,
		})

		return handleStreamFinishedResult{
			shouldContinueMainLoop: res.shouldContinueMainLoop,
			shouldReturn:           res.shouldReturn,
		}
	}

	autoLoadContextResult := state.checkAutoLoadContext()
	addedSubtasks := state.checkNewSubtasks()
	removedSubtasks := state.checkRemoveSubtasks()
	hasNewSubtasks := len(addedSubtasks) > 0

	log.Println("removedSubtasks:\n", spew.Sdump(removedSubtasks))
	log.Println("addedSubtasks:\n", spew.Sdump(addedSubtasks))
	log.Println("hasNewSubtasks:\n", hasNewSubtasks)

	handleDescAndExecStatusRes := state.handleDescAndExecStatus()
	if handleDescAndExecStatusRes.shouldContinueMainLoop || handleDescAndExecStatusRes.shouldReturn {
		return handleDescAndExecStatusRes.handleStreamFinishedResult
	}
	generatedDescription := handleDescAndExecStatusRes.generatedDescription
	subtaskFinished := handleDescAndExecStatusRes.subtaskFinished

	log.Printf("subtaskFinished: %v\n", subtaskFinished)

	storeOnFinishedResult := state.storeOnFinished(storeOnFinishedParams{
		replyOperations:       replyOperations,
		generatedDescription:  generatedDescription,
		subtaskFinished:       subtaskFinished,
		hasNewSubtasks:        hasNewSubtasks,
		autoLoadContextResult: autoLoadContextResult,
		addedSubtasks:         addedSubtasks,
		removedSubtasks:       removedSubtasks,
	})
	if storeOnFinishedResult.shouldContinueMainLoop || storeOnFinishedResult.shouldReturn {
		return storeOnFinishedResult.handleStreamFinishedResult
	}
	allSubtasksFinished := storeOnFinishedResult.allSubtasksFinished

	log.Println("allSubtasksFinished:\n", spew.Sdump(allSubtasksFinished))

	// summarize convo needs to come *after* the reply is stored in order to correctly summarize the latest message
	log.Println("summarizing convo in background")
	// summarize in the background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in summarizeConvo: %v\n%s", r, debug.Stack())
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    fmt.Sprintf("Error summarizing convo: %v", r),
				}
			}
		}()

		err := summarizeConvo(clients, settings.ModelPack.PlanSummary, summarizeConvoParams{
			auth:                  auth,
			plan:                  plan,
			branch:                branch,
			convo:                 convo,
			summaries:             summaries,
			userPrompt:            state.userPrompt,
			currentOrgId:          currentOrgId,
			currentReply:          active.CurrentReplyContent,
			currentReplyNumTokens: active.NumTokens,
			modelPackName:         settings.ModelPack.Name,
		}, active.SummaryCtx)

		if err != nil {
			log.Printf("Error summarizing convo: %v\n", err)
			active.StreamDoneCh <- err
		}
	}()

	log.Println("Sending active.CurrentReplyDoneCh <- true")

	active.CurrentReplyDoneCh <- true

	log.Println("Resetting active.CurrentReplyDoneCh")

	UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
		ap.CurrentStreamingReplyId = ""
		ap.CurrentReplyDoneCh = nil
	})

	autoLoadPaths := autoLoadContextResult.autoLoadPaths
	log.Printf("len(autoLoadPaths): %d\n", len(autoLoadPaths))
	if len(autoLoadPaths) > 0 {
		log.Println("Sending stream message to load context files")

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic streaming auto-load context: %v\n%s", r, debug.Stack())
					go notify.NotifyErr(notify.SeverityError, fmt.Errorf("panic streaming auto-load context: %v\n%s", r, debug.Stack()))
				}
			}()

			active.Stream(shared.StreamMessage{
				Type:             shared.StreamMessageLoadContext,
				LoadContextFiles: autoLoadPaths,
			})
			active.FlushStreamBuffer()
		}()

		log.Println("Waiting for client to auto load context (30s timeout)")

		select {
		case <-active.Ctx.Done():
			log.Println("Context cancelled while waiting for auto load context")
			state.execHookOnStop(false)
			return handleStreamFinishedResult{
				shouldContinueMainLoop: false,
				shouldReturn:           true,
			}
		case <-time.After(30 * time.Second):
			log.Println("Timeout waiting for auto load context")
			res := state.onError(onErrorParams{
				streamErr: fmt.Errorf("timeout waiting for auto load context response"),
				storeDesc: true,
			})
			return handleStreamFinishedResult{
				shouldContinueMainLoop: res.shouldContinueMainLoop,
				shouldReturn:           res.shouldReturn,
			}
		case <-active.AutoLoadContextCh:
		}
	}

	willContinue := state.willContinuePlan(willContinuePlanParams{
		hasNewSubtasks:      hasNewSubtasks,
		allSubtasksFinished: allSubtasksFinished,
		activatePaths:       autoLoadContextResult.activatePaths,
		removedSubtasks:     len(removedSubtasks) > 0,
		hasExplicitPaths:    autoLoadContextResult.hasExplicitPaths,
	})

	if willContinue {
		log.Println("Auto continue plan")
		// continue plan
		execTellPlan(execTellPlanParams{
			clients:   clients,
			plan:      plan,
			branch:    branch,
			auth:      auth,
			req:       req,
			iteration: iteration + 1,
		})
	} else {
		var buildFinished bool
		UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
			buildFinished = ap.BuildFinished()
			ap.RepliesFinished = true
		})

		log.Printf("Won't continue plan. Build finished: %v\n", buildFinished)

		time.Sleep(50 * time.Millisecond)

		if buildFinished {
			log.Println("Reply is finished and build is finished, calling active.Finish()")
			active := GetActivePlan(planId, branch)

			if active == nil {
				state.onActivePlanMissingError()
				return handleStreamFinishedResult{
					shouldContinueMainLoop: true,
					shouldReturn:           false,
				}
			}

			active.Finish()
		} else {
			log.Println("Plan is still building")
			log.Println("Updating status to building")
			err := db.SetPlanStatus(planId, branch, shared.PlanStatusBuilding, "")
			if err != nil {
				log.Printf("Error setting plan status to building: %v\n", err)
				go notify.NotifyErr(notify.SeverityError, fmt.Errorf("error setting plan status to building: %v", err))

				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Error setting plan status to building",
				}

				return handleStreamFinishedResult{
					shouldContinueMainLoop: true,
					shouldReturn:           false,
				}
			}

			log.Println("Sending RepliesFinished stream message")
			active.Stream(shared.StreamMessage{
				Type: shared.StreamMessageRepliesFinished,
			})

		}
	}

	return handleStreamFinishedResult{}
}
