package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plandex-server/syntax/file_map"

	"github.com/plandex/plandex/shared"
)

func GetFileMapHandler(w http.ResponseWriter, r *http.Request) {
	var req shared.GetFileMapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding request: %v", err), http.StatusBadRequest)
		return
	}

	maps := make(shared.FileMapBodies)
	for path, input := range req.MapInputs {
		fileMap, err := file_map.MapFile(r.Context(), path, []byte(input))
		if err != nil {
			// Skip files that can't be parsed, just log the error
			log.Printf("Error mapping file %s: %v", path, err)
			continue
		}
		maps[path] = fileMap.String()
	}

	resp := shared.GetFileMapResponse{
		MapBodies: maps,
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error marshalling response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}
