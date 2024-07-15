package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"plandex-server/db"

	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
)

func CreateCustomModelHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreateCustomModelHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	var model shared.AvailableModel
	if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
		log.Printf("Error decoding request body: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	dbModel := &db.AvailableModel{
		Id:                          model.Id,
		OrgId:                       auth.OrgId,
		Provider:                    model.Provider,
		CustomProvider:              model.CustomProvider,
		BaseUrl:                     model.BaseUrl,
		ModelName:                   model.ModelName,
		MaxTokens:                   model.MaxTokens,
		ApiKeyEnvVar:                model.ApiKeyEnvVar,
		IsOpenAICompatible:          model.IsOpenAICompatible,
		HasJsonResponseMode:         model.HasJsonResponseMode,
		HasStreaming:                model.HasStreaming,
		HasFunctionCalling:          model.HasFunctionCalling,
		HasStreamingFunctionCalls:   model.HasStreamingFunctionCalls,
		DefaultMaxConvoTokens:       model.DefaultMaxConvoTokens,
		DefaultReservedOutputTokens: model.DefaultReservedOutputTokens,
	}

	if err := db.CreateCustomModel(dbModel); err != nil {
		log.Printf("Error creating custom model: %v\n", err)
		http.Error(w, "Failed to create custom model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	log.Println("Successfully created custom model")
}

func ListCustomModelsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListCustomModelsHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	models, err := db.ListCustomModels(auth.OrgId)
	if err != nil {
		log.Printf("Error fetching custom models: %v\n", err)
		http.Error(w, "Failed to fetch custom models: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(models)

	log.Println("Successfully fetched custom models")
}

func DeleteAvailableModelHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for DeleteAvailableModelHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	modelId := mux.Vars(r)["modelId"]
	if err := db.DeleteAvailableModel(modelId); err != nil {
		log.Printf("Error deleting custom model: %v\n", err)
		http.Error(w, "Failed to delete custom model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	log.Println("Successfully deleted custom model")
}

func CreateModelPackHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreateModelPackHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	var ms shared.ModelPack
	if err := json.NewDecoder(r.Body).Decode(&ms); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	dbMs := &db.ModelPack{
		OrgId:       auth.OrgId,
		Name:        ms.Name,
		Description: ms.Description,
		Planner:     ms.Planner,
		PlanSummary: ms.PlanSummary,
		Builder:     ms.Builder,
		Namer:       ms.Namer,
		CommitMsg:   ms.CommitMsg,
		ExecStatus:  ms.ExecStatus,
	}

	if err := db.CreateModelPack(dbMs); err != nil {
		log.Printf("Error creating model pack: %v\n", err)
		http.Error(w, "Failed to create model pack: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	log.Println("Successfully created model pack")
}

func ListModelPacksHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListModelPacksHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	sets, err := db.ListModelPacks(auth.OrgId)
	if err != nil {
		log.Printf("Error fetching model packs: %v\n", err)
		http.Error(w, "Failed to fetch model packs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiPacks []*shared.ModelPack

	for _, mp := range sets {
		apiPacks = append(apiPacks, mp.ToApi())
	}

	json.NewEncoder(w).Encode(apiPacks)

	log.Println("Successfully fetched model packs")
}

func DeleteModelPackHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for DeleteModelPackHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	setId := mux.Vars(r)["setId"]

	log.Printf("Deleting model pack with id: %s\n", setId)

	if err := db.DeleteModelPack(setId); err != nil {
		log.Printf("Error deleting model pack: %v\n", err)
		http.Error(w, "Failed to delete model pack: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	log.Println("Successfully deleted model pack")
}
