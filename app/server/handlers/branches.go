package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"plandex-server/db"

	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
)

func ListBranchesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListBranchesHandler")

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
	unlockFn := lockRepo(w, r, auth, db.LockScopeRead, ctx, cancel, false)
	if unlockFn == nil {
		return
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	branches, err := db.ListPlanBranches(auth.OrgId, planId)

	if err != nil {
		log.Printf("Error getting branches: %v\n", err)
		http.Error(w, "Error getting branches: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonBytes, err := json.Marshal(branches)

	if err != nil {
		log.Printf("Error marshalling branches: %v\n", err)
		http.Error(w, "Error marshalling branches: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully retrieved branches")

	w.Write(jsonBytes)
}

func CreateBranchHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreateBranchHandler")

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

	var req shared.CreateBranchRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body ", http.StatusBadRequest)
		return
	}

	parentBranch, err := db.GetDbBranch(planId, branch)

	if err != nil {
		log.Printf("Error getting parent branch: %v\n", err)
		http.Error(w, "Error getting parent branch: "+err.Error(), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	unlockFn := lockRepo(w, r, auth, db.LockScopeWrite, ctx, cancel, true)
	if unlockFn == nil {
		return
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v\n", err)
		http.Error(w, "Error starting transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Ensure that rollback is attempted in case of failure
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("transaction rollback error: %v\n", rbErr)
			} else {
				log.Println("transaction rolled back")
			}
		}
	}()

	_, err = db.CreateBranch(plan, parentBranch, req.Name, tx)

	if err != nil {
		log.Printf("Error creating branch: %v\n", err)
		http.Error(w, "Error creating branch: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// commit the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v\n", err)
		http.Error(w, "Error committing transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully created branch")
}

func DeleteBranchHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for DeleteBranchHandler")

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

	if branch == "main" {
		log.Println("Cannot delete main branch")
		http.Error(w, "Cannot delete main branch", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	repoLockId, err := db.LockRepo(
		db.LockRepoParams{
			OrgId:    auth.OrgId,
			UserId:   auth.User.Id,
			PlanId:   planId,
			Branch:   "main",
			Scope:    db.LockScopeRead,
			Ctx:      ctx,
			CancelFn: cancel,
		},
	)

	if err != nil {
		log.Printf("Error locking repo: %v\n", err)
		http.Error(w, "Error locking repo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	defer func() {
		err := db.UnlockRepo(repoLockId)
		if err != nil {
			log.Printf("Error unlocking repo: %v\n", err)
		}
	}()

	err = db.DeleteBranch(auth.OrgId, planId, branch)

	if err != nil {
		log.Printf("Error deleting branch: %v\n", err)
		http.Error(w, "Error deleting branch: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully deleted branch")
}
