package plan

import (
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/notify"
	"plandex-server/syntax"
	"plandex-server/types"
	"runtime"
	"runtime/debug"

	shared "plandex-shared"
)

func (state *activeBuildStreamState) loadPendingBuilds(sessionId string) (map[string][]*types.ActiveBuild, error) {
	clients := state.clients
	plan := state.plan
	branch := state.branch
	auth := state.auth

	active, err := activatePlan(clients, plan, branch, auth, "", true, false, sessionId)

	if err != nil {
		log.Printf("Error activating plan: %v\n", err)
	}

	modelStreamId := active.ModelStreamId
	state.modelStreamId = modelStreamId

	var modelContext []*db.Context
	var pendingBuildsByPath map[string][]*types.ActiveBuild
	var settings *shared.PlanSettings

	err = db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    auth.OrgId,
		UserId:   auth.User.Id,
		PlanId:   plan.Id,
		Branch:   branch,
		Scope:    db.LockScopeRead,
		Ctx:      active.Ctx,
		CancelFn: active.CancelFn,
		Reason:   "load pending builds",
	}, func(repo *db.GitRepo) error {
		errCh := make(chan error, 3)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in getPlanContexts: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("error getting plan modelContext: %v", r)
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			res, err := db.GetPlanContexts(auth.OrgId, plan.Id, true, false)
			if err != nil {
				log.Printf("Error getting plan modelContext: %v\n", err)
				errCh <- fmt.Errorf("error getting plan modelContext: %v", err)
				return
			}
			modelContext = res

			errCh <- nil
		}()

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in getPlanSettings: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("error getting plan settings: %v", r)
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			res, err := active.PendingBuildsByPath(auth.OrgId, auth.User.Id, nil)

			if err != nil {
				log.Printf("Error getting pending builds by path: %v\n", err)
				errCh <- fmt.Errorf("error getting pending builds by path: %v", err)
				return
			}

			pendingBuildsByPath = res

			errCh <- nil
		}()

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
			errCh <- nil
		}()

		for i := 0; i < 3; i++ {
			err = <-errCh
			if err != nil {
				log.Printf("Error getting plan data: %v\n", err)
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error getting plan data: %v", err)
	}

	UpdateActivePlan(plan.Id, branch, func(ap *types.ActivePlan) {
		ap.Contexts = modelContext
		for _, context := range modelContext {
			if context.FilePath != "" {
				ap.ContextsByPath[context.FilePath] = context
			}
		}
	})

	state.modelContext = modelContext
	state.settings = settings

	return pendingBuildsByPath, nil
}

func (state *activeBuildStreamFileState) loadBuildFile(activeBuild *types.ActiveBuild) error {
	currentOrgId := state.currentOrgId
	planId := state.plan.Id
	branch := state.branch
	filePath := state.filePath

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		return fmt.Errorf("active plan not found")
	}

	convoMessageId := activeBuild.ReplyId

	parser, lang, fallbackParser, fallbackLang := syntax.GetParserForPath(filePath)

	if parser != nil {
		validationRes, err := syntax.ValidateWithParsers(activePlan.Ctx, lang, parser, fallbackLang, fallbackParser, state.preBuildState)
		if err != nil {
			log.Printf(" error validating original file syntax: %v\n", err)
			return fmt.Errorf("error validating original file syntax: %v", err)
		}

		state.language = validationRes.Lang
		state.parser = validationRes.Parser

		state.builderRun.Lang = string(validationRes.Lang)

		if validationRes.TimedOut {
			state.syntaxCheckTimedOut = true
		} else if !validationRes.Valid {
			state.preBuildStateSyntaxInvalid = true
		}
	}

	build := &db.PlanBuild{
		OrgId:          currentOrgId,
		PlanId:         planId,
		ConvoMessageId: convoMessageId,
		FilePath:       filePath,
	}
	err := db.StorePlanBuild(build)

	if err != nil {
		log.Printf("Error storing plan build: %v\n", err)
		UpdateActivePlan(activePlan.Id, activePlan.Branch, func(ap *types.ActivePlan) {
			ap.IsBuildingByPath[filePath] = false
		})
		go notify.NotifyErr(notify.SeverityError, fmt.Errorf("error storing plan build: %v", err))

		activePlan.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error storing plan build: " + err.Error(),
		}
		return err
	}

	var currentPlan *shared.CurrentPlanState
	var convo []*db.ConvoMessage

	log.Println("Locking repo for load build file")

	err = db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:       currentOrgId,
		UserId:      state.activeBuildStreamState.currentUserId,
		PlanId:      planId,
		Branch:      branch,
		PlanBuildId: build.Id,
		Scope:       db.LockScopeRead,
		Ctx:         activePlan.Ctx,
		CancelFn:    activePlan.CancelFn,
		Reason:      "load build file",
	}, func(repo *db.GitRepo) error {
		errCh := make(chan error, 2)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in getCurrentPlanState: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("error getting current plan state: %v", r)
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			log.Println("loadBuildFile - Getting current plan state")
			res, err := db.GetCurrentPlanState(db.CurrentPlanStateParams{
				OrgId:  currentOrgId,
				PlanId: planId,
			})
			if err != nil {
				log.Printf("Error getting current plan state: %v\n", err)
				UpdateActivePlan(activePlan.Id, activePlan.Branch, func(ap *types.ActivePlan) {
					ap.IsBuildingByPath[filePath] = false
				})
				go notify.NotifyErr(notify.SeverityError, fmt.Errorf("error getting current plan state: %v", err))

				activePlan.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Error getting current plan state: " + err.Error(),
				}
				errCh <- fmt.Errorf("error getting current plan state: %v", err)
				return
			}
			currentPlan = res

			log.Println("Got current plan state")
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

			errCh <- nil
		}()

		for i := 0; i < 2; i++ {
			err = <-errCh
			if err != nil {
				log.Printf("Error getting plan data: %v\n", err)
				return err
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("Error loading build file: %v\n", err)
		UpdateActivePlan(activePlan.Id, activePlan.Branch, func(ap *types.ActivePlan) {
			ap.IsBuildingByPath[filePath] = false
		})
		go notify.NotifyErr(notify.SeverityError, fmt.Errorf("error loading build file: %v", err))

		activePlan.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error loading build file: " + err.Error(),
		}
		return err
	}

	state.filePath = filePath
	state.convoMessageId = convoMessageId
	state.build = build
	state.currentPlanState = currentPlan
	state.convo = convo

	return nil

}
