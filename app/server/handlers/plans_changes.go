package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"plandex-server/db"
	"time"

	"github.com/gorilla/mux"
)

func CurrentPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CurrentPlanHandler")

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
	unlockFn := lockRepo(w, r, auth, db.LockScopeRead, ctx, cancel)
	if unlockFn == nil {
		return
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	planState, err := db.GetCurrentPlanState(db.CurrentPlanStateParams{
		OrgId:  auth.OrgId,
		PlanId: planId,
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

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]
	log.Println("planId: ", planId)

	plan := authorizePlan(w, planId, auth)
	if plan == nil {
		return
	}

	var err error
	ctx, cancel := context.WithCancel(context.Background())
	unlockFn := lockRepo(w, r, auth, db.LockScopeWrite, ctx, cancel)
	if unlockFn == nil {
		return
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	err = db.ApplyPlan(auth.OrgId, auth.User.Id, branch, plan)

	if err != nil {
		log.Printf("Error applying plan: %v\n", err)
		http.Error(w, "Error applying plan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully applied plan", planId)
}

func RejectAllChangesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for RejectAllChangesHandler")

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
	unlockFn := lockRepo(w, r, auth, db.LockScopeWrite, ctx, cancel)
	if unlockFn == nil {
		return
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	err = db.RejectAllResults(auth.OrgId, planId)

	if err != nil {
		log.Printf("Error rejecting all changes: %v\n", err)
		http.Error(w, "Error rejecting all changes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully rejected all changes for plan", planId)
}

func RejectResultHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for RejectResultHandler")

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	resultId := vars["resultId"]

	log.Println("planId: ", planId, "resultId: ", resultId)

	if authorizePlan(w, planId, auth) == nil {
		return
	}

	var err error
	ctx, cancel := context.WithCancel(context.Background())
	unlockFn := lockRepo(w, r, auth, db.LockScopeWrite, ctx, cancel)
	if unlockFn == nil {
		return
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	err = db.RejectPlanFileResult(auth.OrgId, planId, resultId, time.Now())

	if err != nil {
		log.Printf("Error rejecting result: %v\n", err)
		http.Error(w, "Error rejecting result: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully rejected plan result", resultId)
}

func RejectReplacementHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for RejectReplacementHandler")

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	resultId := vars["resultId"]
	replacementId := vars["replacementId"]

	log.Println("planId: ", planId, "resultId: ", resultId, "replacementId: ", replacementId)

	if authorizePlan(w, planId, auth) == nil {
		return
	}

	var err error
	ctx, cancel := context.WithCancel(context.Background())
	unlockFn := lockRepo(w, r, auth, db.LockScopeWrite, ctx, cancel)
	if unlockFn == nil {
		return
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	err = db.RejectReplacement(auth.OrgId, planId, resultId, replacementId)

	if err != nil {
		log.Printf("Error rejecting replacement: %v\n", err)
		http.Error(w, "Error rejecting replacement: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully rejected replacement", replacementId)
}

func ArchivePlanHandler(w http.ResponseWriter, r *http.Request) {
	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	log.Println("Received request for ArchivePlanHandler")

	vars := mux.Vars(r)
	planId := vars["planId"]
	log.Println("planId: ", planId)

	plan := authorizePlanArchive(w, planId, auth)

	if plan == nil {
		return
	}

	if plan.ArchivedAt != nil {
		log.Println("Plan already archived")
		http.Error(w, "Plan already archived", http.StatusBadRequest)
		return
	}

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
