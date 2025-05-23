package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	// Adjust import paths as necessary, similar to ui/handlers_test.go
	// "github.com/plandex-ai/plandex/app/server/utils"

	"github.com/stretchr/testify/assert"
)

// Helper function to create a new request and response recorder for handler testing.
func newTestRequest(method, target string) (*http.Request, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, target, nil)
	rr := httptest.NewRecorder()
	return req, rr
}

// TestMCPGetStatusHandler tests the MCPGetStatusHandler.
func TestMCPGetStatusHandler(t *testing.T) {
	req, rr := newTestRequest(http.MethodGet, "/api/mcp/status") // The exact path doesn't matter for handler unit test
	MCPGetStatusHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "MCPGetStatusHandler returned wrong status code")
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"), "MCPGetStatusHandler did not set Content-Type to application/json")

	var response map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err, "Error unmarshalling JSON response from MCPGetStatusHandler")

	expectedStatus := "MCP module is active but not yet configured"
	assert.Equal(t, expectedStatus, response["status"], "MCPGetStatusHandler returned unexpected status in JSON response")
}

// Note: Similar to UI handler tests, if the MCPGetStatusHandler relied on more complex
// dependencies that needed initialization (e.g., database, config), those would need
// to be mocked or set up for the test environment. Currently, it's self-contained.
```
