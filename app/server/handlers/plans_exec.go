package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"plandex-server/db"
	"plandex-server/host"
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
		active := model.GetActivePlan(planId, branch)
		subscriptionId, ch := model.SubscribePlan(planId, branch)
		startResponseStream(w, ch, active, func() {
			model.UnsubscribePlan(planId, branch, subscriptionId)
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

	active := model.GetActivePlan(planId, branch)

	if active == nil {
		modelStream, err := db.GetActiveModelStream(planId, branch)

		if err != nil {
			log.Printf("Error getting active model stream: %v\n", err)
			http.Error(w, "Error getting active model stream", http.StatusInternalServerError)
			return
		}

		if modelStream == nil {
			log.Printf("No active model stream for plan %s\n", planId)
			http.Error(w, "No active model stream for plan", http.StatusNotFound)
			return
		}

		if modelStream.InternalIp == host.Ip {
			db.SetModelStreamFinished(modelStream.Id)
			log.Printf("No active plan for plan %s\n", planId)
			http.Error(w, "No active plan for plan", http.StatusNotFound)
			return
		} else {
			log.Printf("Forwarding request to %s\n", modelStream.InternalIp)
			proxyUrl := fmt.Sprintf("http://%s:%s/plans/%s/%s/stop", modelStream.InternalIp, os.Getenv("EXTERNAL_PORT"), planId, branch)
			proxyRequest(w, r, proxyUrl)
			return
		}
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
