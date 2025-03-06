package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"plandex-server/db"

	shared "plandex-shared"

	"github.com/gorilla/mux"
)

func ListLogsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListLogsHandler")

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

	var body string
	var shas []string

	err := db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    auth.OrgId,
		UserId:   auth.User.Id,
		PlanId:   planId,
		Branch:   branch,
		Reason:   "list logs",
		Scope:    db.LockScopeRead,
		Ctx:      ctx,
		CancelFn: cancel,
	}, func(repo *db.GitRepo) error {
		var err error
		body, shas, err = repo.GetGitCommitHistory(branch)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Println("Error getting logs: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res := shared.LogResponse{
		Body: body,
		Shas: shas,
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

	// read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var requestBody shared.RewindPlanRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())

	err = db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    auth.OrgId,
		UserId:   auth.User.Id,
		PlanId:   planId,
		Branch:   branch,
		Reason:   "rewind plan",
		Scope:    db.LockScopeWrite,
		Ctx:      ctx,
		CancelFn: cancel,
	}, func(repo *db.GitRepo) error {
		return repo.GitRewindToSha(branch, requestBody.Sha)
	})

	if err != nil {
		log.Println("Error rewinding plan: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = db.SyncPlanTokens(auth.OrgId, planId, branch)

	if err != nil {
		log.Println("Error syncing plan tokens: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var sha string
	var latest string

	err = db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    auth.OrgId,
		UserId:   auth.User.Id,
		PlanId:   planId,
		Branch:   branch,
		Reason:   "get latest commit",
		Scope:    db.LockScopeRead,
		Ctx:      ctx,
		CancelFn: cancel,
	}, func(repo *db.GitRepo) error {
		sha, latest, err = repo.GetLatestCommit(branch)
		return err
	})

	if err != nil {
		log.Println("Error getting latest commit: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res := shared.RewindPlanResponse{
		LatestSha:    sha,
		LatestCommit: latest,
	}

	bytes, err := json.Marshal(res)

	if err != nil {
		log.Println("Error marshalling response: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(bytes)

	log.Println("Successfully processed request for RewindPlanHandler")
}
