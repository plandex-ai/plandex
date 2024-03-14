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

	convoMessage, err := db.GetPlanConvo(auth.OrgId, planId)

	if err != nil {
		log.Println("Error getting plan convo: ", err)
		http.Error(w, "Error getting plan convo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(convoMessage)

	if err != nil {
		log.Println("Error marshalling plan convo: ", err)
		http.Error(w, "Error marshalling plan convo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully processed request for ListConvoHandler")
	w.Write(bytes)

}
