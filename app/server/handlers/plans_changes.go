package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"plandex-server/db"
	modelPlan "plandex-server/model/plan"
	"time"

	shared "plandex-shared"

	"github.com/gorilla/mux"
)

func CurrentPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CurrentPlanHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]
	sha := vars["sha"]

	log.Println("planId: ", planId, "branch: ", branch, "sha: ", sha)

	if authorizePlan(w, planId, auth) == nil {
		return
	}

	// Just in case this was sent immediately after a stream finished, wait a little before locking to allow for cleanup
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithCancel(r.Context())
	scope := db.LockScopeRead
	if sha != "" {
		scope = db.LockScopeWrite
	}
	log.Printf("locking with scope: %s", scope)

	var planState *shared.CurrentPlanState

	err := db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    auth.OrgId,
		UserId:   auth.User.Id,
		PlanId:   planId,
		Branch:   branch,
		Scope:    scope,
		Ctx:      ctx,
		CancelFn: cancel,
	}, func(repo *db.GitRepo) error {
		var err error
		if sha != "" {
			err = repo.GitCheckoutSha(sha)
			if err != nil {
				return fmt.Errorf("error checking out sha: %v", err)
			}

			defer func() {
				checkoutErr := repo.GitCheckoutBranch(branch)
				if checkoutErr != nil {
					log.Printf("Error checking out branch: %v\n", checkoutErr)
				}
			}()
		}

		planState, err = db.GetCurrentPlanState(db.CurrentPlanStateParams{
			OrgId:  auth.OrgId,
			PlanId: planId,
		})

		if err != nil {
			return fmt.Errorf("error getting current plan state: %v", err)
		}

		return nil
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

	auth := Authenticate(w, r, true)
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

	// read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var requestBody shared.ApplyPlanRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	// Just in case this was sent immediately after a stream finished, wait a little before locking to allow for cleanup
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithCancel(r.Context())

	var settings *shared.PlanSettings
	var currentPlanParams db.CurrentPlanStateParams
	var currentPlan *shared.CurrentPlanState

	err = db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    auth.OrgId,
		UserId:   auth.User.Id,
		PlanId:   planId,
		Branch:   branch,
		Scope:    db.LockScopeRead,
		Ctx:      ctx,
		CancelFn: cancel,
	}, func(repo *db.GitRepo) error {
		var err error
		settings, err = db.GetPlanSettings(plan, true)
		if err != nil {
			return fmt.Errorf("error getting plan settings: %v", err)
		}

		currentPlanParams, err = db.GetFullCurrentPlanStateParams(auth.OrgId, planId)
		if err != nil {
			return fmt.Errorf("error getting current plan state params: %v", err)
		}

		currentPlan, err = db.GetCurrentPlanState(currentPlanParams)
		if err != nil {
			return fmt.Errorf("error getting current plan state: %v", err)
		}

		return nil
	})

	if err != nil {
		log.Printf("Error getting current plan state: %v\n", err)
		http.Error(w, "Error getting current plan state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("ApplyPlanHandler: Got current plan state:", currentPlan != nil)

	clients := initClients(
		initClientsParams{
			w:           w,
			auth:        auth,
			apiKeys:     requestBody.ApiKeys,
			openAIBase:  requestBody.OpenAIBase,
			openAIOrgId: requestBody.OpenAIOrgId,
			plan:        plan,
		},
	)

	commitMsg, err := modelPlan.GenCommitMsgForPendingResults(auth, plan, clients, settings, currentPlan, r.Context())

	if err != nil {
		log.Printf("Error generating commit message: %v\n", err)
		http.Error(w, "Error generating commit message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:          auth.OrgId,
		UserId:         auth.User.Id,
		PlanId:         planId,
		Branch:         branch,
		Scope:          db.LockScopeWrite,
		Ctx:            ctx,
		CancelFn:       cancel,
		ClearRepoOnErr: true,
	}, func(repo *db.GitRepo) error {
		return db.ApplyPlan(repo, ctx, db.ApplyPlanParams{
			OrgId:                  auth.OrgId,
			UserId:                 auth.User.Id,
			BranchName:             branch,
			Plan:                   plan,
			CurrentPlanState:       currentPlan,
			CurrentPlanStateParams: &currentPlanParams,
			CommitMsg:              commitMsg,
		})
	})

	if err != nil {
		log.Printf("Error applying plan: %v\n", err)
		http.Error(w, "Error applying plan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(commitMsg))

	log.Println("Successfully applied plan", planId)
}

func RejectAllChangesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for RejectAllChangesHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]
	log.Println("planId: ", planId, "branch: ", branch)

	if authorizePlan(w, planId, auth) == nil {
		return
	}

	ctx, cancel := context.WithCancel(r.Context())

	err := db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:          auth.OrgId,
		UserId:         auth.User.Id,
		PlanId:         planId,
		Branch:         branch,
		Scope:          db.LockScopeWrite,
		Ctx:            ctx,
		CancelFn:       cancel,
		ClearRepoOnErr: true,
	}, func(repo *db.GitRepo) error {
		err := db.RejectAllResults(auth.OrgId, planId)
		if err != nil {
			return err
		}

		err = repo.GitAddAndCommit(branch, "🚫 Rejected all pending changes")
		if err != nil {
			return fmt.Errorf("error committing rejected changes: %v", err)
		}

		return nil
	})

	if err != nil {
		log.Printf("Error rejecting all changes: %v\n", err)
		http.Error(w, "Error rejecting all changes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully rejected all changes for plan", planId)
}

func RejectFileHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for RejectFileHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]

	log.Println("planId: ", planId, "branch: ", branch)

	if authorizePlan(w, planId, auth) == nil {
		return
	}

	var req shared.RejectFileRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Error decoding request: %v\n", err)
		http.Error(w, "Error decoding request: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())

	err = db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:          auth.OrgId,
		UserId:         auth.User.Id,
		PlanId:         planId,
		Branch:         branch,
		Scope:          db.LockScopeWrite,
		Ctx:            ctx,
		CancelFn:       cancel,
		ClearRepoOnErr: true,
	}, func(repo *db.GitRepo) error {
		err = db.RejectPlanFile(auth.OrgId, planId, req.FilePath, time.Now())
		if err != nil {
			return err
		}

		err = repo.GitAddAndCommit(branch, fmt.Sprintf("🚫 Rejected pending changes to file: %s", req.FilePath))
		if err != nil {
			return fmt.Errorf("error committing rejected changes: %v", err)
		}

		return nil
	})

	if err != nil {
		log.Printf("Error rejecting result: %v\n", err)
		http.Error(w, "Error rejecting result: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully rejected plan file", req.FilePath)
}

func RejectFilesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for RejectFilesHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]

	log.Println("planId: ", planId, "branch: ", branch)

	if authorizePlan(w, planId, auth) == nil {
		return
	}

	var req shared.RejectFilesRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Error decoding request: %v\n", err)
		http.Error(w, "Error decoding request: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())

	err = db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:          auth.OrgId,
		UserId:         auth.User.Id,
		PlanId:         planId,
		Branch:         branch,
		Scope:          db.LockScopeWrite,
		Ctx:            ctx,
		CancelFn:       cancel,
		ClearRepoOnErr: true,
	}, func(repo *db.GitRepo) error {
		err = db.RejectPlanFiles(auth.OrgId, planId, req.Paths, time.Now())
		if err != nil {
			return err
		}

		msg := "🚫 Rejected pending changes to file"
		if len(req.Paths) > 1 {
			msg += "s"
		}
		msg += ":"

		for _, path := range req.Paths {
			msg += fmt.Sprintf("\n • %s", path)
		}

		err = repo.GitAddAndCommit(branch, msg)
		if err != nil {
			return fmt.Errorf("error committing rejected changes: %v", err)
		}

		return nil
	})

	if err != nil {
		log.Printf("Error rejecting result: %v\n", err)
		http.Error(w, "Error rejecting result: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully rejected plan files", req.Paths)
}

func ArchivePlanHandler(w http.ResponseWriter, r *http.Request) {
	auth := Authenticate(w, r, true)
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

func UnarchivePlanHandler(w http.ResponseWriter, r *http.Request) {
	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	log.Println("Received request for UnarchivePlanHandler")

	vars := mux.Vars(r)
	planId := vars["planId"]
	log.Println("planId: ", planId)

	plan := authorizePlanArchive(w, planId, auth)

	if plan == nil {
		return
	}

	if plan.ArchivedAt == nil {
		log.Println("Plan isn't archived")
		http.Error(w, "Plan isn't archived", http.StatusBadRequest)
		return
	}

	res, err := db.Conn.Exec("UPDATE plans SET archived_at = NULL WHERE id = $1", planId)

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

	log.Println("Successfully unarchived plan", planId)
}

func GetPlanDiffsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for GetPlanDiffs")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]
	plain := r.URL.Query().Get("plain") == "true"

	log.Println("planId: ", planId, "branch: ", branch)

	if authorizePlan(w, planId, auth) == nil {
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	var diffs string

	err := db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    auth.OrgId,
		UserId:   auth.User.Id,
		PlanId:   planId,
		Branch:   branch,
		Scope:    db.LockScopeRead,
		Ctx:      ctx,
		CancelFn: cancel,
	}, func(repo *db.GitRepo) error {
		var err error
		diffs, err = db.GetPlanDiffs(auth.OrgId, planId, plain)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Printf("Error getting plan diffs: %v\n", err)
		http.Error(w, "Error getting plan diffs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(diffs))

	log.Println("Successfully retrieved plan diffs")
}
