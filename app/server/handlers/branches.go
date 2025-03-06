package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"plandex-server/db"

	shared "plandex-shared"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

func ListBranchesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListBranchesHandler")

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

	ctx, cancel := context.WithCancel(r.Context())
	var branches []*db.Branch

	err = db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    auth.OrgId,
		UserId:   auth.User.Id,
		PlanId:   planId,
		Branch:   "main",
		Reason:   "list branches",
		Scope:    db.LockScopeRead,
		Ctx:      ctx,
		CancelFn: cancel,
	}, func(repo *db.GitRepo) error {
		res, err := db.ListPlanBranches(repo, planId)

		if err != nil {
			return err
		}

		branches = res

		return nil
	})

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

	auth := Authenticate(w, r, true)
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

	ctx, cancel := context.WithCancel(r.Context())

	err = db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    auth.OrgId,
		UserId:   auth.User.Id,
		PlanId:   planId,
		Branch:   "main",
		Reason:   "create branch",
		Scope:    db.LockScopeWrite,
		Ctx:      ctx,
		CancelFn: cancel,
	}, func(repo *db.GitRepo) error {

		err := db.WithTx(ctx, "create branch", func(tx *sqlx.Tx) error {
			_, err = db.CreateBranch(repo, plan, parentBranch, req.Name, tx)

			if err != nil {
				return fmt.Errorf("error creating branch: %v", err)
			}

			return nil
		})

		return err
	})

	if err != nil {
		log.Printf("Error creating branch: %v\n", err)
		http.Error(w, "Error creating branch: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully created branch")
}

func DeleteBranchHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for DeleteBranchHandler")

	auth := Authenticate(w, r, true)
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

	ctx, cancel := context.WithCancel(r.Context())

	err := db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    auth.OrgId,
		UserId:   auth.User.Id,
		PlanId:   planId,
		Branch:   "main",
		Reason:   "delete branch",
		Scope:    db.LockScopeWrite,
		Ctx:      ctx,
		CancelFn: cancel,
	}, func(repo *db.GitRepo) error {
		err := repo.GitDeleteBranch(branch)
		return err
	})

	if err != nil {
		log.Printf("Error deleting branch: %v\n", err)
		http.Error(w, "Error deleting branch: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully deleted branch")
}
