package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func ListLogsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListLogsHandler")

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

}

func RewindPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for RewindPlanHandler")

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

}
