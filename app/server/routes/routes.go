package routes

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"plandex-server/handlers"
	"plandex-server/hooks"

	"github.com/gorilla/mux"
)

func AddHealthRoutes(r *mux.Router) {
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, apiErr := hooks.ExecHook(hooks.HealthCheck, hooks.HookParams{})
		if apiErr != nil {
			log.Printf("Error in health check hook: %v\n", apiErr)
			http.Error(w, apiErr.Msg, apiErr.Status)
			return
		}
		fmt.Fprint(w, "OK")
	})

	r.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		// Log the host
		host := r.Host
		log.Printf("Host header: %s", host)

		execPath, err := os.Executable()
		if err != nil {
			log.Fatal("Error getting current directory: ", err)
		}
		currentDir := filepath.Dir(execPath)

		// get version from version.txt
		bytes, err := os.ReadFile(filepath.Join(currentDir, "..", "version.txt"))

		if err != nil {
			http.Error(w, "Error getting version", http.StatusInternalServerError)
			return
		}

		fmt.Fprint(w, string(bytes))
	})
}

func AddApiRoutes(r *mux.Router) {
	addApiRoutes(r, "")
}

func AddApiRoutesWithPrefix(r *mux.Router, prefix string) {
	addApiRoutes(r, prefix)
}

func AddProxyableApiRoutes(r *mux.Router) {
	addProxyableApiRoutes(r, "")
}

func AddProxyableApiRoutesWithPrefix(r *mux.Router, prefix string) {
	addProxyableApiRoutes(r, prefix)
}

