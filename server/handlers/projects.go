package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"plandex-server/db"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
)

func CreateProjectHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreateProjectHandler")

	currentOrgId := "2ff5bc12-1160-4305-8707-9a165319de5a" // TODO: get from auth when implemented

	// read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var requestBody shared.CreateProjectRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	if requestBody.Name == "" {
		log.Println("Received empty name field")
		http.Error(w, "name field is required", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()

	_, err = db.Conn.Exec("INSERT INTO projects (id, org_id, name) VALUES ($1, $2, $3)", id, currentOrgId, requestBody.Name)

	if err != nil {
		log.Printf("Error creating project: %v\n", err)
		http.Error(w, "Error creating project: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := shared.CreateProjectResponse{
		Id: id,
	}

	bytes, err := json.Marshal(resp)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(bytes)

	log.Println("Successfully created project", id)

}

func ListProjectsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListProjectsHandler")

	userId := "bc9c75ee-57b0-4552-aa1b-f80cf8c09f3f" // TODO: get from auth when implemented

	rows, err := db.Conn.Query("SELECT project.id, project.name FROM users_projects INNER JOIN project ON users_projects.project_id = project.id WHERE users_projects.user_id = $1", userId)

	if err != nil {
		log.Printf("Error listing projects: %v\n", err)
		http.Error(w, "Error listing projects: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var projects []shared.Project

	for rows.Next() {
		var project shared.Project
		err := rows.Scan(&project.Id, &project.Name)
		if err != nil {
			log.Printf("Error scanning project: %v\n", err)
			http.Error(w, "Error scanning project: "+err.Error(), http.StatusInternalServerError)
			return
		}
		projects = append(projects, project)
	}

	bytes, err := json.Marshal(projects)
	if err != nil {
		log.Printf("Error marshalling projects: %v\n", err)
		http.Error(w, "Error marshalling projects: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(bytes)
}

func ProjectSetPlanHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for UpdateProjectSetPlanHandler")

	vars := mux.Vars(r)
	projectId := vars["projectId"]

	log.Println("projectId: ", projectId)

	// read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var requestBody shared.SetProjectPlanRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	if requestBody.PlanId == "" {
		log.Println("Received empty planId field")
		http.Error(w, "planId field is required", http.StatusBadRequest)
		return
	}

	// update statement here -- need auth / current user id

	if err != nil {
		log.Printf("Error updating project: %v\n", err)
		http.Error(w, "Error updating project: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully set project plan", projectId)
}

func RenameProjectHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for RenameProjectHandler")

	vars := mux.Vars(r)
	projectId := vars["projectId"]

	log.Println("projectId: ", projectId)

	// read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var requestBody shared.RenameProjectRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	if requestBody.Name == "" {
		log.Println("Received empty name field")
		http.Error(w, "name field is required", http.StatusBadRequest)
		return
	}

	res, err := db.Conn.Exec("UPDATE projects SET name = $1 WHERE id = $2", requestBody.Name, projectId)

	if err != nil {
		log.Printf("Error updating project: %v\n", err)
		http.Error(w, "Error updating project: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := res.RowsAffected()

	if err != nil {
		log.Printf("Error getting rows affected: %v\n", err)
		http.Error(w, "Error getting rows affected: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		log.Printf("Project not found: %v\n", projectId)
		http.Error(w, "Project not found: "+projectId, http.StatusNotFound)
		return
	}

	log.Println("Successfully renamed project", projectId)

}
