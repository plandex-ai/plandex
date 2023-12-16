package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"plandex-server/db"

	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
)

func ListLogsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListLogsHandler")

	// TODO: get from auth when implemented
	currentOrgId := "org1"

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	body, err := db.GetGitCommitHistory(currentOrgId, planId)

	if err != nil {
		log.Println("Error getting logs: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res := shared.LogResponse{
		Body: body,
	}

	bytes, err := json.Marshal(res)

	if err != nil {
		log.Println("Error marshalling logs: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(bytes)

	log.Println("Successfully processed request for ListLogsHandler")
}

func RewindPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for RewindPlanHandler")

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

}