func addApiRoutes(r *mux.Router, prefix string) {
	r.HandleFunc(prefix+"/accounts/email_verifications", handlers.CreateEmailVerificationHandler).Methods("POST")
	r.HandleFunc(prefix+"/accounts/email_verifications/check_pin", handlers.CheckEmailPinHandler).Methods("POST")
	r.HandleFunc(prefix+"/accounts/sign_in_codes", handlers.CreateSignInCodeHandler).Methods("POST")
	r.HandleFunc(prefix+"/accounts/sign_in", handlers.SignInHandler).Methods("POST")
	r.HandleFunc(prefix+"/accounts/sign_out", handlers.SignOutHandler).Methods("POST")
	r.HandleFunc(prefix+"/accounts", handlers.CreateAccountHandler).Methods("POST")

	r.HandleFunc(prefix+"/orgs/session", handlers.GetOrgSessionHandler).Methods("GET")
	r.HandleFunc(prefix+"/orgs", handlers.ListOrgsHandler).Methods("GET")
	r.HandleFunc(prefix+"/orgs", handlers.CreateOrgHandler).Methods("POST")

	r.HandleFunc(prefix+"/users", handlers.ListUsersHandler).Methods("GET")
	r.HandleFunc(prefix+"/orgs/users/{userId}", handlers.DeleteOrgUserHandler).Methods("DELETE")
	r.HandleFunc(prefix+"/orgs/roles", handlers.ListOrgRolesHandler).Methods("GET")

	r.HandleFunc(prefix+"/invites", handlers.InviteUserHandler).Methods("POST")
	r.HandleFunc(prefix+"/invites/pending", handlers.ListPendingInvitesHandler).Methods("GET")
	r.HandleFunc(prefix+"/invites/accepted", handlers.ListAcceptedInvitesHandler).Methods("GET")
	r.HandleFunc(prefix+"/invites/all", handlers.ListAllInvitesHandler).Methods("GET")
	r.HandleFunc(prefix+"/invites/{inviteId}", handlers.DeleteInviteHandler).Methods("DELETE")

	r.HandleFunc(prefix+"/projects", handlers.CreateProjectHandler).Methods("POST")
	r.HandleFunc(prefix+"/projects", handlers.ListProjectsHandler).Methods("GET")
	r.HandleFunc(prefix+"/projects/{projectId}/set_plan", handlers.ProjectSetPlanHandler).Methods("PUT")
	r.HandleFunc(prefix+"/projects/{projectId}/rename", handlers.RenameProjectHandler).Methods("PUT")

	r.HandleFunc(prefix+"/projects/{projectId}/plans/current_branches", handlers.GetCurrentBranchByPlanIdHandler).Methods("POST")

	r.HandleFunc(prefix+"/plans", handlers.ListPlansHandler).Methods("GET")
	r.HandleFunc(prefix+"/plans/archive", handlers.ListArchivedPlansHandler).Methods("GET")
	r.HandleFunc(prefix+"/plans/ps", handlers.ListPlansRunningHandler).Methods("GET")

	r.HandleFunc(prefix+"/projects/{projectId}/plans", handlers.CreatePlanHandler).Methods("POST")

	r.HandleFunc(prefix+"/projects/{projectId}/plans", handlers.CreatePlanHandler).Methods("DELETE")

	r.HandleFunc(prefix+"/plans/{planId}", handlers.GetPlanHandler).Methods("GET")
	r.HandleFunc(prefix+"/plans/{planId}", handlers.DeletePlanHandler).Methods("DELETE")

	r.HandleFunc(prefix+"/plans/{planId}/{branch}/current_plan", handlers.CurrentPlanHandler).Methods("GET")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/apply", handlers.ApplyPlanHandler).Methods("PATCH")
	r.HandleFunc(prefix+"/plans/{planId}/archive", handlers.ArchivePlanHandler).Methods("PATCH")
	r.HandleFunc(prefix+"/plans/{planId}/unarchive", handlers.UnarchivePlanHandler).Methods("PATCH")

	r.HandleFunc(prefix+"/plans/{planId}/rename", handlers.RenamePlanHandler).Methods("PATCH")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/reject_all", handlers.RejectAllChangesHandler).Methods("PATCH")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/reject_file", handlers.RejectFileHandler).Methods("PATCH")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/reject_files", handlers.RejectFilesHandler).Methods("PATCH")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/diffs", handlers.GetPlanDiffsHandler).Methods("GET")

	r.HandleFunc(prefix+"/plans/{planId}/{branch}/context", handlers.ListContextHandler).Methods("GET")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/context", handlers.LoadContextHandler).Methods("POST")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/context/{contextId}/body", handlers.GetContextBodyHandler).Methods("GET")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/context", handlers.UpdateContextHandler).Methods("PUT")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/context", handlers.DeleteContextHandler).Methods("DELETE")

	r.HandleFunc(prefix+"/plans/{planId}/{branch}/convo", handlers.ListConvoHandler).Methods("GET")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/rewind", handlers.RewindPlanHandler).Methods("PATCH")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/logs", handlers.ListLogsHandler).Methods("GET")

	r.HandleFunc(prefix+"/plans/{planId}/branches", handlers.ListBranchesHandler).Methods("GET")
	r.HandleFunc(prefix+"/plans/{planId}/branches/{branch}", handlers.DeleteBranchHandler).Methods("DELETE")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/branches", handlers.CreateBranchHandler).Methods("POST")

	r.HandleFunc(prefix+"/plans/{planId}/{branch}/settings", handlers.GetSettingsHandler).Methods("GET")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/settings", handlers.UpdateSettingsHandler).Methods("PUT")

	r.HandleFunc(prefix+"/plans/{planId}/{branch}/status", handlers.GetPlanStatusHandler).Methods("GET")

	r.HandleFunc(prefix+"/plans/{planId}/{branch}/tell", handlers.TellPlanHandler).Methods("POST")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/build", handlers.BuildPlanHandler).Methods("PATCH")

	r.HandleFunc(prefix+"/custom_models", handlers.ListCustomModelsHandler).Methods("GET")
	r.HandleFunc(prefix+"/custom_models", handlers.CreateCustomModelHandler).Methods("POST")
	r.HandleFunc(prefix+"/custom_models/{modelId}", handlers.DeleteAvailableModelHandler).Methods("DELETE")

	r.HandleFunc(prefix+"/model_sets", handlers.ListModelPacksHandler).Methods("GET")
	r.HandleFunc(prefix+"/model_sets", handlers.CreateModelPackHandler).Methods("POST")
	r.HandleFunc(prefix+"/model_sets/{setId}", handlers.DeleteModelPackHandler).Methods("DELETE")

	r.HandleFunc(prefix+"/default_settings", handlers.GetDefaultSettingsHandler).Methods("GET")
	r.HandleFunc(prefix+"/default_settings", handlers.UpdateDefaultSettingsHandler).Methods("PUT")

	r.HandleFunc(prefix+"/file_map", handlers.GetFileMapHandler).Methods("POST")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/load_cached_file_map", handlers.LoadCachedFileMapHandler).Methods("POST")

	r.HandleFunc(prefix+"/plans/{planId}/config", handlers.GetPlanConfigHandler).Methods("GET")
	r.HandleFunc(prefix+"/plans/{planId}/config", handlers.UpdatePlanConfigHandler).Methods("PUT")

	r.HandleFunc(prefix+"/default_plan_config", handlers.GetDefaultPlanConfigHandler).Methods("GET")
	r.HandleFunc(prefix+"/default_plan_config", handlers.UpdateDefaultPlanConfigHandler).Methods("PUT")
}

func addProxyableApiRoutes(r *mux.Router, prefix string) {
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/connect", handlers.ConnectPlanHandler).Methods("PATCH")
	r.HandleFunc(prefix+"/plans/{planId}/{branch}/stop", handlers.StopPlanHandler).Methods("DELETE")

	r.HandleFunc(prefix+"/plans/{planId}/{branch}/respond_missing_file", handlers.RespondMissingFileHandler).Methods("POST")

	r.HandleFunc(prefix+"/plans/{planId}/{branch}/auto_load_context", handlers.AutoLoadContextHandler).Methods("POST")
}
