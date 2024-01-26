package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"plandex-server/db"
	model "plandex-server/model/plan"
	"plandex-server/types"

	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
)

const TrialMaxReplies = 10

func TellPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for TellPlanHandler")

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]

	log.Println("planId: ", planId)

	plan := authorizePlanUpdate(w, planId, auth)

	if plan == nil {
		return
	}

	if plan.OwnerId != auth.User.Id && !auth.HasPermission(types.PermissionUpdateAnyPlan) {
		log.Println("User does not have permission to update plan")
		http.Error(w, "User does not have permission to update plan", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer func() {
		log.Println("Closing request body")
		r.Body.Close()
	}()

	var requestBody shared.TellPlanRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	if os.Getenv("IS_CLOUD") != "" {
		user, err := db.GetUser(auth.User.Id)

		if err != nil {
			log.Printf("Error getting user: %v\n", err)
			http.Error(w, "Error getting user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if user.IsTrial {
			if plan.TotalReplies >= types.TrialMaxReplies {
				writeApiError(w, shared.ApiError{
					Type:   shared.ApiErrorTypeTrialMessagesExceeded,
					Status: http.StatusForbidden,
					Msg:    "Free trial message limit exceeded",
					TrialMessagesExceededError: &shared.TrialMessagesExceededError{
						MaxReplies: types.TrialMaxReplies,
					},
				})
				return
			}
		}
	}

	err = model.Tell(plan, branch, auth, &requestBody)

	if err != nil {
		log.Printf("Error telling plan: %v\n", err)
		http.Error(w, "Error telling plan", http.StatusInternalServerError)
		return
	}

	if requestBody.ConnectStream {
		active := model.Active.Get(planId)
		subscriptionId, ch := model.SubscribePlan(planId)

		startResponseStream(w, ch, active.Ctx, func() {
			model.UnsubscribePlan(planId, subscriptionId)
		})
	}

}

func BuildPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for BuildPlanHandler")
	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	if authorizePlan(w, planId, auth) == nil {
		return
	}

}

func ConnectPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ConnectPlanHandler")
	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	if authorizePlan(w, planId, auth) == nil {
		return
	}

}

func StopPlanHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: this handler should lookup the IP of the active plan stream and forward the request if it's not the same as this server's IP

	log.Println("Received request for StopPlanHandler")
	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]

	log.Println("planId: ", planId)

	if authorizePlan(w, planId, auth) == nil {
		return
	}

	unlockFn := lockRepo(w, r, auth, db.LockScopeWrite)
	if unlockFn == nil {
		return
	} else {
		defer (*unlockFn)()
	}

	err := model.Stop(planId, branch, auth.User.Id, auth.OrgId)

	if err != nil {
		log.Printf("Error stopping plan: %v\n", err)
		http.Error(w, "Error stopping plan", http.StatusInternalServerError)
		return
	}

	log.Println("Successfully processed request for StopPlanHandler")
}
