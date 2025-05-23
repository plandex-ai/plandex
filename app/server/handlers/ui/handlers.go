package ui

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	// "github.com/plandex-ai/plandex/app/server/utils" // Commented out for now
)

// PageData holds the data to be passed to the template.
type PageData struct {
	Title     string      // Title of the HTML page
	PageTitle string      // Title to be displayed in the main content header
	ActiveNav string      // Identifier for the active navigation link
	Content   interface{} // Data specific to the page content template
	// Add any other common data needed by the layout or pages
}

// renderPageTemplate parses and executes the specified HTML templates.
// It renders a specific page template (e.g., "dashboard.html") within the base "layout.html" template.
func renderPageTemplate(w http.ResponseWriter, pageName string, data PageData) {
	// Construct file paths for layout and the specific page template
	// Ensure your templates are in the 'app/server/ui/templates' directory
	baseLayout := filepath.Join("app", "server", "ui", "templates", "layout.html")
	pageFile := filepath.Join("app", "server", "ui", "templates", pageName)

	// Parse the templates. layout.html must be parsed first.
	tmpl, err := template.ParseFiles(baseLayout, pageFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing templates: %v", err), http.StatusInternalServerError)
		// utils.Log.Errorf("Error parsing templates (%s, %s): %v", baseLayout, pageFile, err) // Commented out
		return
	}

	// Execute the template, providing the data.
	// The name of the template to execute is "layout.html" as it's the entry point.
	// The "content" template defined in dashboard.html (or other pages) will be called from within layout.html.
	err = tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error executing template: %v", err), http.StatusInternalServerError)
		// utils.Log.Errorf("Error executing template for %s: %v", pageName, err) // Commented out
	}
}

// DashboardHandler serves the dashboard page using templates.
func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title:     "Dashboard",
		PageTitle: "Dashboard Overview",
		ActiveNav: "dashboard",
		Content:   nil, // No specific content data for dashboard.html's static content for now
	}
	renderPageTemplate(w, "dashboard.html", data)
}

// PlansHandler serves the plans page.
func PlansHandler(w http.ResponseWriter, r *http.Request) {
	// For now, using a simple string content until plans.html is created
	renderBasicPage(w, "Plans", "Plans", "plans", "<h1>Plans</h1><p>Manage your plans here. Coming soon!</p>")
}

// ContextsHandler serves the contexts page.
func ContextsHandler(w http.ResponseWriter, r *http.Request) {
	renderBasicPage(w, "Contexts", "Contexts", "contexts", "<h1>Contexts</h1><p>Manage your contexts here. Coming soon!</p>")
}

// ModelsHandler serves the models page.
func ModelsHandler(w http.ResponseWriter, r *http.Request) {
	renderBasicPage(w, "Models", "Models", "models", "<h1>Models</h1><p>Manage your AI models here. Coming soon!</p>")
}

// MCPHandler serves the MCP page.
func MCPHandler(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title:     "MCP",
		PageTitle: "Multi-Channel Publishing",
		ActiveNav: "mcp",
		Content:   nil, // mcp.html is self-contained for now
	}
	renderPageTemplate(w, "mcp.html", data)
}

// SettingsHandler serves the settings page.
func SettingsHandler(w http.ResponseWriter, r *http.Request) {
	renderBasicPage(w, "Settings", "Settings", "settings", "<h1>Settings</h1><p>Configure Plandex settings here. Coming soon!</p>")
}

// renderBasicPage is a helper for pages not yet converted to full templates.
// It uses an inline template for now but still leverages the PageData for consistency.
func renderBasicPage(w http.ResponseWriter, htmlTitle, pageTitle, activeNav, contentHTML string) {
	baseLayout := filepath.Join("app", "server", "ui", "templates", "layout.html")
	
	// Define a minimal content template string
	contentTmplStr := `{{define "content"}}` + contentHTML + `{{end}}`

	// Parse the base layout
	layoutTpl, err := template.ParseFiles(baseLayout)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing layout template: %v", err), http.StatusInternalServerError)
		// utils.Log.Errorf("Error parsing layout template: %v", err) // Commented out
		return
	}

	// Parse the content template string and add it to the layout's template set
	tmpl, err := layoutTpl.New("content").Parse(contentTmplStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing content template string: %v", err), http.StatusInternalServerError)
		// utils.Log.Errorf("Error parsing content template string: %v", err) // Commented out
		return
	}
	
	data := PageData{
		Title:     htmlTitle,
		PageTitle: pageTitle,
		ActiveNav: activeNav,
	}

	err = tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error executing basic page template: %v", err), http.StatusInternalServerError)
		// utils.Log.Errorf("Error executing basic page template for %s: %v", pageTitle, err) // Commented out
	}
}
