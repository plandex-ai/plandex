package handlers
import (
	"encoding/json"
	"net/http"
	"plandex-server/db"
	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
)

// CreateCustomModelHandler handles the creation of a new custom model.
func CreateCustomModelHandler(w http.ResponseWriter, r *http.Request) {
	var model shared.CustomModel
	if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := db.CreateCustomModel(r.Context(), model); err != nil {
		http.Error(w, "Failed to create custom model", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// ListCustomModelsHandler handles listing all custom models.
func ListCustomModelsHandler(w http.ResponseWriter, r *http.Request) {
	orgId := r.URL.Query().Get("orgId")
	models, err := db.ListCustomModels(r.Context(), orgId)
	if err != nil {
		http.Error(w, "Failed to fetch custom models", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(models)
}

// DeleteCustomModelHandler handles the deletion of a custom model.
func DeleteCustomModelHandler(w http.ResponseWriter, r *http.Request) {
	modelId := mux.Vars(r)["modelId"]
	if err := db.DeleteCustomModel(r.Context(), modelId); err != nil {
		http.Error(w, "Failed to delete custom model", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// CreateModelSetHandler handles the creation of a new model set.
func CreateModelSetHandler(w http.ResponseWriter, r *http.Request) {
	var set shared.ModelSet
	if err := json.NewDecoder(r.Body).Decode(&set); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := db.CreateModelSet(r.Context(), set); err != nil {
		http.Error(w, "Failed to create model set", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// ListModelSetsHandler handles listing all model sets.
func ListModelSetsHandler(w http.ResponseWriter, r *http.Request) {
	orgId := r.URL.Query().Get("orgId")
	sets, err := db.ListModelSets(r.Context(), orgId)
	if err != nil {
		http.Error(w, "Failed to fetch model sets", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(sets)
}

// DeleteModelSetHandler handles the deletion of a model set.
func DeleteModelSetHandler(w http.ResponseWriter, r *http.Request) {
	setId := mux.Vars(r)["setId"]
	if err := db.DeleteModelSet(r.Context(), setId); err != nil {
		http.Error(w, "Failed to delete model set", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

