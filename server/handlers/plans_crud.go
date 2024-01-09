package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"plandex-server/db"

	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
)

func CreatePlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreatePlanHandler")

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	projectId := vars["projectId"]

	log.Println("projectId: ", projectId)

	if !authorizeProject(w, projectId, auth.UserId, auth.OrgId) {
		return
	}

	if os.Getenv("IS_CLOUD") != "" {
		user, err := db.GetUser(auth.UserId)

		if err != nil {
			log.Printf("Error getting user: %v\n", err)
			http.Error(w, "Error getting user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if user.IsTrial {
			if user.NumNonDraftPlans >= 10 {
				log.Println("User has reached max number of plans")
				http.Error(w, "User has reached max number of free trial plans", http.StatusForbidden)
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
		err = db.DeleteDraftPlans(auth.OrgId, projectId, auth.UserId)

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
			err := db.Conn.Get(&count, "SELECT COUNT(*) FROM plans WHERE project_id = $1 AND owner_id = $2 AND name = $3", projectId, auth.UserId, name)

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

	plan, err := db.CreatePlan(auth.OrgId, projectId, auth.UserId, name)

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

	plan := authorizePlan(w, planId, auth.UserId, auth.OrgId)

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

	plan := authorizePlan(w, planId, auth.UserId, auth.OrgId)

	if plan == nil {
		return
	}

	if plan.OwnerId != auth.UserId {
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

	if !authorizeProject(w, projectId, auth.UserId, auth.OrgId) {
		return
	}

	err := db.DeleteOwnerPlans(auth.OrgId, projectId, auth.UserId)

	if err != nil {
		log.Printf("Error deleting plans: %v\n", err)
		http.Error(w, "Error deleting plans: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully deleted all plans")
}

func ListPlansHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListPlansHandler")
	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	projectId := vars["projectId"]

	log.Println("projectId: ", projectId)

	if !authorizeProject(w, projectId, auth.UserId, auth.OrgId) {
		return
	}

	plans, err := db.ListOwnedPlans(projectId, auth.UserId, false, "")

	if err != nil {
		log.Printf("Error listing plans: %v\n", err)
		http.Error(w, "Error listing plans: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonBytes, err := json.Marshal(plans)
	if err != nil {
		log.Printf("Error marshalling plans: %v\n", err)
		http.Error(w, "Error marshalling plans: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully processed ListPlansHandler request")

	w.Write(jsonBytes)
}

func ListArchivedPlansHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListArchivedPlansHandler")
	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	projectId := vars["projectId"]

	log.Println("projectId: ", projectId)

	if !authorizeProject(w, projectId, auth.UserId, auth.OrgId) {
		return
	}

	plans, err := db.ListOwnedPlans(projectId, "", true, "")

	if err != nil {
		log.Printf("Error listing plans: %v\n", err)
		http.Error(w, "Error listing plans: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonBytes, err := json.Marshal(plans)
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

	vars := mux.Vars(r)
	projectId := vars["projectId"]

	log.Println("projectId: ", projectId)

	if !authorizeProject(w, projectId, auth.UserId, auth.OrgId) {
		return
	}

	// TODO: implement when status is figured out

}
