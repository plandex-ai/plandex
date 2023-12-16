package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"plandex-server/db"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
)

func ListContextHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListContextHandler")

	// TODO: get from auth when implemented
	// currentUserId := "user1"
	currentOrgId := "org1"

	// TODO: authenticate user and plan access
	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	dbContexts, err := db.GetPlanContexts(currentOrgId, planId, false)

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

	// TODO: get from auth when implemented
	currentUserId := "user1"
	currentOrgId := "org1"

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

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

	// load plan
	plan, err := db.GetPlan(planId)

	if err != nil {
		log.Printf("Plan not found: %v\n", err)
		http.Error(w, "Plan not found: "+err.Error(), http.StatusBadRequest)
	}

	maxTokens := shared.MaxContextTokens
	tokensAdded := 0
	totalTokens := plan.ContextTokens

	paramsById := make(map[string]*shared.LoadContextParams)
	numTokensById := make(map[string]int)

	for _, context := range requestBody {
		id := uuid.New().String()
		numTokens, err := shared.GetNumTokens(context.Body)

		if err != nil {
			log.Printf("Error getting num tokens: %v\n", err)
			http.Error(w, "Error getting num tokens: "+err.Error(), http.StatusInternalServerError)
			return
		}

		paramsById[id] = context
		numTokensById[id] = numTokens

		tokensAdded += numTokens
		totalTokens += numTokens
	}

	if totalTokens > maxTokens {
		log.Printf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, maxTokens)
		res := shared.LoadContextResponse{
			TokensAdded:       tokensAdded,
			TotalTokens:       totalTokens,
			MaxTokensExceeded: true,
		}

		bytes, err := json.Marshal(res)

		if err != nil {
			log.Printf("Error marshalling response: %v\n", err)
			http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(bytes)
		return
	}

	dbContextsCh := make(chan *db.Context)
	errCh := make(chan error)
	for id, params := range paramsById {

		go func(id string, params *shared.LoadContextParams) {
			hash := sha256.Sum256([]byte(params.Body))
			sha := hex.EncodeToString(hash[:])

			context := db.Context{
				Id:          id,
				OrgId:       currentOrgId,
				CreatorId:   currentUserId,
				PlanId:      planId,
				ContextType: params.ContextType,
				Name:        params.Name,
				Url:         params.Url,
				FilePath:    params.FilePath,
				NumTokens:   numTokensById[id],
				Sha:         sha,
				Body:        params.Body,
			}

			err := db.StoreContext(&context)

			if err != nil {
				errCh <- err
				return
			}

			dbContextsCh <- &context

		}(id, params)
	}

	var apiContexts []*shared.Context

	for i := 0; i < len(requestBody); i++ {
		select {
		case err := <-errCh:
			log.Printf("Error creating context: %v\n", err)
			http.Error(w, "Error creating context: "+err.Error(), http.StatusInternalServerError)
			return
		case dbContext := <-dbContextsCh:
			apiContext := dbContext.ToApi()
			apiContext.Body = ""
			apiContexts = append(apiContexts, apiContext)
		}
	}

	commitMsg := shared.SummaryForLoadContext(apiContexts) + "\n\n" + shared.TableForLoadContext(apiContexts)
	err = db.GitAddAndCommit(currentOrgId, planId, commitMsg)

	if err != nil {
		log.Printf("Error committing changes: %v\n", err)
		http.Error(w, "Error committing changes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = db.AddPlanContextTokens(planId, tokensAdded)
	if err != nil {
		log.Printf("Error updating plan tokens: %v\n", err)
		http.Error(w, "Error updating plan tokens: "+err.Error(), http.StatusInternalServerError)
		return
	}

	res := shared.LoadContextResponse{
		TokensAdded: tokensAdded,
		TotalTokens: totalTokens,
		Msg:         commitMsg,
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

	// TODO: get from auth when implemented
	// currentUserId := "user1"
	currentOrgId := "org1"

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

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

	// load plan
	plan, err := db.GetPlan(planId)

	if err != nil {
		log.Printf("Plan not found: %v\n", err)
		http.Error(w, "Plan not found: "+err.Error(), http.StatusBadRequest)
	}

	maxTokens := shared.MaxContextTokens
	tokensAdded := 0
	totalTokens := plan.ContextTokens
	updateNumTokensById := make(map[string]int)
	contextsById := make(map[string]*db.Context)

	var mu sync.Mutex
	errCh := make(chan error)

	for id, params := range requestBody {
		go func(id string, params *shared.UpdateContextParams) {
			mu.Lock()
			defer mu.Unlock()

			context, err := db.GetContext(currentOrgId, planId, id, true)

			if err != nil {
				errCh <- fmt.Errorf("error getting context: %v", err)
				return
			}

			contextsById[id] = context

			updateNumTokens, err := shared.GetNumTokens(params.Body)

			tokenDiff := updateNumTokens - context.NumTokens

			if err != nil {
				errCh <- fmt.Errorf("error getting num tokens: %v", err)
				return
			}

			updateNumTokensById[id] = updateNumTokens
			tokensAdded += tokenDiff
			totalTokens += tokenDiff
		}(id, params)
	}

	if totalTokens > maxTokens {
		log.Printf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, maxTokens)
		res := shared.LoadContextResponse{
			TokensAdded:       tokensAdded,
			TotalTokens:       totalTokens,
			MaxTokensExceeded: true,
		}

		bytes, err := json.Marshal(res)

		if err != nil {
			log.Printf("Error marshalling response: %v\n", err)
			http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(bytes)
		return
	}

	updatedContextsCh := make(chan *db.Context)
	errCh = make(chan error)
	for id, params := range requestBody {
		go func(id string, params *shared.UpdateContextParams) {
			context := contextsById[id]

			hash := sha256.Sum256([]byte(params.Body))
			sha := hex.EncodeToString(hash[:])

			context.Body = params.Body
			context.Sha = sha
			context.NumTokens = updateNumTokensById[id]

			err = db.StoreContext(context)

			if err != nil {
				errCh <- err
				return
			}

			updatedContextsCh <- context

		}(id, params)
	}

	var apiContexts []*shared.Context
	for i := 0; i < len(requestBody); i++ {
		select {
		case err := <-errCh:
			log.Printf("Error creating context: %v\n", err)
			http.Error(w, "Error creating context: "+err.Error(), http.StatusInternalServerError)
			return
		case dbContext := <-updatedContextsCh:
			apiContext := dbContext.ToApi()
			apiContext.Body = ""
			apiContexts = append(apiContexts, apiContext)
		}
	}

	commitMsg := shared.SummaryForUpdateContext(apiContexts) + "\n\n" + shared.TableForContextUpdate(apiContexts)
	err = db.GitAddAndCommit(currentOrgId, planId, commitMsg)

	if err != nil {
		log.Printf("Error committing changes: %v\n", err)
		http.Error(w, "Error committing changes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = db.AddPlanContextTokens(planId, tokensAdded)
	if err != nil {
		log.Printf("Error updating plan tokens: %v\n", err)
		http.Error(w, "Error updating plan tokens: "+err.Error(), http.StatusInternalServerError)
		return
	}

	res := shared.UpdateContextResponse{
		TokensAdded: tokensAdded,
		TotalTokens: totalTokens,
		Msg:         commitMsg,
	}

	bytes, err := json.Marshal(res)

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

	// TODO: get from auth when implemented
	// currentUserId := "user1"
	currentOrgId := "org1"

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	// load plan
	plan, err := db.GetPlan(planId)

	if err != nil {
		log.Printf("Plan not found: %v\n", err)
		http.Error(w, "Plan not found: "+err.Error(), http.StatusBadRequest)
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

	dbContexts, err := db.GetPlanContexts(currentOrgId, planId, false)

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

	commitMsg := shared.SummaryForRemoveContext(toRemoveApiContexts, plan.ContextTokens) + "\n\n" + shared.TableForRemoveContext(toRemoveApiContexts)
	err = db.GitAddAndCommit(currentOrgId, planId, commitMsg)

	if err != nil {
		log.Printf("Error committing changes: %v\n", err)
		http.Error(w, "Error committing changes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = db.AddPlanContextTokens(planId, -removeTokens)
	if err != nil {
		log.Printf("Error updating plan tokens: %v\n", err)
		http.Error(w, "Error updating plan tokens: "+err.Error(), http.StatusInternalServerError)
		return
	}

	res := shared.DeleteContextResponse{
		TokensRemoved: removeTokens,
		TotalTokens:   plan.ContextTokens - removeTokens,
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
