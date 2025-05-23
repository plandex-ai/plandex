package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetStatus tests the GetStatus function from mcp.go.
func TestGetStatus(t *testing.T) {
	expectedStatus := "MCP module is active but not yet configured"
	actualStatus := GetStatus()

	assert.Equal(t, expectedStatus, actualStatus, "GetStatus() returned unexpected status string")
}
