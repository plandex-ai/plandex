package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"plandex-server/db"
	model "plandex-server/model/plan"

	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
)

const TrialMaxMessages = 15

func TellPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for TellPlanHandler")

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}
	currentUserId := auth.User.Id

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	plan := authorizePlan(w, planId, currentUserId, auth.OrgId)

	if plan == nil {
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
		user, err := db.GetUser(currentUserId)

		if err != nil {
			log.Printf("Error getting user: %v\n", err)
			http.Error(w, "Error getting user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if user.IsTrial {
			if plan.TotalMessages >= TrialMaxMessages {
				writeApiError(w, shared.ApiErr{
					Type:   shared.ApiErrorTypeTrialMessagesExceeded,
					Status: http.StatusForbidden,
					Msg:    "Free trial message limit exceeded",
					TrialMessagesExceededError: &shared.TrialMessagesExceededError{
						MaxMessages: TrialMaxMessages,
					},
				})
				return
			}
		}
	}

	err = model.Tell(plan, currentUserId, auth.OrgId, &requestBody)

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
	currentUserId := auth.User.Id

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	if authorizePlan(w, planId, currentUserId, auth.OrgId) == nil {
		return
	}

}

func ConnectPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ConnectPlanHandler")
	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}
	currentUserId := auth.User.Id

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	if authorizePlan(w, planId, currentUserId, auth.OrgId) == nil {
		return
	}

}

func StopPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for StopPlanHandler")
	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}
	currentUserId := auth.User.Id

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	if authorizePlan(w, planId, currentUserId, auth.OrgId) == nil {
		return
	}

}
