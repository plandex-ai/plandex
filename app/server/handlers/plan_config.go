package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"plandex-server/db"

	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
)

func GetPlanConfigHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for GetPlanConfigHandler")

	auth := Authenticate(w, r, true)
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

	config, err := db.GetPlanConfig(planId)
	if err != nil {
		log.Println("Error getting plan config: ", err)
		http.Error(w, "Error getting plan config", http.StatusInternalServerError)
		return
	}

	res := shared.GetPlanConfigResponse{
		Config: config,
	}

	bytes, err := json.Marshal(res)
	if err != nil {
		log.Println("Error marshalling response: ", err)
		http.Error(w, "Error marshalling response", http.StatusInternalServerError)
		return
	}

	w.Write(bytes)
	log.Println("GetPlanConfigHandler processed successfully")
}

func UpdatePlanConfigHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for UpdatePlanConfigHandler")

	auth := Authenticate(w, r, true)
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

	var req shared.UpdatePlanConfigRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Println("Error decoding request body: ", err)
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	err = db.StorePlanConfig(planId, req.Config)
	if err != nil {
		log.Println("Error storing plan config: ", err)
		http.Error(w, "Error storing plan config", http.StatusInternalServerError)
		return
	}

	res := shared.UpdatePlanConfigResponse{
		Msg: "Successfully updated plan config",
	}

	bytes, err := json.Marshal(res)
	if err != nil {
		log.Println("Error marshalling response: ", err)
		http.Error(w, "Error marshalling response", http.StatusInternalServerError)
		return
	}

	w.Write(bytes)
	log.Println("UpdatePlanConfigHandler processed successfully")
}

func GetDefaultPlanConfigHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for GetDefaultPlanConfigHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	config, err := db.GetDefaultPlanConfig(auth.UserId)
	if err != nil {
		log.Println("Error getting default plan config: ", err)
		http.Error(w, "Error getting default plan config", http.StatusInternalServerError)
		return
	}

	res := shared.GetDefaultPlanConfigResponse{
		Config: config,
	}

	bytes, err := json.Marshal(res)
	if err != nil {
		log.Println("Error marshalling response: ", err)
		http.Error(w, "Error marshalling response", http.StatusInternalServerError)
		return
	}

	w.Write(bytes)
	log.Println("GetDefaultPlanConfigHandler processed successfully")
}

func UpdateDefaultPlanConfigHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for UpdateDefaultPlanConfigHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	var req shared.UpdateDefaultPlanConfigRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Println("Error decoding request body: ", err)
		http.Error(w, "Error decoding request body", http.StatusBadRequest)
		return
	}

	tx, err := db.Conn.Beginx()
	if err != nil {
		log.Println("Error starting transaction: ", err)
		http.Error(w, "Error starting transaction", http.StatusInternalServerError)
		return
	}

	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("Error rolling back transaction: %v\n", rbErr)
			}
		}
	}()

	err = db.StoreDefaultPlanConfig(auth.UserId, req.Config, tx)
	if err != nil {
		log.Println("Error storing default plan config: ", err)
		http.Error(w, "Error storing default plan config", http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Println("Error committing transaction: ", err)
		http.Error(w, "Error committing transaction", http.StatusInternalServerError)
		return
	}

	res := shared.UpdateDefaultPlanConfigResponse{
		Msg: "Successfully updated default plan config",
	}

	bytes, err := json.Marshal(res)
	if err != nil {
		log.Println("Error marshalling response: ", err)
		http.Error(w, "Error marshalling response", http.StatusInternalServerError)
		return
	}

	w.Write(bytes)
	log.Println("UpdateDefaultPlanConfigHandler processed successfully")
}
