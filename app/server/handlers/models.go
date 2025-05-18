package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"plandex-server/db"

	shared "plandex-shared"

	"github.com/gorilla/mux"
)

const CustomModelsMinClientVersion = "2.2.0"

func CreateCustomModelHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreateCustomModelHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	if !requireMinClientVersion(w, r, CustomModelsMinClientVersion) {
		return
	}

	var model shared.CustomModel
	if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
		log.Printf("Error decoding request body: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if model.ModelId == "" {
		msg := "Model id is required"
		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if shared.BuiltInBaseModelsById[model.ModelId] != nil {
		msg := fmt.Sprintf("%s is a built-in base model id, so it can't be used for a custom model", model.ModelId)
		log.Println(msg)
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}

	dbModel := db.CustomModelFromApi(&model)
	dbModel.OrgId = auth.OrgId

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

	if !requireMinClientVersion(w, r, CustomModelsMinClientVersion) {
		return
	}

	id := mux.Vars(r)["modelId"]

	var model shared.CustomModel
	if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
		log.Printf("Error decoding request body: %v\n", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	res, err := db.GetCustomModel(auth.OrgId, id)
	if err != nil {
		log.Printf("Error fetching custom model: %v\n", err)
		http.Error(w, "Failed to fetch custom model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if res == nil {
		http.Error(w, "Custom model not found", http.StatusNotFound)
		return
	}

	dbModel := db.CustomModelFromApi(&model)
	dbModel.Id = id
	dbModel.OrgId = auth.OrgId

	if err := db.UpdateCustomModel(dbModel); err != nil {
		log.Printf("Error updating custom model: %v\n", err)
		http.Error(w, "Failed to update custom model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	log.Println("Successfully updated custom model")
}

func GetCustomModelHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for GetCustomModelHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	id := mux.Vars(r)["modelId"]

	res, err := db.GetCustomModel(auth.OrgId, id)
	if err != nil {
		log.Printf("Error fetching custom model: %v\n", err)
		http.Error(w, "Failed to fetch custom model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if res == nil {
		http.Error(w, "Custom model not found", http.StatusNotFound)
		return
	}

	err = json.NewEncoder(w).Encode(res.ToApi())
	if err != nil {
		log.Printf("Error encoding custom model: %v\n", err)
		http.Error(w, fmt.Sprintf("Error encoding custom model: %v", err), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully fetched custom model")
}

func ListCustomModelsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListCustomModelsHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	if !requireMinClientVersion(w, r, CustomModelsMinClientVersion) {
		return
	}

	models, err := db.ListCustomModels(auth.OrgId)
	if err != nil {
		log.Printf("Error fetching custom models: %v\n", err)
		http.Error(w, "Failed to fetch custom models: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiList []*shared.CustomModel
	for _, m := range models {
		apiList = append(apiList, m.ToApi())
	}

	err = json.NewEncoder(w).Encode(apiList)
	if err != nil {
		log.Printf("Error encoding custom models: %v\n", err)
		http.Error(w, fmt.Sprintf("Error encoding custom models: %v", err), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully fetched custom models")
}

func DeleteCustomModelHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for DeleteAvailableModelHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	if !requireMinClientVersion(w, r, CustomModelsMinClientVersion) {
		return
	}

	id := mux.Vars(r)["modelId"]

	models, err := db.ListCustomModels(auth.OrgId)
	if err != nil {
		log.Printf("Error fetching custom models: %v\n", err)
		http.Error(w, "Failed to fetch custom models: "+err.Error(), http.StatusInternalServerError)
		return
	}

	found := false
	for _, m := range models {
		if m.Id == id {
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Custom model not found", http.StatusNotFound)
		return
	}

	if err := db.DeleteCustomModel(auth.OrgId, id); err != nil {
		log.Printf("Error deleting custom model: %v\n", err)
		http.Error(w, "Failed to delete custom model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	log.Println("Successfully deleted custom model")
}

func CreateCustomProviderHandler(w http.ResponseWriter, r *http.Request) {
	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	if os.Getenv("IS_CLOUD") != "" {
		http.Error(w, "Custom model providers are not supported on Plandex Cloud", http.StatusBadRequest)
		return
	}

	var p shared.CustomProvider
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	dbP := db.CustomProviderFromApi(&p)
	dbP.OrgId = auth.OrgId

	if err := db.CreateCustomProvider(dbP); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)

	log.Println("Successfully created custom provider")
}

func UpdateCustomProviderHandler(w http.ResponseWriter, r *http.Request) {
	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	if os.Getenv("IS_CLOUD") != "" {
		http.Error(w, "Custom model providers are not supported on Plandex Cloud", http.StatusBadRequest)
		return
	}

	id := mux.Vars(r)["providerId"]

	var p shared.CustomProvider
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	p.Id = id

	dbP := db.CustomProviderFromApi(&p)
	dbP.OrgId = auth.OrgId

	if err := db.UpdateCustomProvider(dbP); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

	log.Println("Successfully updated custom provider")
}

func GetCustomProviderHandler(w http.ResponseWriter, r *http.Request) {
	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	id := mux.Vars(r)["providerId"]

	res, err := db.GetCustomProvider(auth.OrgId, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(res.ToApi())
	if err != nil {
		log.Printf("Error encoding custom provider: %v\n", err)
		http.Error(w, fmt.Sprintf("Error encoding custom provider: %v", err), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully fetched custom provider")
}

func ListCustomProvidersHandler(w http.ResponseWriter, r *http.Request) {
	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	if os.Getenv("IS_CLOUD") != "" {
		http.Error(w, "Custom model providers are not supported on Plandex Cloud", http.StatusBadRequest)
		return
	}

	list, err := db.ListCustomProviders(auth.OrgId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var apiList []*shared.CustomProvider
	for _, p := range list {
		apiList = append(apiList, p.ToApi())
	}

	err = json.NewEncoder(w).Encode(apiList)
	if err != nil {
		log.Printf("Error encoding custom providers: %v\n", err)
		http.Error(w, fmt.Sprintf("Error encoding custom providers: %v", err), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully fetched custom providers")
}

func DeleteCustomProviderHandler(w http.ResponseWriter, r *http.Request) {
	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	if os.Getenv("IS_CLOUD") != "" {
		http.Error(w, "Custom model providers are not supported on Plandex Cloud", http.StatusBadRequest)
		return
	}

	id := mux.Vars(r)["providerId"]
	if err := db.DeleteCustomProvider(auth.OrgId, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

	log.Println("Successfully deleted custom provider")
}

func CreateModelPackHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreateModelPackHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	if !requireMinClientVersion(w, r, CustomModelsMinClientVersion) {
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

	if !requireMinClientVersion(w, r, CustomModelsMinClientVersion) {
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

	if !requireMinClientVersion(w, r, CustomModelsMinClientVersion) {
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

	if !requireMinClientVersion(w, r, CustomModelsMinClientVersion) {
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
