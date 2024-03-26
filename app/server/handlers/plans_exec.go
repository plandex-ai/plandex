package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"plandex-server/db"
	"plandex-server/host"
	"plandex-server/model"
	modelPlan "plandex-server/model/plan"
	"plandex-server/types"
	"time"

	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
)

const TrialMaxReplies = 10

func TellPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for TellPlanHandler", "ip:", host.Ip)

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]

	log.Println("planId: ", planId)

	plan := authorizePlanExecUpdate(w, planId, auth)
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

	var requestBody shared.TellPlanRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	if requestBody.ApiKey == "" {
		log.Println("API key is required")
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	if os.Getenv("IS_CLOUD") != "" {
		user, err := db.GetUser(auth.User.Id)

		if err != nil {
			log.Printf("Error getting user: %v\n", err)
			http.Error(w, "Error getting user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if user.IsTrial {
			if plan.TotalReplies >= types.TrialMaxReplies {
				writeApiError(w, shared.ApiError{
					Type:   shared.ApiErrorTypeTrialMessagesExceeded,
					Status: http.StatusForbidden,
					Msg:    "Anonymous trial message limit exceeded",
					TrialMessagesExceededError: &shared.TrialMessagesExceededError{
						MaxReplies: types.TrialMaxReplies,
					},
				})
				return
			}
		}
	}

	client := model.NewClient(requestBody.ApiKey)
	err = modelPlan.Tell(client, plan, branch, auth, &requestBody)

	if err != nil {
		log.Printf("Error telling plan: %v\n", err)
		http.Error(w, "Error telling plan", http.StatusInternalServerError)
		return
	}

	if requestBody.ConnectStream {
		startResponseStream(w, auth, planId, branch, false)
	}

	log.Println("Successfully processed request for TellPlanHandler")
}

func BuildPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for BuildPlanHandler", "ip:", host.Ip)
	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]

	log.Println("planId: ", planId)
	plan := authorizePlanExecUpdate(w, planId, auth)
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

	var requestBody shared.BuildPlanRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	if requestBody.ApiKey == "" {
		log.Println("API key is required")
		http.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	client := model.NewClient(requestBody.ApiKey)
	numBuilds, err := modelPlan.Build(client, plan, branch, auth)

	if err != nil {
		log.Printf("Error building plan: %v\n", err)
		http.Error(w, "Error building plan", http.StatusInternalServerError)
		return
	}

	if numBuilds == 0 {
		log.Println("No builds were executed")
		http.Error(w, shared.NoBuildsErr, http.StatusNotFound)
		return
	}

	if requestBody.ConnectStream {
		startResponseStream(w, auth, planId, branch, false)
	}

	log.Println("Successfully processed request for BuildPlanHandler")
}

func ConnectPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ConnectPlanHandler", "ip:", host.Ip)

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]
	log.Println("planId: ", planId)
	log.Println("branch: ", branch)
	active := modelPlan.GetActivePlan(planId, branch)
	isProxy := r.URL.Query().Get("proxy") == "true"

	if active == nil {
		if isProxy {
			log.Println("No active plan on proxied request")
			http.Error(w, "No active plan", http.StatusNotFound)
			return
		}

		log.Println("No active plan -- proxying request")

		proxyActivePlanMethod(w, r, planId, branch, "connect")
		return
	}

	auth := authenticate(w, r, true)
	if auth == nil {
		log.Println("No auth")
		return
	}

	plan := authorizePlan(w, planId, auth)
	if plan == nil {
		log.Println("No plan")
		return
	}

	startResponseStream(w, auth, planId, branch, true)

	log.Println("Successfully processed request for ConnectPlanHandler")
}

func StopPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for StopPlanHandler", "ip:", host.Ip)

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]
	log.Println("planId: ", planId)
	log.Println("branch: ", branch)
	active := modelPlan.GetActivePlan(planId, branch)
	isProxy := r.URL.Query().Get("proxy") == "true"

	if active == nil {
		if isProxy {
			log.Println("No active plan on proxied request")
			http.Error(w, "No active plan", http.StatusNotFound)
			return
		}
		proxyActivePlanMethod(w, r, planId, branch, "stop")
		return
	}

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	if authorizePlan(w, planId, auth) == nil {
		return
	}

	log.Println("Sending stream aborted message to client")

	active.Stream(shared.StreamMessage{
		Type: shared.StreamMessageAborted,
	})

	// give some time for stream message to be processed before canceling
	log.Println("Sleeping for 100ms before canceling")
	time.Sleep(100 * time.Millisecond)

	var err error
	ctx, cancel := context.WithCancel(context.Background())
	unlockFn := lockRepo(w, r, auth, db.LockScopeWrite, ctx, cancel, true)
	if unlockFn == nil {
		return
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	log.Println("Stopping plan")
	err = modelPlan.Stop(planId, branch, auth.User.Id, auth.OrgId)

	if err != nil {
		log.Printf("Error stopping plan: %v\n", err)
		http.Error(w, "Error stopping plan", http.StatusInternalServerError)
		return
	}

	log.Println("Successfully processed request for StopPlanHandler")
}

func RespondMissingFileHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for RespondMissingFileHandler", "ip:", host.Ip)

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]
	log.Println("planId: ", planId)
	log.Println("branch: ", branch)
	isProxy := r.URL.Query().Get("proxy") == "true"

	active := modelPlan.GetActivePlan(planId, branch)
	if active == nil {
		if isProxy {
			log.Println("No active plan on proxied request")
			http.Error(w, "No active plan", http.StatusNotFound)
			return
		}

		proxyActivePlanMethod(w, r, planId, branch, "respond_missing_file")
		return
	}

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

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

	var requestBody shared.RespondMissingFileRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	log.Println("missing file choice:", requestBody.Choice)

	if requestBody.Choice == shared.RespondMissingFileChoiceLoad {
		log.Println("loading missing file")
		res, dbContexts := loadContexts(w, r, auth, &shared.LoadContextRequest{
			&shared.LoadContextParams{
				ContextType: shared.ContextFileType,
				Name:        requestBody.FilePath,
				FilePath:    requestBody.FilePath,
				Body:        requestBody.Body,
			},
		}, plan, branch)
		if res == nil {
			return
		}

		dbContext := dbContexts[0]

		log.Println("loaded missing file:", dbContext.FilePath)

		modelPlan.UpdateActivePlan(planId, branch, func(activePlan *types.ActivePlan) {
			activePlan.Contexts = append(activePlan.Contexts, dbContext)
			activePlan.ContextsByPath[dbContext.FilePath] = dbContext
		})
	}

	// This will resume model stream
	log.Println("Resuming model stream")
	active.MissingFileResponseCh <- requestBody.Choice

	log.Println("Successfully processed request for RespondMissingFileHandler")
}

func authorizePlanExecUpdate(w http.ResponseWriter, planId string, auth *types.ServerAuth) *db.Plan {
	plan := authorizePlan(w, planId, auth)
	if plan == nil {
		return nil
	}

	if plan.OwnerId != auth.User.Id && !auth.HasPermission(types.PermissionUpdateAnyPlan) {
		log.Println("User does not have permission to update plan")
		http.Error(w, "User does not have permission to update plan", http.StatusForbidden)
		return nil
	}

	return plan
}
