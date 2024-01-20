package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/plandex/plandex/shared"
)

func writeApiError(w http.ResponseWriter, apiErr shared.ApiError) {
	bytes, err := json.Marshal(apiErr)
	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		// If marshalling fails, fall back to a simpler error message
		http.Error(w, "Error marshalling response", http.StatusInternalServerError)
		return
	}

	log.Printf("API Error: %v\n", apiErr.Msg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Status)

	_, writeErr := w.Write(bytes)
	if writeErr != nil {
		log.Printf("Error writing response: %v\n", writeErr)
	}
}
