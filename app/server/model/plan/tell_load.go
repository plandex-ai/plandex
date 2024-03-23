package plan

import (
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/model"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) loadTellPlan() error {
	client := state.client
	req := state.req
	auth := state.auth
	plan := state.plan
	planId := plan.Id
	branch := state.branch
	currentUserId := state.currentUserId
	currentOrgId := state.currentOrgId
	iteration := state.iteration
	missingFileResponse := state.missingFileResponse

	active := GetActivePlan(plan.Id, branch)

	lockScope := db.LockScopeWrite
	if iteration > 0 || missingFileResponse != "" {
		lockScope = db.LockScopeRead
	}
	repoLockId, err := db.LockRepo(
		db.LockRepoParams{
			OrgId:    auth.OrgId,
			UserId:   auth.User.Id,
			PlanId:   planId,
			Branch:   branch,
			Scope:    lockScope,
			Ctx:      active.Ctx,
			CancelFn: active.CancelFn,
		},
	)

	if err != nil {
		log.Printf("execTellPlan: Error locking repo for plan ID %s on branch %s: %v\n", plan.Id, branch, err)
		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error locking repo",
		}
		return err
	}

	errCh := make(chan error)
	var modelContext []*db.Context
	var convo []*db.ConvoMessage
	var summaries []*db.ConvoSummary
	var settings *shared.PlanSettings

	// get name for plan and rename it's a draft
	go func() {
		res, err := db.GetPlanSettings(plan, true)
		if err != nil {
			log.Printf("Error getting plan settings: %v\n", err)
			errCh <- fmt.Errorf("error getting plan settings: %v", err)
			return
		}
		settings = res

		if plan.Name == "draft" {
			name, err := model.GenPlanName(client, settings.ModelSet.Namer, req.Prompt)

			if err != nil {
				log.Printf("Error generating plan name: %v\n", err)
				errCh <- fmt.Errorf("error generating plan name: %v", err)
				return
			}

			tx, err := db.Conn.Begin()
			if err != nil {
				log.Printf("Error starting transaction: %v\n", err)
				errCh <- fmt.Errorf("error starting transaction: %v", err)
			}

			// Ensure that rollback is attempted in case of failure
			defer func() {
				if err != nil {
					if rbErr := tx.Rollback(); rbErr != nil {
						log.Printf("transaction rollback error: %v\n", rbErr)
					} else {
						log.Println("transaction rolled back")
					}
				}
			}()

			err = db.RenamePlan(planId, name, tx)

			if err != nil {
				log.Printf("Error renaming plan: %v\n", err)
				errCh <- fmt.Errorf("error renaming plan: %v", err)
				return
			}

			err = db.IncNumNonDraftPlans(currentUserId, tx)

			if err != nil {
				log.Printf("Error incrementing num non draft plans: %v\n", err)
				errCh <- fmt.Errorf("error incrementing num non draft plans: %v", err)
				return
			}

			err = tx.Commit()
			if err != nil {
				log.Printf("Error committing transaction: %v\n", err)
				errCh <- fmt.Errorf("error committing transaction: %v", err)
				return
			}
		}

		errCh <- nil
	}()

	go func() {
		if iteration > 0 || missingFileResponse != "" {
			modelContext = active.Contexts
		} else {
			res, err := db.GetPlanContexts(currentOrgId, planId, true)
			if err != nil {
				log.Printf("Error getting plan modelContext: %v\n", err)
				errCh <- fmt.Errorf("error getting plan modelContext: %v", err)
				return
			}
			modelContext = res
		}
		errCh <- nil
	}()

	go func() {
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

		promptTokens, err := shared.GetNumTokens(req.Prompt)
		if err != nil {
			log.Printf("Error getting prompt num tokens: %v\n", err)
			errCh <- fmt.Errorf("error getting prompt num tokens: %v", err)
			return
		}

		innerErrCh := make(chan error)
		var userMsg *db.ConvoMessage

		go func() {
			if iteration == 0 && missingFileResponse == "" && !req.IsUserContinue {
				num := len(convo) + 1

				log.Printf("storing user message | len(convo): %d | num: %d\n", len(convo), num)

				userMsg = &db.ConvoMessage{
					OrgId:   currentOrgId,
					PlanId:  planId,
					UserId:  currentUserId,
					Role:    openai.ChatMessageRoleUser,
					Tokens:  promptTokens,
					Num:     num,
					Message: req.Prompt,
				}

				_, err = db.StoreConvoMessage(userMsg, auth.User.Id, branch, true)

				if err != nil {
					log.Printf("Error storing user message: %v\n", err)
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

			innerErrCh <- nil
		}()

		for i := 0; i < 2; i++ {
			err := <-innerErrCh
			if err != nil {
				errCh <- err
				return
			}
		}

		if userMsg != nil {
			convo = append(convo, userMsg)
		}

		errCh <- nil
	}()

	err = func() error {
		var err error
		defer func() {
			if err != nil {
				log.Printf("Error: %v\n", err)
				err = db.GitClearUncommittedChanges(auth.OrgId, planId)
				if err != nil {
					log.Printf("Error clearing uncommitted changes: %v\n", err)
				}
			}

			err = db.UnlockRepo(repoLockId)
			if err != nil {
				log.Printf("Error unlocking repo: %v\n", err)
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Error unlocking repo",
				}
				return
			}
		}()

		for i := 0; i < 3; i++ {
			err = <-errCh
			if err != nil {
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Error getting plan, context, convo, or summaries",
				}
				return err
			}
		}

		return nil
	}()

	if err != nil {
		return err
	}

	state.modelContext = modelContext
	state.convo = convo
	state.summaries = summaries
	state.settings = settings

	return nil
}
