package mcp

import (
	"encoding/json"
	"net/http"

	"github.com/plandex-ai/plandex/app/server/mcp" // Import the new mcp logic package
	// "github.com/plandex-ai/plandex/app/server/utils" // Commented out for now
)

// MCPGetStatusHandler handles requests for MCP status.
func MCPGetStatusHandler(w http.ResponseWriter, r *http.Request) {
	status := mcp.GetStatus()
	response := map[string]string{"status": status}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		// utils.Log.Errorf("Error encoding MCP status response: %v", err) // Commented out
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
