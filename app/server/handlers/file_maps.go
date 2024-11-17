package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"plandex-server/syntax"

	"github.com/plandex/plandex/shared"
)

func GetFileMapHandler(w http.ResponseWriter, r *http.Request) {
	var req shared.GetFileMapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding request: %v", err), http.StatusBadRequest)
		return
	}

	maps := make(shared.FileMapBodies)
	for filepath, content := range req.Files {
		fileMap, err := syntax.MapFile(r.Context(), filepath, []byte(content))
		if err != nil {
			// Skip files that can't be parsed, just log the error
			continue
		}
		maps[filepath] = fileMap.String()
	}

	resp := shared.GetFileMapResponse{
		Maps: maps,
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error marshalling response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}
