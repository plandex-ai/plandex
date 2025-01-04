package plan

import (
	"log"
	"net/http"

	"github.com/plandex/plandex/shared"
)

func (state *activeTellStreamState) queuePendingBuilds() {
	plan := state.plan
	planId := plan.Id
	branch := state.branch
	auth := state.auth
	clients := state.clients
	currentOrgId := state.currentOrgId
	currentUserId := state.currentUserId
	active := GetActivePlan(planId, branch)

	if active == nil {
		log.Printf("execTellPlan: Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return
	}

	pendingBuildsByPath, err := active.PendingBuildsByPath(auth.OrgId, auth.User.Id, state.convo)

	if err != nil {
		log.Printf("Error getting pending builds by path: %v\n", err)
		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error getting pending builds by path",
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
		tellState:     state,
		clients:       clients,
		auth:          auth,
		currentOrgId:  currentOrgId,
		currentUserId: currentUserId,
		plan:          plan,
		branch:        branch,
		settings:      state.settings,
		modelContext:  state.modelContext,
	}

	for _, pendingBuilds := range pendingBuildsByPath {
		buildState.queueBuilds(pendingBuilds)
	}
}
