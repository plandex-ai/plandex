package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"plandex-server/db"
	"time"

	"github.com/gorilla/mux"
)

func CurrentPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListPlanChangesHandler")

	// TODO: get this from auth when implemented
	currentOrgId := "2ff5bc12-1160-4305-8707-9a165319de5a"

	// TODO: authenticate user and plan access

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	contexts, err := db.GetPlanContexts(currentOrgId, planId, true)

	if err != nil {
		log.Printf("Error getting plan contexts: %v\n", err)
		http.Error(w, "Error getting plan contexts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	planState, err := db.GetCurrentPlanState(db.CurrentPlanStateParams{
		OrgId:    currentOrgId,
		PlanId:   planId,
		Contexts: contexts,
	})

	if err != nil {
		log.Printf("Error getting current plan state: %v\n", err)
		http.Error(w, "Error getting current plan state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonBytes, err := json.Marshal(planState)

	if err != nil {
		log.Printf("Error marshalling plan state: %v\n", err)
		http.Error(w, "Error marshalling plan state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully retrieved current plan state")

	w.Write(jsonBytes)
}

func ApplyPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ApplyPlanHandler")

	// TODO: get this from auth when implemented
	currentOrgId := "2ff5bc12-1160-4305-8707-9a165319de5a"

	// TODO: authenticate user and plan access

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	err := db.ApplyPlan(currentOrgId, planId)

	if err != nil {
		log.Printf("Error applying plan: %v\n", err)
		http.Error(w, "Error applying plan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully applied plan", planId)
}

func RejectAllChangesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for RejectAllChangesHandler")

	// TODO: get this from auth when implemented
	currentOrgId := "2ff5bc12-1160-4305-8707-9a165319de5a"

	// TODO: authenticate user and plan access

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	err := db.RejectAllResults(currentOrgId, planId)

	if err != nil {
		log.Printf("Error rejecting all changes: %v\n", err)
		http.Error(w, "Error rejecting all changes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully rejected all changes for plan", planId)
}

func RejectResultHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for RejectResultHandler")

	// TODO: get this from auth when implemented
	currentOrgId := "2ff5bc12-1160-4305-8707-9a165319de5a"

	// TODO: authenticate user and plan access

	vars := mux.Vars(r)
	planId := vars["planId"]
	resultId := vars["resultId"]

	log.Println("planId: ", planId, "resultId: ", resultId)

	err := db.RejectPlanFileResult(currentOrgId, planId, resultId, time.Now())

	if err != nil {
		log.Printf("Error rejecting result: %v\n", err)
		http.Error(w, "Error rejecting result: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully rejected plan result", resultId)
}

func RejectReplacementHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for RejectReplacementHandler")

	// TODO: get this from auth when implemented
	currentOrgId := "2ff5bc12-1160-4305-8707-9a165319de5a"

	// TODO: authenticate user and plan access

	vars := mux.Vars(r)
	planId := vars["planId"]
	resultId := vars["resultId"]
	replacementId := vars["replacementId"]

	log.Println("planId: ", planId, "resultId: ", resultId, "replacementId: ", replacementId)

	err := db.RejectReplacement(currentOrgId, planId, resultId, replacementId)

	if err != nil {
		log.Printf("Error rejecting replacement: %v\n", err)
		http.Error(w, "Error rejecting replacement: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully rejected replacement", replacementId)
}

func ArchivePlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ArchivePlanHandler")

	// TODO: authenticate user and plan access

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	res, err := db.Conn.Exec("UPDATE plans SET archived_at = NOW() WHERE id = $1", planId)

	if err != nil {
		log.Printf("Error archiving plan: %v\n", err)
		http.Error(w, "Error archiving plan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v\n", err)
		http.Error(w, "Error getting rows affected: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		log.Println("Plan not found")
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	log.Println("Successfully archived plan", planId)
}
