package plan

import (
	"fmt"
	"log"
	"net/http"
	"plandex-server/notify"
	"runtime/debug"

	shared "plandex-shared"
)

func (state *activeTellStreamState) queuePendingBuilds() {
	plan := state.plan
	planId := plan.Id
	branch := state.branch
	auth := state.auth
	clients := state.clients
	authVars := state.authVars
	currentOrgId := state.currentOrgId
	currentUserId := state.currentUserId
	active := GetActivePlan(planId, branch)

	if active == nil {
		log.Printf("execTellPlan: Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic in queuePendingBuilds: %v\n%s", r, debug.Stack())
			go notify.NotifyErr(notify.SeverityError, fmt.Errorf("error getting pending builds by path: %v", r))
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    fmt.Sprintf("Error getting pending builds by path: %v\n%s", r, debug.Stack()),
			}
		}
	}()

	pendingBuildsByPath, err := active.PendingBuildsByPath(auth.OrgId, auth.User.Id, state.convo)

	if err != nil {
		log.Printf("Error getting pending builds by path: %v\n", err)
		go notify.NotifyErr(notify.SeverityError, fmt.Errorf("error getting pending builds by path: %v", err))

		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    fmt.Sprintf("Error getting pending builds by path: %v", err),
		}
		return
	}

	if len(pendingBuildsByPath) == 0 {
		log.Println("Tell plan: no pending builds")
		return
	}

	log.Printf("Tell plan: found %d pending builds\n", len(pendingBuildsByPath))
	// spew.Dump(pendingBuildsByPath)

	buildState := &activeBuildStreamState{
		modelStreamId: active.ModelStreamId,
		clients:       clients,
		authVars:      authVars,
		auth:          auth,
		currentOrgId:  currentOrgId,
		currentUserId: currentUserId,
		plan:          plan,
		branch:        branch,
		settings:      state.settings,
		modelContext:  state.modelContext,
		orgUserConfig: state.orgUserConfig,
	}

	for _, pendingBuilds := range pendingBuildsByPath {
		buildState.queueBuilds(pendingBuilds)
	}
}
