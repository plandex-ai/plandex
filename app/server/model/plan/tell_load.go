package plan

import (
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/model"
	"plandex-server/notify"
	"plandex-server/types"
	"runtime"
	"runtime/debug"

	shared "plandex-shared"

	"github.com/jmoiron/sqlx"
	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) loadTellPlan() error {
	clients := state.clients
	req := state.req
	auth := state.auth
	plan := state.plan
	planId := plan.Id
	branch := state.branch
	currentUserId := state.currentUserId
	currentOrgId := state.currentOrgId
	iteration := state.iteration
	missingFileResponse := state.missingFileResponse

	err := state.setActivePlan()
	if err != nil {
		return err
	}
	active := state.activePlan

	lockScope := db.LockScopeWrite
	if iteration > 0 || missingFileResponse != "" {
		lockScope = db.LockScopeRead
	}

	var modelContext []*db.Context
	var convo []*db.ConvoMessage
	var promptMsg *db.ConvoMessage
	var summaries []*db.ConvoSummary
	var subtasks []*db.Subtask
	var settings *shared.PlanSettings
	var latestSummaryTokens int
	var currentPlan *shared.CurrentPlanState

	log.Printf("[TellLoad] Tell plan - loadTellPlan - iteration: %d, missingFileResponse: %s, req.IsUserContinue: %t, lockScope: %s\n", iteration, missingFileResponse, req.IsUserContinue, lockScope)

	db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    auth.OrgId,
		UserId:   auth.User.Id,
		PlanId:   planId,
		Branch:   branch,
		Scope:    lockScope,
		Ctx:      active.Ctx,
		CancelFn: active.CancelFn,
		Reason:   "load tell plan",
	}, func(repo *db.GitRepo) error {
		errCh := make(chan error, 4)

		// get name for plan and rename if it's a draft
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in getPlanSettings: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("error getting plan settings: %v", r)
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()

			res, err := db.GetPlanSettings(plan, true)
			if err != nil {
				log.Printf("Error getting plan settings: %v\n", err)
				errCh <- fmt.Errorf("error getting plan settings: %v", err)
				return
			}
			settings = res

			if plan.Name == "draft" {
				name, err := model.GenPlanName(
					auth,
					plan,
					settings,
					clients,
					req.Prompt,
					active.SessionId,
					active.Ctx,
				)

				if err != nil {
					log.Printf("Error generating plan name: %v\n", err)
					errCh <- fmt.Errorf("error generating plan name: %v", err)
					return
				}

				err = db.WithTx(active.Ctx, "rename plan", func(tx *sqlx.Tx) error {
					err := db.RenamePlan(planId, name, tx)

					if err != nil {
						log.Printf("Error renaming plan: %v\n", err)
						return fmt.Errorf("error renaming plan: %v", err)
					}

					err = db.IncNumNonDraftPlans(currentUserId, tx)

					if err != nil {
						log.Printf("Error incrementing num non draft plans: %v\n", err)
						return fmt.Errorf("error incrementing num non draft plans: %v", err)
					}

					return nil
				})

				if err != nil {
					log.Printf("Error renaming plan: %v\n", err)
					errCh <- fmt.Errorf("error renaming plan: %v", err)
					return
				}
			}

			errCh <- nil
		}()

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in getPlanContexts: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("error getting plan modelContext: %v", r)
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()

			if iteration > 0 || missingFileResponse != "" {
				modelContext = active.Contexts
			} else {
				res, err := db.GetPlanContexts(currentOrgId, planId, true, false)
				if err != nil {
					log.Printf("Error getting plan modelContext: %v\n", err)
					errCh <- fmt.Errorf("error getting plan modelContext: %v", err)
					return
				}

				log.Printf("[TellLoad] Tell plan - loadTellPlan - modelContext: %v\n", len(modelContext))
				// for _, part := range modelContext {
				// 	log.Printf("[TellLoad] Tell plan - loadTellPlan - part: %s - %s - %s - %d tokens\n", part.ContextType, part.Name, part.FilePath, part.NumTokens)
				// }

				modelContext = res
			}

			errCh <- nil
		}()

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in getPlanConvo: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("error getting plan convo: %v", r)
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()

			res, err := db.GetPlanConvo(currentOrgId, planId)
			if err != nil {
				log.Printf("Error getting plan convo: %v\n", err)
				errCh <- fmt.Errorf("error getting plan convo: %v", err)
				return
			}
			convo = res
			UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
				ap.MessageNum = len(convo)
			})

			promptTokens := shared.GetNumTokensEstimate(req.Prompt)
			innerErrCh := make(chan error, 2)

			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("panic in storeUserMessage: %v\n%s", r, debug.Stack())
						innerErrCh <- fmt.Errorf("error storing user message: %v", r)
						runtime.Goexit() // don't allow outer function to continue and double-send to channel
					}
				}()

				if iteration == 0 && missingFileResponse == "" && !req.IsUserContinue {
					num := len(convo) + 1

					log.Printf("[TellLoad] storing user message | len(convo): %d | num: %d\n", len(convo), num)

					promptMsg = &db.ConvoMessage{
						OrgId:   currentOrgId,
						PlanId:  planId,
						UserId:  currentUserId,
						Role:    openai.ChatMessageRoleUser,
						Tokens:  promptTokens,
						Num:     num,
						Message: req.Prompt,
						Flags: shared.ConvoMessageFlags{
							IsApplyDebug: req.IsApplyDebug,
							IsUserDebug:  req.IsUserDebug,
							IsChat:       req.IsChatOnly,
						},
					}

					log.Println("[TellLoad] storing user message")
					// repo.LogGitRepoState()

					_, err = db.StoreConvoMessage(repo, promptMsg, auth.User.Id, branch, true)

					if err != nil {
						log.Printf("[TellLoad] Error storing user message: %v\n", err)
						innerErrCh <- fmt.Errorf("error storing user message: %v", err)
						return
					}

					UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
						ap.MessageNum = num
					})
				}

				innerErrCh <- nil
			}()

			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("panic in getPlanSummaries: %v\n%s", r, debug.Stack())
						innerErrCh <- fmt.Errorf("error getting plan summaries: %v", r)
						runtime.Goexit() // don't allow outer function to continue and double-send to channel
					}
				}()

				var convoMessageIds []string

				for _, convoMessage := range convo {
					convoMessageIds = append(convoMessageIds, convoMessage.Id)
				}

				log.Println("getting plan summaries")
				log.Println("convoMessageIds:", convoMessageIds)

				res, err := db.GetPlanSummaries(planId, convoMessageIds)
				if err != nil {
					log.Printf("Error getting plan summaries: %v\n", err)
					innerErrCh <- fmt.Errorf("error getting plan summaries: %v", err)
					return
				}
				summaries = res

				log.Printf("got %d plan summaries", len(summaries))

				if len(summaries) > 0 {
					latestSummaryTokens = shared.GetNumTokensEstimate(summaries[len(summaries)-1].Summary)
				}

				innerErrCh <- nil
			}()

			for i := 0; i < 2; i++ {
				err := <-innerErrCh
				if err != nil {
					errCh <- err
					return
				}
			}

			if promptMsg != nil {
				convo = append(convo, promptMsg)
			}

			errCh <- nil
		}()

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in getPlanSubtasks: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("error getting plan subtasks: %v", r)
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()

			res, err := db.GetPlanSubtasks(auth.OrgId, planId)
			if err != nil {
				log.Printf("Error getting plan subtasks: %v\n", err)
				errCh <- fmt.Errorf("error getting plan subtasks: %v", err)
				return
			}
			subtasks = res
			errCh <- nil
		}()

		for i := 0; i < 4; i++ {
			err = <-errCh
			if err != nil {
				go notify.NotifyErr(notify.SeverityError, fmt.Errorf("error loading plan: %v", err))

				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    fmt.Sprintf("Error loading plan: %v", err),
				}
				return err
			}
		}

		res, err := db.GetCurrentPlanState(db.CurrentPlanStateParams{
			OrgId:    currentOrgId,
			PlanId:   planId,
			Contexts: modelContext,
		})

		if err != nil {
			return fmt.Errorf("error getting current plan state: %v", err)
		}

		currentPlan = res

		return nil
	})

	if err != nil {
		log.Printf("execTellPlan: error loading tell plan: %v\n", err)
		go notify.NotifyErr(notify.SeverityError, fmt.Errorf("error loading tell plan: %v", err))

		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error loading tell plan",
		}
		return err
	}

	state.modelContext = modelContext
	state.convo = convo
	state.promptConvoMessage = promptMsg
	state.summaries = summaries
	state.latestSummaryTokens = latestSummaryTokens
	state.settings = settings
	state.currentPlanState = currentPlan
	state.subtasks = subtasks

	for _, subtask := range state.subtasks {
		if !subtask.IsFinished {
			state.currentSubtask = subtask
			break
		}
	}

	log.Printf("[TellLoad] Subtasks: %+v", state.subtasks)
	log.Printf("[TellLoad] Current subtask: %+v", state.currentSubtask)

	state.hasContextMap = false
	state.contextMapEmpty = true
	for _, context := range state.modelContext {
		if context.ContextType == shared.ContextMapType {
			state.hasContextMap = true
			if context.NumTokens > 0 {
				state.contextMapEmpty = false
			}
			break
		}
	}

	state.hasAssistantReply = false
	for _, convoMessage := range state.convo {
		if convoMessage.Role == openai.ChatMessageRoleAssistant {
			state.hasAssistantReply = true
			break
		}
	}

	if iteration == 0 && missingFileResponse == "" {
		UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
			ap.Contexts = state.modelContext

			for _, context := range state.modelContext {
				if context.FilePath != "" {
					ap.ContextsByPath[context.FilePath] = context
				}
			}
		})
	} else if missingFileResponse == "" {
		// reset current reply content and num tokens
		UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
			ap.CurrentReplyContent = ""
			ap.NumTokens = 0
		})
	}

	// if any skipped paths have since been added to context, remove them from skipped paths
	if len(active.SkippedPaths) > 0 {
		var toUnskipPaths []string
		for contextPath := range active.ContextsByPath {
			if active.SkippedPaths[contextPath] {
				toUnskipPaths = append(toUnskipPaths, contextPath)
			}
		}
		if len(toUnskipPaths) > 0 {
			UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
				for _, path := range toUnskipPaths {
					delete(ap.SkippedPaths, path)
				}
			})
		}
	}

	return nil
}

func (state *activeTellStreamState) setActivePlan() error {
	plan := state.plan
	branch := state.branch

	active := GetActivePlan(plan.Id, branch)

	if active == nil {
		return fmt.Errorf("no active plan with id %s", plan.Id)
	}

	state.activePlan = active

	return nil
}
