package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"plandex-server/db"

	"github.com/gorilla/mux"
)

func ListConvoHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for ListConvoHandler")
	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	if authorizePlan(w, planId, auth) == nil {
		return
	}

	var err error
	ctx, cancel := context.WithCancel(context.Background())
	unlockFn := lockRepo(w, r, auth, db.LockScopeRead, ctx, cancel, true)
	if unlockFn == nil {
		return
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	convoMessages, err := db.GetPlanConvo(auth.OrgId, planId)

	if err != nil {
		log.Println("Error getting plan convo: ", err)
		http.Error(w, "Error getting plan convo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(convoMessages)

	if err != nil {
		log.Println("Error marshalling plan convo: ", err)
		http.Error(w, "Error marshalling plan convo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully processed request for ListConvoHandler")
	w.Write(bytes)

}

func GetPlanStatusHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for GetPlanStatusHandler")

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]

	log.Println("planId: ", planId, "branch: ", branch)

	plan := authorizePlan(w, planId, auth)
	if plan == nil {
		return
	}

	var err error
	ctx, cancel := context.WithCancel(context.Background())
	unlockFn := lockRepo(w, r, auth, db.LockScopeRead, ctx, cancel, true)
	if unlockFn == nil {
		return
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	convoMessages, err := db.GetPlanConvo(auth.OrgId, planId)

	if err != nil {
		log.Println("Error getting plan convo: ", err)
		http.Error(w, "Error getting plan convo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(convoMessages) == 0 {
		log.Println("No messages found for plan")
		return
	}

	convoMessageIds := make([]string, len(convoMessages))
	for i, convoMessage := range convoMessages {
		convoMessageIds[i] = convoMessage.Id
	}

	summmaries, err := db.GetPlanSummaries(planId, convoMessageIds)

	if err != nil {
		log.Println("Error getting plan summaries: ", err)
		http.Error(w, "Error getting plan summaries: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(summmaries) == 0 {
		log.Println("No summaries found for plan")
		return
	}

	latestSummary := summmaries[len(summmaries)-1]

	bytes := []byte(latestSummary.Summary)

	w.Write(bytes)

	log.Println("Successfully processed request for GetPlanStatusHandler")
}
