package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"plandex-server/model/plan"

	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
)

func TellPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for TellPlanHandler")

	// TODO: get this from auth when implemented
	currentOrgId := "2ff5bc12-1160-4305-8707-9a165319de5a"
	currentUserId := "bc9c75ee-57b0-4552-aa1b-f80cf8c09f3f"

	// TODO: authenticate user and plan access

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var requestBody shared.TellPlanRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	err = plan.Tell(planId, currentUserId, currentOrgId, &requestBody)

	if err != nil {
		log.Printf("Error telling plan: %v\n", err)
		http.Error(w, "Error telling plan", http.StatusInternalServerError)
		return
	}

	if requestBody.ConnectStream {
		active := plan.Active.Get(planId)
		subscriptionId, ch := plan.SubscribePlan(planId)

		startResponseStream(w, ch, active.Ctx, func() {
			plan.UnsubscribePlan(planId, subscriptionId)
		})
	}

}

func BuildPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for BuildPlanHandler")

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

}

func ConnectPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ConnectPlanHandler")

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

}

func StopPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for StopPlanHandler")

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

}
