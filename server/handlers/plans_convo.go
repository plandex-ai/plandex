package handlers

import (
	"log"
	"net/http"

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

}
