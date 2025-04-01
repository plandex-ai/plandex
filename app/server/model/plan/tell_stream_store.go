package plan

import (
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/notify"
	"plandex-server/types"
	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

type storeOnFinishedParams struct {
	replyOperations       []*shared.Operation
	generatedDescription  *db.ConvoMessageDescription
	subtaskFinished       bool
	hasNewSubtasks        bool
	autoLoadContextResult checkAutoLoadContextResult
	addedSubtasks         []*db.Subtask
	removedSubtasks       []string
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
	autoLoadContextResult := params.autoLoadContextResult
	currentOrgId := state.currentOrgId
	currentUserId := state.currentUserId
	planId := state.plan.Id
	branch := state.branch
	summarizedToMessageId := state.summarizedToMessageId
	active := state.activePlan
	addedSubtasks := params.addedSubtasks
	removedSubtasks := params.removedSubtasks
	var allSubtasksFinished bool

	log.Println("[storeOnFinished] Locking repo to store assistant reply and description")

	err := db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    currentOrgId,
		UserId:   currentUserId,
		PlanId:   planId,
		Branch:   branch,
		Scope:    db.LockScopeWrite,
		Ctx:      active.Ctx,
		CancelFn: active.CancelFn,
		Reason:   "store on finished",
	}, func(repo *db.GitRepo) error {
		log.Println("storeOnFinished: hasNewSubtasks", hasNewSubtasks)
		log.Println("storeOnFinished: subtaskFinished", subtaskFinished)
		log.Println("storeOnFinished: removedSubtasks", removedSubtasks)

		messageSubtask := state.currentSubtask

		// first resolve subtask state
		if hasNewSubtasks || len(removedSubtasks) > 0 || subtaskFinished {
			if subtaskFinished && state.currentSubtask != nil {
				log.Printf("[storeOnFinished] Marking subtask as finished: %q", state.currentSubtask.Title)
				state.currentSubtask.IsFinished = true

				log.Printf("[storeOnFinished] Current subtask state after marking as finished: %+v", state.currentSubtask)
			}

			log.Printf("[storeOnFinished] Storing plan subtasks (hasNewSubtasks=%v, subtaskFinished=%v)", hasNewSubtasks, subtaskFinished)
			log.Printf("[storeOnFinished] Current subtasks state before storing:")
			for i, task := range state.subtasks {
				log.Printf("[storeOnFinished] Task %d: %q (finished=%v)", i+1, task.Title, task.IsFinished)
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

			if state.currentSubtask != nil {
				log.Printf("[storeOnFinished] Set new current subtask: %q", state.currentSubtask.Title)
			} else {
				log.Println("[storeOnFinished] No new current subtask set")
			}
			log.Printf("[storeOnFinished] All subtasks finished: %v", allSubtasksFinished)
		} else if state.currentSubtask != nil && !subtaskFinished {
			log.Printf("[storeOnFinished] Current subtask is not finished: %q", state.currentSubtask.Title)
			state.currentSubtask.NumTries++
		}

		log.Println("storeOnFinished: state.currentSubtask", state.currentSubtask)
		log.Println("storeOnFinished: state.subtasks", state.subtasks)
		log.Println("storeOnFinished: state.currentStage", state.currentStage)

		var flags shared.ConvoMessageFlags

		flags.CurrentStage = state.currentStage

		if len(replyOperations) > 0 {
			flags.DidWriteCode = true
		}
		if hasNewSubtasks {
			log.Println("storeOnFinished: hasNewSubtasks")
			flags.DidMakePlan = true
		}
		if len(removedSubtasks) > 0 {
			log.Println("storeOnFinished: len(removedSubtasks) > 0")
			flags.DidMakePlan = true
			flags.DidRemoveTasks = true
		}
		if len(autoLoadContextResult.autoLoadPaths) > 0 {
			flags.DidLoadContext = true
		}
		if subtaskFinished && messageSubtask != nil {
			flags.DidCompleteTask = true
		}
		if allSubtasksFinished {
			log.Println("storeOnFinished: allSubtasksFinished")
			flags.DidCompletePlan = true
		}
		if hasNewSubtasks && (state.req.IsApplyDebug || state.req.IsUserDebug) {
			log.Println("storeOnFinished: hasNewSubtasks && (state.req.IsApplyDebug || state.req.IsUserDebug)")
			flags.DidMakeDebuggingPlan = true
		}

		log.Println("storeOnFinished: flags", flags)

		assistantMsg, convoCommitMsg, err := state.storeAssistantReply(repo, storeAssistantReplyParams{
			flags:                flags,
			subtask:              messageSubtask,
			addedSubtasks:        addedSubtasks,
			activatePaths:        autoLoadContextResult.activatePaths,
			activatePathsOrdered: autoLoadContextResult.activatePathsOrdered,
			removedSubtasks:      removedSubtasks,
		}) // updates state.convo

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
				WroteFiles:            false,
			}
		} else {
			description = generatedDescription
			description.ConvoMessageId = assistantMsg.Id
		}

		log.Println("[storeOnFinished] Storing description")
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
		log.Println("[storeOnFinished] Description stored")

		// store subtasks
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

		log.Println("Comitting after store on finished")

		err = repo.GitAddAndCommit(branch, convoCommitMsg)
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
	})

	if err != nil {
		log.Printf("Error storing on finished: %v\n", err)
		go notify.NotifyErr(notify.SeverityError, fmt.Errorf("error storing on finished: %v", err))

		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error storing on finished",
		}
		return storeOnFinishedResult{
			handleStreamFinishedResult: handleStreamFinishedResult{
				shouldContinueMainLoop: true,
				shouldReturn:           false,
			},
			allSubtasksFinished: false,
		}
	}

	return storeOnFinishedResult{
		handleStreamFinishedResult: handleStreamFinishedResult{},
		allSubtasksFinished:        allSubtasksFinished,
	}
}

type storeAssistantReplyParams struct {
	flags                shared.ConvoMessageFlags
	subtask              *db.Subtask
	addedSubtasks        []*db.Subtask
	activatePaths        map[string]bool
	activatePathsOrdered []string
	removedSubtasks      []string
}

func (state *activeTellStreamState) storeAssistantReply(repo *db.GitRepo, params storeAssistantReplyParams) (*db.ConvoMessage, string, error) {
	flags := params.flags
	subtask := params.subtask
	addedSubtasks := params.addedSubtasks
	activatePaths := params.activatePaths
	activatePathsOrdered := params.activatePathsOrdered
	removedSubtasks := params.removedSubtasks

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

	// fmt.Println("raw message: ", activePlan.CurrentReplyContent)

	assistantMsg := db.ConvoMessage{
		Id:                    replyId,
		OrgId:                 currentOrgId,
		PlanId:                planId,
		UserId:                currentUserId,
		Role:                  openai.ChatMessageRoleAssistant,
		Tokens:                replyNumTokens,
		Num:                   num,
		Message:               activePlan.CurrentReplyContent,
		Flags:                 flags,
		Subtask:               subtask,
		AddedSubtasks:         addedSubtasks,
		ActivatedPaths:        activatePaths,
		ActivatedPathsOrdered: activatePathsOrdered,
		RemovedSubtasks:       removedSubtasks,
	}

	commitMsg, err := db.StoreConvoMessage(repo, &assistantMsg, auth.User.Id, branch, false)

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
