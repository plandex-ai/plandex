package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"plandex-server/db"
	"plandex-server/types"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
)

func CreatePlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreatePlanHandler")

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	if !auth.HasPermission(types.PermissionCreatePlan) {
		log.Println("User does not have permission to create a plan")
		http.Error(w, "User does not have permission to create a plan", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	projectId := vars["projectId"]

	log.Println("projectId: ", projectId)

	if !authorizeProject(w, projectId, auth) {
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
			if user.NumNonDraftPlans >= types.TrialMaxPlans {
				writeApiError(w, shared.ApiError{
					Type:   shared.ApiErrorTypeTrialPlansExceeded,
					Status: http.StatusForbidden,
					Msg:    "User has reached max number of anonymous trial plans",
					TrialPlansExceededError: &shared.TrialPlansExceededError{
						MaxPlans: types.TrialMaxPlans,
					},
				})
				return
			}
		}
	}

	// read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var requestBody shared.CreatePlanRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	name := requestBody.Name
	if name == "" {
		name = "draft"
	}

	if name == "draft" {
		// delete any existing draft plans
		err = db.DeleteDraftPlans(auth.OrgId, projectId, auth.User.Id)

		if err != nil {
			log.Printf("Error deleting draft plans: %v\n", err)
			http.Error(w, "Error deleting draft plans: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		i := 2
		originalName := name
		for {
			var count int
			err := db.Conn.Get(&count, "SELECT COUNT(*) FROM plans WHERE project_id = $1 AND owner_id = $2 AND name = $3", projectId, auth.User.Id, name)

			if err != nil {
				log.Printf("Error checking if plan exists: %v\n", err)
				http.Error(w, "Error checking if plan exists: "+err.Error(), http.StatusInternalServerError)
				return
			}

			if count == 0 {
				break
			}

			name = originalName + "." + fmt.Sprint(i)
			i++
		}
	}

	plan, err := db.CreatePlan(auth.OrgId, projectId, auth.User.Id, name)

	if err != nil {
		log.Printf("Error creating plan: %v\n", err)
		http.Error(w, "Error creating plan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := shared.CreatePlanResponse{
		Id:   plan.Id,
		Name: plan.Name,
	}

	bytes, err := json.Marshal(resp)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(bytes)

	log.Printf("Successfully created plan: %v\n", plan)
}

func GetPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for GetPlanHandler")

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	plan := authorizePlan(w, planId, auth)

	if plan == nil {
		return
	}

	bytes, err := json.Marshal(plan)

	if err != nil {
		log.Printf("Error marshalling plan: %v\n", err)
		http.Error(w, "Error marshalling plan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(bytes)
}

func DeletePlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for DeletePlanHandler")

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

	plan := authorizePlanDelete(w, planId, auth)

	if plan == nil {
		return
	}

	if plan.OwnerId != auth.User.Id {
		log.Println("Only the plan owner can delete a plan")
		http.Error(w, "Only the plan owner can delete a plan", http.StatusForbidden)
		return
	}

	res, err := db.Conn.Exec("DELETE FROM plans WHERE id = $1", planId)

	if err != nil {
		log.Printf("Error deleting plan: %v\n", err)
		http.Error(w, "Error deleting plan: "+err.Error(), http.StatusInternalServerError)
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

	err = db.DeletePlanDir(auth.OrgId, planId)

	if err != nil {
		log.Printf("Error deleting plan dir: %v\n", err)
		http.Error(w, "Error deleting plan dir: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully deleted plan", planId)
}

func DeleteAllPlansHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for DeleteAllPlansHandler")

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	projectId := vars["projectId"]

	log.Println("projectId: ", projectId)

	if !authorizeProject(w, projectId, auth) {
		return
	}

	err := db.DeleteOwnerPlans(auth.OrgId, projectId, auth.User.Id)

	if err != nil {
		log.Printf("Error deleting plans: %v\n", err)
		http.Error(w, "Error deleting plans: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully deleted all plans")
}

func ListPlansHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListPlans")

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	projectIds := r.URL.Query()["projectId"]

	log.Println("projectIds: ", projectIds)

	if len(projectIds) == 0 {
		log.Println("No project ids provided")
		http.Error(w, "No project ids provided", http.StatusBadRequest)
		return
	}

	for _, projectId := range projectIds {
		if !authorizeProject(w, projectId, auth) {
			return
		}
	}

	plans, err := db.ListOwnedPlans(projectIds, auth.User.Id, false)

	if err != nil {
		log.Printf("Error listing plans: %v\n", err)
		http.Error(w, "Error listing plans: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiPlans []*shared.Plan
	for _, plan := range plans {
		apiPlans = append(apiPlans, plan.ToApi())
	}

	bytes, err := json.Marshal(apiPlans)

	if err != nil {
		log.Printf("Error marshalling plans: %v\n", err)
		http.Error(w, "Error marshalling plans: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(bytes)
}

func ListArchivedPlansHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListArchivedPlansHandler")
	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	projectIds := r.URL.Query()["projectId"]

	log.Println("projectIds: ", projectIds)

	if len(projectIds) == 0 {
		log.Println("No project ids provided")
		http.Error(w, "No project ids provided", http.StatusBadRequest)
		return
	}

	for _, projectId := range projectIds {
		if !authorizeProject(w, projectId, auth) {
			return
		}
	}

	plans, err := db.ListOwnedPlans(projectIds, "", true)

	if err != nil {
		log.Printf("Error listing plans: %v\n", err)
		http.Error(w, "Error listing plans: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiPlans []*shared.Plan
	for _, plan := range plans {
		apiPlans = append(apiPlans, plan.ToApi())
	}

	jsonBytes, err := json.Marshal(apiPlans)
	if err != nil {
		log.Printf("Error marshalling plans: %v\n", err)
		http.Error(w, "Error marshalling plans: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully processed ListArchivedPlansHandler request")

	w.Write(jsonBytes)
}

func ListPlansRunningHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListPlansRunningHandler")
	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	projectIds := r.URL.Query()["projectId"]
	includeRecent := r.URL.Query().Get("recent") == "true"

	log.Println("projectIds: ", projectIds)

	if len(projectIds) == 0 {
		log.Println("No project ids provided")
		http.Error(w, "No project ids provided", http.StatusBadRequest)
		return
	}

	for _, projectId := range projectIds {
		if !authorizeProject(w, projectId, auth) {
			return
		}
	}

	plans, err := db.ListOwnedPlans(projectIds, auth.User.Id, false)

	if err != nil {
		log.Printf("Error listing plans: %v\n", err)
		http.Error(w, "Error listing plans: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var planIds []string
	for _, plan := range plans {
		planIds = append(planIds, plan.Id)
	}

	errCh := make(chan error)
	var streams []*db.ModelStream
	var branches []*db.Branch

	go func() {
		var err error
		if includeRecent {
			streams, err = db.GetActiveOrRecentModelStreams(planIds)
		} else {
			streams, err = db.GetActiveModelStreams(planIds)
		}
		if err != nil {
			errCh <- fmt.Errorf("error getting recent model streams: %v", err)
			return
		}
		errCh <- nil
	}()

	go func() {
		var err error
		branches, err = db.ListBranchesForPlans(auth.OrgId, planIds)
		if err != nil {
			errCh <- fmt.Errorf("error getting branches: %v", err)
			return
		}
		errCh <- nil
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	res := shared.ListPlansRunningResponse{
		Branches:                   []*shared.Branch{},
		StreamStartedAtByBranchId:  map[string]time.Time{},
		StreamFinishedAtByBranchId: map[string]time.Time{},
		PlansById:                  map[string]*shared.Plan{},
		StreamIdByBranchId:         map[string]string{},
	}

	var apiPlansById = make(map[string]*shared.Plan)
	for _, plan := range plans {
		apiPlan := plan.ToApi()
		apiPlansById[plan.Id] = apiPlan
	}

	var apiBranchesByComposite = make(map[string]*shared.Branch)
	for _, branch := range branches {
		apiBranch := branch.ToApi()
		apiBranchesByComposite[branch.PlanId+"|"+branch.Name] = apiBranch
	}

	addedBranches := make(map[string]bool)
	for _, stream := range streams {
		branchComposite := stream.PlanId + "|" + stream.Branch
		apiBranch, ok := apiBranchesByComposite[branchComposite]
		if !ok {
			log.Printf("Stream %s has no branch\n", stream.Id)
			http.Error(w, "Stream has no branch", http.StatusInternalServerError)
			return
		}

		apiPlan, ok := apiPlansById[stream.PlanId]
		if !ok {
			log.Printf("Stream %s has no plan\n", stream.Id)
			http.Error(w, "Stream has no plan", http.StatusInternalServerError)
			return
		}

		if !addedBranches[branchComposite] {
			res.Branches = append(res.Branches, apiBranch)
			addedBranches[branchComposite] = true
		}

		res.StreamStartedAtByBranchId[apiBranch.Id] = stream.CreatedAt
		if stream.FinishedAt != nil {
			res.StreamFinishedAtByBranchId[apiBranch.Id] = *stream.FinishedAt
		}
		res.StreamIdByBranchId[apiBranch.Id] = stream.Id

		res.PlansById[stream.PlanId] = apiPlan
	}

	sort.Slice(res.Branches, func(i, j int) bool {
		iComposite := res.Branches[i].PlanId + "|" + res.Branches[i].Name
		jComposite := res.Branches[j].PlanId + "|" + res.Branches[j].Name
		iFinishedAt, iOk := res.StreamFinishedAtByBranchId[iComposite]
		jFinishedAt, jOk := res.StreamFinishedAtByBranchId[jComposite]
		iCreatedAt := res.StreamStartedAtByBranchId[iComposite]
		jCreatedAt := res.StreamStartedAtByBranchId[jComposite]

		if iOk && jOk {
			return iFinishedAt.Before(jFinishedAt) // Sort finished streams by finishedAt in ascending order.
		}
		if iOk {
			return false // Place i after j if i is finished and j is not.
		}
		if jOk {
			return true // Place i before j if i is not finished and j is.
		}
		return iCreatedAt.Before(jCreatedAt) // Sort by createdAt in ascending order if both are unfinished.
	})

	bytes, err := json.Marshal(res)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully processed ListPlansRunningHandler request")

	w.Write(bytes)
}

func GetCurrentBranchByPlanIdHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CurrentBranchByPlanIdHandler")
	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	projectId := vars["projectId"]

	log.Println("projectId: ", projectId)

	if !authorizeProject(w, projectId, auth) {
		return
	}

	var req shared.GetCurrentBranchByPlanIdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	plans, err := db.ListOwnedPlans([]string{projectId}, auth.User.Id, false)

	if err != nil {
		log.Printf("Error listing plans: %v\n", err)
		http.Error(w, "Error listing plans: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(plans) == 0 {
		log.Println("No plans found")
		http.Error(w, "No plans found", http.StatusNotFound)
		return
	}

	query := "SELECT * FROM branches WHERE "

	var orConditions []string
	var queryArgs []interface{}
	currentArg := 1
	for _, plan := range plans {
		branchName, ok := req.CurrentBranchByPlanId[plan.Id]

		if !ok {
			continue
		}

		orConditions = append(orConditions, fmt.Sprintf("(plan_id = $%d AND name = $%d)", currentArg, currentArg+1))
		queryArgs = append(queryArgs, plan.Id, branchName)

		currentArg += 2
	}

	query += "(" + strings.Join(orConditions, " OR ") + ") AND archived_at IS NULL AND deleted_at IS NULL"

	var branches []db.Branch
	err = db.Conn.Select(&branches, query, queryArgs...)

	if err != nil {
		log.Printf("Error getting branches: %v\n", err)
		http.Error(w, "Error getting branches: "+err.Error(), http.StatusInternalServerError)
		return
	}

	res := map[string]*shared.Branch{}
	for _, branch := range branches {
		res[branch.PlanId] = branch.ToApi()
	}

	bytes, err := json.Marshal(res)

	if err != nil {
		log.Printf("Error marshalling branches: %v\n", err)
		http.Error(w, "Error marshalling branches: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully processed GetCurrentBranchByPlanIdHandler request")

	w.Write(bytes)
}
