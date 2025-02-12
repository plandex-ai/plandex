package plan

import (
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/types"
	"strings"
	"time"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

const MaxAutoContinueIterations = 100

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

	autoLoadContextFiles := state.checkAutoLoadContext()
	hasNewSubtasks := state.checkNewSubtasks()
	followUpNeedsContextStage := state.followUpNeedsContextStage()

	handleDescAndExecStatusRes := state.handleDescAndExecStatus(autoLoadContextFiles, hasNewSubtasks)
	if handleDescAndExecStatusRes.shouldContinueMainLoop || handleDescAndExecStatusRes.shouldReturn {
		return handleDescAndExecStatusRes.handleStreamFinishedResult
	}
	generatedDescription := handleDescAndExecStatusRes.generatedDescription
	shouldContinue := handleDescAndExecStatusRes.shouldContinue
	subtaskFinished := handleDescAndExecStatusRes.subtaskFinished

	storeOnFinishedResult := state.storeOnFinished(storeOnFinishedParams{
		replyOperations:           replyOperations,
		generatedDescription:      generatedDescription,
		subtaskFinished:           subtaskFinished,
		hasNewSubtasks:            hasNewSubtasks,
		followUpNeedsContextStage: followUpNeedsContextStage,
		autoLoadContextFiles:      autoLoadContextFiles,
	})
	if storeOnFinishedResult.shouldContinueMainLoop || storeOnFinishedResult.shouldReturn {
		return storeOnFinishedResult.handleStreamFinishedResult
	}
	allSubtasksFinished := storeOnFinishedResult.allSubtasksFinished

	// summarize convo needs to come *after* the reply is stored in order to correctly summarize the latest message
	log.Println("summarize convo")
	// summarize in the background
	go func() {
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

	log.Printf("len(autoLoadContextFiles): %d\n", len(autoLoadContextFiles))
	if len(autoLoadContextFiles) > 0 {
		log.Println("Sending stream message to load context files")

		active.Stream(shared.StreamMessage{
			Type:             shared.StreamMessageLoadContext,
			LoadContextFiles: autoLoadContextFiles,
		})
		active.FlushStreamBuffer()

		// Force a small delay to ensure message is processed
		time.Sleep(100 * time.Millisecond)

		log.Println("Waiting for client to auto load context (30s timeout)")

		select {
		case <-active.Ctx.Done():
			log.Println("Context cancelled while waiting for auto load context")
			state.execHookOnStop(true)
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

	if followUpNeedsContextStage {
		log.Println("Clearing context before follow up context stage")

	}

	// if we're auto-loading context files, we always want to continue for at least another iteration with the loaded context
	log.Printf("req.AutoContinue: %v\n", req.AutoContinue)
	log.Printf("shouldContinue: %v\n", shouldContinue)
	log.Printf("iteration: %d\n", iteration)
	log.Printf("MaxAutoContinueIterations: %d\n", MaxAutoContinueIterations)
	log.Printf("hasNewSubtasks: %v\n", hasNewSubtasks)
	log.Printf("len(state.subtasks): %d\n", len(state.subtasks))
	log.Printf("allSubtasksFinished: %v\n", allSubtasksFinished)
	log.Printf("followUpNeedsContextStage: %v\n", followUpNeedsContextStage)

	if followUpNeedsContextStage ||
		(len(autoLoadContextFiles) > 0 && !hasNewSubtasks) ||
		(len(autoLoadContextFiles) > 0 && req.IsChatOnly) ||
		(req.AutoContinue && shouldContinue && iteration < MaxAutoContinueIterations &&
			!(len(state.subtasks) > 0 && allSubtasksFinished)) {
		log.Println("Auto continue plan")
		// continue plan
		execTellPlan(execTellPlanParams{
			clients:                   clients,
			plan:                      plan,
			branch:                    branch,
			auth:                      auth,
			req:                       req,
			iteration:                 iteration + 1,
			shouldLoadFollowUpContext: followUpNeedsContextStage,
			didMakeFollowUpPlan:       state.isFollowUp && !followUpNeedsContextStage && hasNewSubtasks,
			didLoadChatOnlyContext:    len(autoLoadContextFiles) > 0 && req.IsChatOnly,
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

type handleDescAndExecStatusResult struct {
	handleStreamFinishedResult
	subtaskFinished      bool
	generatedDescription *db.ConvoMessageDescription
	shouldContinue       bool
}

func (state *activeTellStreamState) handleDescAndExecStatus(autoLoadContextFiles []string, hasNewSubtasks bool) handleDescAndExecStatusResult {
	req := state.req
	currentOrgId := state.currentOrgId
	summarizedToMessageId := state.summarizedToMessageId
	planId := state.plan.Id
	branch := state.branch
	replyOperations := state.chunkProcessor.replyOperations

	active := GetActivePlan(planId, branch)
	if active == nil {
		state.onActivePlanMissingError()
		return handleDescAndExecStatusResult{
			handleStreamFinishedResult: handleStreamFinishedResult{
				shouldContinueMainLoop: true,
				shouldReturn:           false,
			},
		}
	}

	var generatedDescription *db.ConvoMessageDescription
	var shouldContinue bool
	var subtaskFinished bool

	var errCh = make(chan *shared.ApiError, 2)

	go func() {
		if len(replyOperations) > 0 {
			log.Println("Generating plan description")

			res, err := state.genPlanDescription()
			if err != nil {
				errCh <- err
				return
			}

			generatedDescription = res
			generatedDescription.OrgId = currentOrgId
			generatedDescription.SummarizedToMessageId = summarizedToMessageId
			generatedDescription.MadePlan = true
			generatedDescription.Operations = replyOperations

			log.Println("Generated plan description.")
		}
		errCh <- nil
	}()

	if req.IsChatOnly || len(autoLoadContextFiles) > 0 || hasNewSubtasks || !req.AutoContinue {

		// if we're auto-loading context files, we always want to continue for at least another iteration with the loaded context (even if it's chat only)
		if len(autoLoadContextFiles) > 0 && !hasNewSubtasks {
			log.Printf("Auto loading context files, so continuing to planning phase")
			shouldContinue = true
		} else if req.IsChatOnly {
			log.Printf("Chat only, won't continue")
			shouldContinue = false
		} else if hasNewSubtasks {
			log.Printf("Has new subtasks, can continue")
			shouldContinue = req.AutoContinue
		}

		errCh <- nil
	} else {
		go func() {
			log.Println("Getting exec status")
			var err *shared.ApiError
			subtaskFinished, shouldContinue, err = state.execStatusShouldContinue(active.CurrentReplyContent, active.Ctx)
			if err != nil {
				errCh <- err
				return
			}

			log.Printf("Should continue: %v\n", shouldContinue)

			errCh <- nil
		}()
	}

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			res := state.onError(onErrorParams{
				streamApiErr: err,
				storeDesc:    true,
			})
			return handleDescAndExecStatusResult{
				handleStreamFinishedResult: handleStreamFinishedResult{
					shouldContinueMainLoop: res.shouldContinueMainLoop,
					shouldReturn:           res.shouldReturn,
				},
				subtaskFinished:      subtaskFinished,
				shouldContinue:       shouldContinue,
				generatedDescription: generatedDescription,
			}
		}
	}

	return handleDescAndExecStatusResult{
		handleStreamFinishedResult: handleStreamFinishedResult{},
		subtaskFinished:            subtaskFinished,
		shouldContinue:             shouldContinue,
		generatedDescription:       generatedDescription,
	}
}

type storeOnFinishedParams struct {
	replyOperations           []*shared.Operation
	generatedDescription      *db.ConvoMessageDescription
	subtaskFinished           bool
	hasNewSubtasks            bool
	followUpNeedsContextStage bool
	autoLoadContextFiles      []string
}

type storeOnFinishedResult struct {
	handleStreamFinishedResult
	allSubtasksFinished bool
}

func (state *activeTellStreamState) storeOnFinished(params storeOnFinishedParams) storeOnFinishedResult {
	replyOperations := params.replyOperations
	generatedDescription := params.generatedDescription
	subtaskFinished := params.subtaskFinished
	hasNewSubtasks := params.hasNewSubtasks
	followUpNeedsContextStage := params.followUpNeedsContextStage
	autoLoadContextFiles := params.autoLoadContextFiles
	currentOrgId := state.currentOrgId
	currentUserId := state.currentUserId
	planId := state.plan.Id
	branch := state.branch
	auth := state.auth
	summarizedToMessageId := state.summarizedToMessageId
	active := state.activePlan

	var allSubtasksFinished bool

	log.Println("Locking repo to store assistant reply and description")

	repoLockId, err := db.LockRepo(
		db.LockRepoParams{
			OrgId:    currentOrgId,
			UserId:   currentUserId,
			PlanId:   planId,
			Branch:   branch,
			Scope:    db.LockScopeWrite,
			Ctx:      active.Ctx,
			CancelFn: active.CancelFn,
		},
	)

	if err != nil {
		log.Printf("Error locking repo: %v\n", err)
		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error locking repo",
		}
		return storeOnFinishedResult{
			handleStreamFinishedResult: handleStreamFinishedResult{
				shouldContinueMainLoop: true,
				shouldReturn:           false,
			},
			allSubtasksFinished: false,
		}
	}

	log.Println("Locked repo for assistant reply and description")

	err = func() error {
		defer func() {
			if err != nil {
				log.Printf("Error storing reply and description: %v\n", err)
				err = db.GitClearUncommittedChanges(auth.OrgId, planId)
				if err != nil {
					log.Printf("Error clearing uncommitted changes: %v\n", err)
				}
			}

			log.Println("Unlocking repo for assistant reply and description")

			err = db.DeleteRepoLock(repoLockId, planId)
			if err != nil {
				log.Printf("Error unlocking repo: %v\n", err)
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Error unlocking repo",
				}
			}
		}()

		var replyType shared.ReplyType
		if len(replyOperations) > 0 {
			replyType = shared.ReplyTypeImplementation
		} else if hasNewSubtasks {
			replyType = shared.ReplyTypeMadePlan
		} else if len(autoLoadContextFiles) > 0 {
			replyType = shared.ReplyTypeLoadedContext
		} else if followUpNeedsContextStage {
			replyType = shared.ReplyTypeContextAssessment
		} else {
			replyType = shared.ReplyTypeChat
		}

		assistantMsg, convoCommitMsg, err := state.storeAssistantReply(replyType) // updates state.convo

		if err != nil {
			state.onError(onErrorParams{
				streamErr: fmt.Errorf("failed to store assistant message: %v", err),
				storeDesc: true,
			})
			return err
		}

		log.Println("getting description for assistant message: ", assistantMsg.Id)

		var description *db.ConvoMessageDescription
		if len(replyOperations) == 0 {
			description = &db.ConvoMessageDescription{
				OrgId:                 currentOrgId,
				PlanId:                planId,
				ConvoMessageId:        assistantMsg.Id,
				SummarizedToMessageId: summarizedToMessageId,
				BuildPathsInvalidated: map[string]bool{},
				MadePlan:              false,
			}
		} else {
			description = generatedDescription
			description.ConvoMessageId = assistantMsg.Id
		}

		log.Println("Storing description")
		err = db.StoreDescription(description)

		if err != nil {
			state.onError(onErrorParams{
				streamErr:      fmt.Errorf("failed to store description: %v", err),
				storeDesc:      false,
				convoMessageId: assistantMsg.Id,
				commitMsg:      convoCommitMsg,
			})
			return err
		}
		log.Println("Description stored")

		if hasNewSubtasks || subtaskFinished {
			if subtaskFinished && state.currentSubtask != nil {
				log.Println("Subtask finished")
				log.Println("Current subtask:")
				log.Println(state.currentSubtask.Title)
				state.currentSubtask.IsFinished = true

				log.Println("Updated state. Current subtask:")
				log.Println(state.currentSubtask)
			}

			log.Println("Storing plan subtasks")
			err = db.StorePlanSubtasks(currentOrgId, planId, state.subtasks)
			if err != nil {
				log.Printf("Error storing plan subtasks: %v\n", err)
				state.onError(onErrorParams{
					streamErr:      fmt.Errorf("failed to store plan subtasks: %v", err),
					storeDesc:      false,
					convoMessageId: assistantMsg.Id,
					commitMsg:      convoCommitMsg,
				})
				return err
			}

			state.currentSubtask = nil
			allSubtasksFinished = true
			for _, subtask := range state.subtasks {
				if !subtask.IsFinished {
					state.currentSubtask = subtask
					allSubtasksFinished = false
					break
				}
			}

			log.Println("Set new current subtask. Current subtask:")
			log.Println(state.currentSubtask)
			log.Println("All subtasks finished:", allSubtasksFinished)

			// log.Println("Update state of subtasks")
			// spew.Dump(state.subtasks)

		}

		if followUpNeedsContextStage {
			log.Println("Clearing non-map/non-pending context before follow up context stage")
			err := db.ClearContext(db.ClearContextParams{
				OrgId:       currentOrgId,
				PlanId:      planId,
				SkipMaps:    true,
				SkipPending: true,
			})
			if err != nil {
				log.Printf("Error clearing context: %v\n", err)
			}
		}

		log.Println("Comitting after store on finished")

		err = db.GitAddAndCommit(currentOrgId, planId, branch, convoCommitMsg)
		if err != nil {
			state.onError(onErrorParams{
				streamErr:      fmt.Errorf("failed to commit: %v", err),
				storeDesc:      false,
				convoMessageId: assistantMsg.Id,
				commitMsg:      convoCommitMsg,
			})
			return err
		}
		log.Println("Assistant reply, description, and subtasks committed")

		return nil
	}()

	if err != nil {
		return storeOnFinishedResult{
			handleStreamFinishedResult: handleStreamFinishedResult{
				shouldContinueMainLoop: true,
				shouldReturn:           false,
			},
			allSubtasksFinished: allSubtasksFinished,
		}
	}

	return storeOnFinishedResult{
		handleStreamFinishedResult: handleStreamFinishedResult{},
		allSubtasksFinished:        allSubtasksFinished,
	}
}

func (state *activeTellStreamState) storeAssistantReply(replyType shared.ReplyType) (*db.ConvoMessage, string, error) {
	currentOrgId := state.currentOrgId
	currentUserId := state.currentUserId
	planId := state.plan.Id
	branch := state.branch
	auth := state.auth
	replyNumTokens := state.replyNumTokens
	replyId := state.replyId
	convo := state.convo

	num := len(convo) + 1

	log.Printf("storing assistant reply | len(convo) %d | num %d\n", len(convo), num)

	activePlan := state.activePlan

	assistantMsg := db.ConvoMessage{
		Id:        replyId,
		OrgId:     currentOrgId,
		PlanId:    planId,
		UserId:    currentUserId,
		Role:      openai.ChatMessageRoleAssistant,
		Tokens:    replyNumTokens,
		Num:       num,
		Message:   activePlan.CurrentReplyContent,
		ReplyType: replyType,
	}

	commitMsg, err := db.StoreConvoMessage(&assistantMsg, auth.User.Id, branch, false)

	if err != nil {
		log.Printf("Error storing assistant message: %v\n", err)
		return nil, "", err
	}

	UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
		ap.MessageNum = num
		ap.StoredReplyIds = append(ap.StoredReplyIds, replyId)
	})

	convo = append(convo, &assistantMsg)
	state.convo = convo

	return &assistantMsg, commitMsg, err
}

func (state *activeTellStreamState) followUpNeedsContextStage() bool {
	if !(state.isPlanningStage && state.isFollowUp && state.iteration == 0) {
		return false
	}

	active := state.activePlan

	return strings.Contains(active.CurrentReplyContent, "clear all context") ||
		strings.Contains(active.CurrentReplyContent, "decide what context I need")
}
