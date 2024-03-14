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

func ListContextHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListContextHandler")

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
	unlockFn := lockRepo(w, r, auth, db.LockScopeRead, ctx, cancel, true)
	if unlockFn == nil {
		return
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	dbContexts, err := db.GetPlanContexts(auth.OrgId, planId, false)

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

func LoadContextHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for LoadContextHandler")

	auth := authenticate(w, r, true)
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

	res, _ := loadContexts(w, r, auth, &requestBody, plan, branchName)

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

	auth := authenticate(w, r, true)
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

	ctx, cancel := context.WithCancel(context.Background())
	unlockFn := lockRepo(w, r, auth, db.LockScopeWrite, ctx, cancel, true)
	if unlockFn == nil {
		return
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	updateRes, err := db.UpdateContexts(db.UpdateContextsParams{
		Req:        &requestBody,
		OrgId:      auth.OrgId,
		Plan:       plan,
		BranchName: branchName,
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

	err = db.GitAddAndCommit(auth.OrgId, planId, branchName, updateRes.Msg)

	if err != nil {
		log.Printf("Error committing changes: %v\n", err)
		http.Error(w, "Error committing changes: "+err.Error(), http.StatusInternalServerError)
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

	auth := authenticate(w, r, true)
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

	ctx, cancel := context.WithCancel(context.Background())
	unlockFn := lockRepo(w, r, auth, db.LockScopeWrite, ctx, cancel, true)
	if unlockFn == nil {
		return
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	dbContexts, err := db.GetPlanContexts(auth.OrgId, planId, false)

	if err != nil {
		log.Printf("Error getting contexts: %v\n", err)
		http.Error(w, "Error getting contexts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var toRemove []*db.Context
	for _, dbContext := range dbContexts {
		if _, ok := requestBody.Ids[dbContext.Id]; ok {
			toRemove = append(toRemove, dbContext)
		}
	}

	err = db.ContextRemove(toRemove)

	if err != nil {
		log.Printf("Error deleting contexts: %v\n", err)
		http.Error(w, "Error deleting contexts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	removeTokens := 0
	var toRemoveApiContexts []*shared.Context
	for _, dbContext := range toRemove {
		toRemoveApiContexts = append(toRemoveApiContexts, dbContext.ToApi())
		removeTokens += dbContext.NumTokens
	}

	commitMsg := shared.SummaryForRemoveContext(toRemoveApiContexts, branch.ContextTokens) + "\n\n" + shared.TableForRemoveContext(toRemoveApiContexts)
	err = db.GitAddAndCommit(auth.OrgId, planId, branchName, commitMsg)

	if err != nil {
		log.Printf("Error committing changes: %v\n", err)
		http.Error(w, "Error committing changes: "+err.Error(), http.StatusInternalServerError)
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
