package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"plandex-server/db"

	shared "plandex-shared"

	"github.com/gorilla/mux"
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

	if os.Getenv("IS_CLOUD") != "" && model.Provider == shared.ModelProviderCustom {
		http.Error(w, "Custom model providers are not supported on Plandex Cloud", http.StatusBadRequest)
		return
	}

	baseModelConfig := model.BaseModelConfig

	dbModel := &db.AvailableModel{
		Id:                    model.Id,
		OrgId:                 auth.OrgId,
		Provider:              baseModelConfig.Provider,
		CustomProvider:        baseModelConfig.CustomProvider,
		BaseUrl:               baseModelConfig.BaseUrl,
		ModelId:               baseModelConfig.ModelId,
		ModelName:             baseModelConfig.ModelName,
		Description:           model.Description,
		MaxTokens:             baseModelConfig.MaxTokens,
		ApiKeyEnvVar:          baseModelConfig.ApiKeyEnvVar,
		HasImageSupport:       baseModelConfig.ModelCompatibility.HasImageSupport,
		DefaultMaxConvoTokens: model.DefaultMaxConvoTokens,
		MaxOutputTokens:       baseModelConfig.MaxOutputTokens,
		ReservedOutputTokens:  baseModelConfig.ReservedOutputTokens,
		PreferredOutputFormat: baseModelConfig.PreferredModelOutputFormat,
	}

	if err := db.CreateCustomModel(dbModel); err != nil {
		log.Printf("Error creating custom model: %v\n", err)
		http.Error(w, "Failed to create custom model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	log.Println("Successfully created custom model")
}

func UpdateCustomModelHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for UpdateCustomModelHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	modelId := mux.Vars(r)["modelId"]

	var model shared.AvailableModel
	if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
		log.Printf("Error decoding request body: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	models, err := db.ListCustomModels(auth.OrgId)
	if err != nil {
		log.Printf("Error fetching custom models: %v\n", err)
		http.Error(w, "Failed to fetch custom models: "+err.Error(), http.StatusInternalServerError)
		return
	}

	found := false
	for _, m := range models {
		if m.Id == modelId {
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Custom model not found", http.StatusNotFound)
		return
	}

	if os.Getenv("IS_CLOUD") != "" && model.Provider == shared.ModelProviderCustom {
		http.Error(w, "Custom model providers are not supported on Plandex Cloud", http.StatusBadRequest)
		return
	}

	dbModel := &db.AvailableModel{
		Id:                    modelId,
		OrgId:                 auth.OrgId,
		Provider:              model.BaseModelConfig.Provider,
		CustomProvider:        model.BaseModelConfig.CustomProvider,
		BaseUrl:               model.BaseModelConfig.BaseUrl,
		ModelId:               model.BaseModelConfig.ModelId,
		ModelName:             model.BaseModelConfig.ModelName,
		Description:           model.Description,
		MaxTokens:             model.BaseModelConfig.MaxTokens,
		ApiKeyEnvVar:          model.BaseModelConfig.ApiKeyEnvVar,
		HasImageSupport:       model.BaseModelConfig.ModelCompatibility.HasImageSupport,
		DefaultMaxConvoTokens: model.DefaultMaxConvoTokens,
		MaxOutputTokens:       model.BaseModelConfig.MaxOutputTokens,
		ReservedOutputTokens:  model.BaseModelConfig.ReservedOutputTokens,
		PreferredOutputFormat: model.BaseModelConfig.PreferredModelOutputFormat,
	}

	if err := db.UpdateCustomModel(dbModel); err != nil {
		log.Printf("Error updating custom model: %v\n", err)
		http.Error(w, "Failed to update custom model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	log.Println("Successfully updated custom model")
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

	models, err := db.ListCustomModels(auth.OrgId)
	if err != nil {
		log.Printf("Error fetching custom models: %v\n", err)
		http.Error(w, "Failed to fetch custom models: "+err.Error(), http.StatusInternalServerError)
		return
	}

	found := false
	for _, m := range models {
		if m.Id == modelId {
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Custom model not found", http.StatusNotFound)
		return
	}

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
		OrgId:            auth.OrgId,
		Name:             ms.Name,
		Description:      ms.Description,
		Planner:          ms.Planner,
		Architect:        ms.Architect,
		Coder:            ms.Coder,
		Builder:          ms.Builder,
		WholeFileBuilder: ms.WholeFileBuilder,
		Namer:            ms.Namer,
		CommitMsg:        ms.CommitMsg,
		PlanSummary:      ms.PlanSummary,
		ExecStatus:       ms.ExecStatus,
	}

	if err := db.CreateModelPack(dbMs); err != nil {
		log.Printf("Error creating model pack: %v\n", err)
		http.Error(w, "Failed to create model pack: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	log.Println("Successfully created model pack")
}

func UpdateModelPackHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for UpdateModelPackHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	mpId := mux.Vars(r)["setId"]

	var ms shared.ModelPack
	if err := json.NewDecoder(r.Body).Decode(&ms); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	packs, err := db.ListModelPacks(auth.OrgId)
	if err != nil {
		log.Printf("Error fetching model packs: %v\n", err)
		http.Error(w, "Failed to fetch model packs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	found := false
	for _, m := range packs {
		if m.Id == mpId {
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Model pack not found", http.StatusNotFound)
		return
	}

	dbMs := &db.ModelPack{
		Id:               mpId,
		OrgId:            auth.OrgId,
		Name:             ms.Name,
		Description:      ms.Description,
		Planner:          ms.Planner,
		Coder:            ms.Coder,
		PlanSummary:      ms.PlanSummary,
		Builder:          ms.Builder,
		WholeFileBuilder: ms.WholeFileBuilder,
		Namer:            ms.Namer,
		CommitMsg:        ms.CommitMsg,
		ExecStatus:       ms.ExecStatus,
		Architect:        ms.Architect,
	}

	if err := db.UpdateModelPack(dbMs); err != nil {
		log.Printf("Error updating model pack: %v\n", err)
		http.Error(w, "Failed to update model pack: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	log.Println("Successfully updated model pack")
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

	mpId := mux.Vars(r)["setId"]

	packs, err := db.ListModelPacks(auth.OrgId)
	if err != nil {
		log.Printf("Error fetching model packs: %v\n", err)
		http.Error(w, "Failed to fetch model packs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	found := false
	for _, m := range packs {
		if m.Id == mpId {
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Model pack not found", http.StatusNotFound)
		return
	}

	log.Printf("Deleting model pack with id: %s\n", mpId)

	if err := db.DeleteModelPack(mpId); err != nil {
		log.Printf("Error deleting model pack: %v\n", err)
		http.Error(w, "Failed to delete model pack: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	log.Println("Successfully deleted model pack")
}
