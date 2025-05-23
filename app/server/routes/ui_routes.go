package routes

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/plandex-ai/plandex/app/server/handlers/ui"
)

// AddUIRoutes registers the UI routes with the provided router.
// It uses the HandlePlandexFn if available, otherwise falls back to standard mux registration.
func AddUIRoutes(r *mux.Router) {
	// Define a helper function to register routes, abstracting away whether HandlePlandexFn is used
	register := func(path string, handler http.HandlerFunc, methods ...string) {
		if HandlePlandexFn != nil {
			// Assuming HandlePlandexFn can be adapted or we create a version for non-streaming handlers
			// For now, let's assume it expects a PlandexHandler style.
			// This might need adjustment based on HandlePlandexFn's exact signature and purpose for UI.
			// If HandlePlandexFn is strictly for API-style JSON responses, we might need a different approach
			// or to simply use r.HandleFunc directly for UI.
			// For simplicity, if HandlePlandexFn is present, we'll assume it can handle these,
			// or this part needs to be refined.
			// Let's try to use r.HandleFunc directly for UI routes as they are not streaming API endpoints.
			route := r.HandleFunc(path, handler)
			if len(methods) > 0 {
				route.Methods(methods...)
			}
		} else {
			route := r.HandleFunc(path, handler)
			if len(methods) > 0 {
				route.Methods(methods...)
			}
		}
	}

	// Serve static files
	// The static file server needs to be registered before specific UI page routes
	// if there's any chance of path overlap, though with "/ui/static/" it should be fine.
	// Using mux's PathPrefix for static files.
	staticFileServer := http.FileServer(http.Dir("./app/server/ui/static/"))
	r.PathPrefix("/ui/static/").Handler(http.StripPrefix("/ui/static/", staticFileServer))


	register("/ui/dashboard", ui.DashboardHandler, http.MethodGet)
	register("/ui/plans", ui.PlansHandler, http.MethodGet)
	register("/ui/contexts", ui.ContextsHandler, http.MethodGet)
	register("/ui/models", ui.ModelsHandler, http.MethodGet)
	register("/ui/mcp", ui.MCPHandler, http.MethodGet)
	register("/ui/settings", ui.SettingsHandler, http.MethodGet)

	// It's often good practice to have a root redirect for /ui or /ui/
	register("/ui/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/dashboard", http.StatusFound)
	}), http.MethodGet)
	register("/ui", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/dashboard", http.StatusFound)
	}), http.MethodGet)
}
