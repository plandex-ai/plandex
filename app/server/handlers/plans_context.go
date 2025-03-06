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
)

func ListContextHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListContextHandler")

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
	var dbContexts []*db.Context

	err := db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    auth.OrgId,
		UserId:   auth.User.Id,
		PlanId:   planId,
		Branch:   branch,
		Reason:   "list contexts",
		Scope:    db.LockScopeRead,
		Ctx:      ctx,
		CancelFn: cancel,
	}, func(repo *db.GitRepo) error {
		res, err := db.GetPlanContexts(auth.OrgId, planId, false, false)
		if err != nil {
			return err
		}

		dbContexts = res

		return nil
	})

	if err != nil {
		log.Printf("Error getting contexts: %v\n", err)
		http.Error(w, "Error getting contexts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiContexts []*shared.Context

	for _, dbContext := range dbContexts {
		apiContexts = append(apiContexts, dbContext.ToApi())
	}

	bytes, err := json.Marshal(apiContexts)

	if err != nil {
		log.Printf("Error marshalling contexts: %v\n", err)
		http.Error(w, "Error marshalling contexts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(bytes)
}

func GetContextBodyHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for GetContextBodyHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]
	contextId := vars["contextId"]
	log.Println("planId:", planId, "branch:", branch, "contextId:", contextId)

	if authorizePlan(w, planId, auth) == nil {
		return
	}

	ctx, cancel := context.WithCancel(r.Context())

	var dbContexts []*db.Context
	err := db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    auth.OrgId,
		UserId:   auth.User.Id,
		PlanId:   planId,
		Branch:   branch,
		Reason:   "get context body",
		Scope:    db.LockScopeRead,
		Ctx:      ctx,
		CancelFn: cancel,
	}, func(repo *db.GitRepo) error {
		res, err := db.GetPlanContexts(auth.OrgId, planId, true, false)
		if err != nil {
			return err
		}

		dbContexts = res

		return nil
	})

	if err != nil {
		log.Printf("Error getting contexts: %v\n", err)
		http.Error(w, "Error getting contexts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var targetContext *db.Context
	for _, dbContext := range dbContexts {
		if dbContext.Id == contextId {
			targetContext = dbContext
			break
		}
	}

	if targetContext == nil {
		http.Error(w, "Context not found", http.StatusNotFound)
		return
	}

	response := shared.GetContextBodyResponse{
		Body: targetContext.Body,
	}

	bytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(bytes)
}

func LoadContextHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for LoadContextHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branchName := vars["branch"]
	log.Println("planId: ", planId)

	plan := authorizePlan(w, planId, auth)
	if plan == nil {
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

	var requestBody shared.LoadContextRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	res, _ := loadContexts(loadContextsParams{
		w:          w,
		r:          r,
		auth:       auth,
		loadReq:    &requestBody,
		plan:       plan,
		branchName: branchName,
	})

	if res == nil {
		return
	}

	bytes, err := json.Marshal(res)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully processed LoadContextHandler request")

	w.Write(bytes)
}

func UpdateContextHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for UpdateContextHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branchName := vars["branch"]
	log.Println("planId: ", planId)

	plan := authorizePlan(w, planId, auth)
	if plan == nil {
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

	var requestBody shared.UpdateContextRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())

	var updateRes *shared.UpdateContextResponse
	err = db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:          auth.OrgId,
		UserId:         auth.User.Id,
		PlanId:         planId,
		Branch:         branchName,
		Reason:         "update contexts",
		Scope:          db.LockScopeWrite,
		Ctx:            ctx,
		CancelFn:       cancel,
		ClearRepoOnErr: true,
	}, func(repo *db.GitRepo) error {
		var err error
		updateRes, err = db.UpdateContexts(db.UpdateContextsParams{
			Req:        &requestBody,
			OrgId:      auth.OrgId,
			Plan:       plan,
			BranchName: branchName,
		})

		if err != nil {
			return err
		}

		if updateRes.MaxTokensExceeded {
			return nil
		}

		err = repo.GitAddAndCommit(branchName, updateRes.Msg)

		if err != nil {
			return fmt.Errorf("error committing changes: %v", err)
		}

		return nil
	})

	if err != nil {
		log.Printf("Error error updating contexts: %v\n", err)
		http.Error(w, "Error error updating contexts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if updateRes.MaxTokensExceeded {
		log.Printf("The total number of tokens (%d) exceeds the maximum allowed (%d)", updateRes.TotalTokens, updateRes.MaxTokens)
		bytes, err := json.Marshal(updateRes)

		if err != nil {
			log.Printf("Error marshalling response: %v\n", err)
			http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(bytes)
		return
	}

	bytes, err := json.Marshal(updateRes)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully processed UpdateContextHandler request")

	w.Write(bytes)
}

func DeleteContextHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for DeleteContextHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branchName := vars["branch"]
	log.Println("planId: ", planId)

	plan := authorizePlan(w, planId, auth)

	if plan == nil {
		return
	}

	branch, err := db.GetDbBranch(planId, branchName)

	if err != nil {
		log.Printf("Error getting branch: %v\n", err)
		http.Error(w, "Error getting branch: "+err.Error(), http.StatusInternalServerError)
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

	var requestBody shared.DeleteContextRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())

	var dbContexts []*db.Context
	var toRemove []*db.Context
	var commitMsg string
	removeTokens := 0
	var toRemoveApiContexts []*shared.Context

	err = db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:          auth.OrgId,
		UserId:         auth.User.Id,
		PlanId:         planId,
		Branch:         branchName,
		Reason:         "delete contexts",
		Scope:          db.LockScopeWrite,
		Ctx:            ctx,
		CancelFn:       cancel,
		ClearRepoOnErr: true,
	}, func(repo *db.GitRepo) error {
		var err error
		dbContexts, err = db.GetPlanContexts(auth.OrgId, planId, false, false)

		if err != nil {
			return fmt.Errorf("error getting contexts: %v", err)
		}

		for _, dbContext := range dbContexts {
			if _, ok := requestBody.Ids[dbContext.Id]; ok {
				toRemove = append(toRemove, dbContext)
			}
		}

		err = db.ContextRemove(auth.OrgId, planId, toRemove)

		if err != nil {
			return fmt.Errorf("error removing contexts: %v", err)
		}

		for _, dbContext := range toRemove {
			toRemoveApiContexts = append(toRemoveApiContexts, dbContext.ToApi())
			removeTokens += dbContext.NumTokens
		}

		commitMsg = shared.SummaryForRemoveContext(toRemoveApiContexts, branch.ContextTokens) + "\n\n" + shared.TableForRemoveContext(toRemoveApiContexts)

		err = repo.GitAddAndCommit(branchName, commitMsg)

		if err != nil {
			return fmt.Errorf("error committing changes: %v", err)
		}

		return nil
	})

	if err != nil {
		log.Printf("Error deleting contexts: %v\n", err)
		http.Error(w, "Error deleting contexts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = db.AddPlanContextTokens(planId, branchName, -removeTokens)
	if err != nil {
		log.Printf("Error updating plan tokens: %v\n", err)
		http.Error(w, "Error updating plan tokens: "+err.Error(), http.StatusInternalServerError)
		return
	}

	res := shared.DeleteContextResponse{
		TokensRemoved: removeTokens,
		TotalTokens:   branch.ContextTokens - removeTokens,
		Msg:           commitMsg,
	}

	bytes, err := json.Marshal(res)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully deleted contexts")

	w.Write(bytes)
}
