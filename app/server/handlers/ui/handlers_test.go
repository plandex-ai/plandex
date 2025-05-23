package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	// Adjust import path if your project root is different or aliased.
	// This assumes Plandex is the root of the Go module.
	// If your go.mod defines a module like "github.com/plandex-ai/plandex",
	// then imports should be "github.com/plandex-ai/plandex/app/server/utils"
	// For now, using relative paths for utils if it's a local package.
	// If utils is in a different module, direct import is needed.
	// Assuming utils is structured such that it can be imported.
	// "github.com/plandex-ai/plandex/app/server/utils"
	// For the purpose of this test, we might not need utils.Log if errors are checked directly.

	"github.com/stretchr/testify/assert"
)

// TestMain can be used for setup/teardown if needed, for now, it's not required.

// Helper function to create a new request and response recorder for handler testing.
func newTestRequest(method, target string) (*http.Request, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, target, nil)
	rr := httptest.NewRecorder()
	return req, rr
}

// TestDashboardHandler tests the DashboardHandler.
func TestDashboardHandler(t *testing.T) {
	req, rr := newTestRequest(http.MethodGet, "/ui/dashboard")
	DashboardHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "DashboardHandler returned wrong status code")
	body := rr.Body.String()
	assert.Contains(t, body, "<title>Dashboard - Plandex</title>", "Response body should contain the correct title")
	assert.Contains(t, body, "<h1>Dashboard Overview</h1>", "Response body should contain the page title")
	assert.Contains(t, body, "Quick Stats", "Response body should contain dashboard content")
}

// TestPlansHandler tests the PlansHandler.
func TestPlansHandler(t *testing.T) {
	req, rr := newTestRequest(http.MethodGet, "/ui/plans")
	PlansHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "PlansHandler returned wrong status code")
	body := rr.Body.String()
	assert.Contains(t, body, "<title>Plans - Plandex</title>", "Response body should contain the correct title")
	assert.Contains(t, body, "<h1>Plans</h1>", "Response body should contain the page title")
	assert.Contains(t, body, "Manage your plans here. Coming soon!", "Response body should contain placeholder content")
}

// TestContextsHandler tests the ContextsHandler.
func TestContextsHandler(t *testing.T) {
	req, rr := newTestRequest(http.MethodGet, "/ui/contexts")
	ContextsHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "ContextsHandler returned wrong status code")
	body := rr.Body.String()
	assert.Contains(t, body, "<title>Contexts - Plandex</title>", "Response body should contain the correct title")
	assert.Contains(t, body, "<h1>Contexts</h1>", "Response body should contain the page title")
	assert.Contains(t, body, "Manage your contexts here. Coming soon!", "Response body should contain placeholder content")
}

// TestModelsHandler tests the ModelsHandler.
func TestModelsHandler(t *testing.T) {
	req, rr := newTestRequest(http.MethodGet, "/ui/models")
	ModelsHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "ModelsHandler returned wrong status code")
	body := rr.Body.String()
	assert.Contains(t, body, "<title>Models - Plandex</title>", "Response body should contain the correct title")
	assert.Contains(t, body, "<h1>Models</h1>", "Response body should contain the page title")
	assert.Contains(t, body, "Manage your AI models here. Coming soon!", "Response body should contain placeholder content")
}

// TestMCPHandler tests the MCPHandler.
// This test will need to be updated if MCPHandler starts fetching real data.
func TestMCPHandler(t *testing.T) {
	req, rr := newTestRequest(http.MethodGet, "/ui/mcp")
	MCPHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "MCPHandler returned wrong status code")
	body := rr.Body.String()
	assert.Contains(t, body, "<title>MCP - Plandex</title>", "Response body should contain the correct title")
	assert.Contains(t, body, "<h1>Multi-Channel Publishing</h1>", "Response body should contain the page title")
	assert.Contains(t, body, "Select Channel:", "Response body should contain MCP form elements")
}

// TestSettingsHandler tests the SettingsHandler.
func TestSettingsHandler(t *testing.T) {
	req, rr := newTestRequest(http.MethodGet, "/ui/settings")
	SettingsHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "SettingsHandler returned wrong status code")
	body := rr.Body.String()
	assert.Contains(t, body, "<title>Settings - Plandex</title>", "Response body should contain the correct title")
	assert.Contains(t, body, "<h1>Settings</h1>", "Response body should contain the page title")
	assert.Contains(t, body, "Configure Plandex settings here. Coming soon!", "Response body should contain placeholder content")
}

// Note: For these tests to run correctly, the template files need to be accessible
// relative to the execution path of the test. Go tests are typically run from the
// package directory (e.g., app/server/handlers/ui/).
// The template paths in handlers.go are like "app/server/ui/templates/layout.html".
// If tests are run from the 'ui' package directory, paths like "../../../app/server/ui/templates/layout.html"
// might be needed, or a more robust path resolution mechanism (e.g., using build tags, environment variables, or a test helper to set a base path).
// For now, assuming the default behavior of `go test` from the package directory and that
// the relative paths work out or that the PWD for the test execution makes these paths valid.
// A common solution is to have a 'testdata' directory or to adjust paths during tests.
// The current template parsing logic in `handlers.go` uses paths relative to the project root.
// We might need to adjust file paths in `renderPageTemplate` or how tests are run.
// One way is to change directory in tests:
// wd, _ := os.Getwd()
// fmt.Println("Current working directory for test:", wd)
// And ensure that 'app/server/ui/templates/...' is found from there.
// If `go.mod` is at `plandex/go.mod`, and tests are in `plandex/app/server/handlers/ui`,
// paths starting `app/server/ui/templates` should work if tests are run from project root.
// If tests run from package dir, then `../../../app/server/ui/templates` would be required.
// Let's assume for now tests are run from project root or paths are correctly resolved.
// The `filepath.Join("app", "server", "ui", "templates", pageName)` in `handlers.go`
// implies that the process running the code has "app/server/ui/templates" in its current view.
// This is often true if you run `go test` from the root of your module.
// If you run `go test ./app/server/handlers/ui/...` from the root, it should work.

// A more robust way to handle file paths for templates in tests:
// - Copy templates to a temporary directory or a test-specific path.
// - Override the template parsing function or path variable during tests.
// For now, we'll rely on the standard `go test` behavior from the module root.

// TestRenderPageTemplate_ErrorHandling (Example for testing error paths, if needed)
// func TestRenderPageTemplate_NonExistentTemplate(t *testing.T) {
// 	rr := httptest.NewRecorder()
// 	data := PageData{Title: "Error Test", PageTitle: "Error Test", ActiveNav: "error"}
// 	// Assuming "non_existent_page.html" does not exist
// 	renderPageTemplate(rr, "non_existent_page.html", data)
//
// 	assert.Equal(t, http.StatusInternalServerError, rr.Code, "Expected internal server error for non-existent template")
// 	assert.Contains(t, rr.Body.String(), "Error parsing templates", "Error message should indicate template parsing issue")
// }

// Add tests for renderBasicPage if it's still used significantly and has complex logic.
// Currently, it's simple enough that testing the handlers covers its usage.
// If renderBasicPage had more complex logic, direct tests would be beneficial.
// For example, TestRenderBasicPage_Success and TestRenderBasicPage_LayoutError.
```
