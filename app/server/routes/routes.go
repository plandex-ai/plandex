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

type PlandexHandler func(w http.ResponseWriter, r *http.Request)
type HandlePlandex func(router *mux.Router, path string, isStreaming bool, handler PlandexHandler) *mux.Route

var HandlePlandexFn HandlePlandex

func RegisterHandlePlandex(fn HandlePlandex) {
	HandlePlandexFn = fn
}

func EnsureHandlePlandex() {
	if HandlePlandexFn == nil {
		panic("handlePlandexFn is not set")
	}
}

func AddHealthRoutes(r *mux.Router) {
	EnsureHandlePlandex()

	HandlePlandexFn(r, "/health", false, func(w http.ResponseWriter, r *http.Request) {
		_, apiErr := hooks.ExecHook(hooks.HealthCheck, hooks.HookParams{})
		if apiErr != nil {
			log.Printf("Error in health check hook: %v\n", apiErr)
			http.Error(w, apiErr.Msg, apiErr.Status)
			return
		}
		fmt.Fprint(w, "OK")
	})

	HandlePlandexFn(r, "/version", false, func(w http.ResponseWriter, r *http.Request) {
		// Log the host
		host := r.Host
		log.Printf("Host header: %s", host)

		execPath, err := os.Executable()
		if err != nil {
			log.Fatal("Error getting current directory: ", err)
		}
		currentDir := filepath.Dir(execPath)

		// get version from version.txt
		var path string
		if os.Getenv("IS_CLOUD") != "" {
			path = filepath.Join(currentDir, "..", "version.txt")
		} else {
			path = filepath.Join(currentDir, "version.txt")
		}

		bytes, err := os.ReadFile(path)

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
	EnsureHandlePlandex()

	HandlePlandexFn(r, prefix+"/accounts/email_verifications", false, handlers.CreateEmailVerificationHandler).Methods("POST")
	HandlePlandexFn(r, prefix+"/accounts/email_verifications/check_pin", false, handlers.CheckEmailPinHandler).Methods("POST")
	HandlePlandexFn(r, prefix+"/accounts/sign_in_codes", false, handlers.CreateSignInCodeHandler).Methods("POST")
	HandlePlandexFn(r, prefix+"/accounts/sign_in", false, handlers.SignInHandler).Methods("POST")
	HandlePlandexFn(r, prefix+"/accounts/sign_out", false, handlers.SignOutHandler).Methods("POST")
	HandlePlandexFn(r, prefix+"/accounts", false, handlers.CreateAccountHandler).Methods("POST")

	HandlePlandexFn(r, prefix+"/orgs/session", false, handlers.GetOrgSessionHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/orgs", false, handlers.ListOrgsHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/orgs", false, handlers.CreateOrgHandler).Methods("POST")

	HandlePlandexFn(r, prefix+"/users", false, handlers.ListUsersHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/orgs/users/{userId}", false, handlers.DeleteOrgUserHandler).Methods("DELETE")
	HandlePlandexFn(r, prefix+"/orgs/roles", false, handlers.ListOrgRolesHandler).Methods("GET")

	HandlePlandexFn(r, prefix+"/invites", false, handlers.InviteUserHandler).Methods("POST")
	HandlePlandexFn(r, prefix+"/invites/pending", false, handlers.ListPendingInvitesHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/invites/accepted", false, handlers.ListAcceptedInvitesHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/invites/all", false, handlers.ListAllInvitesHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/invites/{inviteId}", false, handlers.DeleteInviteHandler).Methods("DELETE")

	HandlePlandexFn(r, prefix+"/projects", false, handlers.CreateProjectHandler).Methods("POST")
	HandlePlandexFn(r, prefix+"/projects", false, handlers.ListProjectsHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/projects/{projectId}/set_plan", false, handlers.ProjectSetPlanHandler).Methods("PUT")
	HandlePlandexFn(r, prefix+"/projects/{projectId}/rename", false, handlers.RenameProjectHandler).Methods("PUT")

	HandlePlandexFn(r, prefix+"/projects/{projectId}/plans/current_branches", false, handlers.GetCurrentBranchByPlanIdHandler).Methods("POST")

	HandlePlandexFn(r, prefix+"/plans", false, handlers.ListPlansHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/plans/archive", false, handlers.ListArchivedPlansHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/plans/ps", false, handlers.ListPlansRunningHandler).Methods("GET")

	HandlePlandexFn(r, prefix+"/projects/{projectId}/plans", false, handlers.CreatePlanHandler).Methods("POST")

	HandlePlandexFn(r, prefix+"/projects/{projectId}/plans", false, handlers.CreatePlanHandler).Methods("DELETE")

	HandlePlandexFn(r, prefix+"/plans/{planId}", false, handlers.GetPlanHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/plans/{planId}", false, handlers.DeletePlanHandler).Methods("DELETE")

	HandlePlandexFn(r, prefix+"/plans/{planId}/current_plan/{sha}", false, handlers.CurrentPlanHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/current_plan", false, handlers.CurrentPlanHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/apply", false, handlers.ApplyPlanHandler).Methods("PATCH")
	HandlePlandexFn(r, prefix+"/plans/{planId}/archive", false, handlers.ArchivePlanHandler).Methods("PATCH")
	HandlePlandexFn(r, prefix+"/plans/{planId}/unarchive", false, handlers.UnarchivePlanHandler).Methods("PATCH")

	HandlePlandexFn(r, prefix+"/plans/{planId}/rename", false, handlers.RenamePlanHandler).Methods("PATCH")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/reject_all", false, handlers.RejectAllChangesHandler).Methods("PATCH")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/reject_file", false, handlers.RejectFileHandler).Methods("PATCH")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/reject_files", false, handlers.RejectFilesHandler).Methods("PATCH")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/diffs", false, handlers.GetPlanDiffsHandler).Methods("GET")

	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/context", false, handlers.ListContextHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/context", false, handlers.LoadContextHandler).Methods("POST")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/context/{contextId}/body", false, handlers.GetContextBodyHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/context", false, handlers.UpdateContextHandler).Methods("PUT")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/context", false, handlers.DeleteContextHandler).Methods("DELETE")

	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/convo", false, handlers.ListConvoHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/rewind", false, handlers.RewindPlanHandler).Methods("PATCH")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/logs", false, handlers.ListLogsHandler).Methods("GET")

	HandlePlandexFn(r, prefix+"/plans/{planId}/branches", false, handlers.ListBranchesHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/plans/{planId}/branches/{branch}", false, handlers.DeleteBranchHandler).Methods("DELETE")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/branches", false, handlers.CreateBranchHandler).Methods("POST")

	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/settings", false, handlers.GetSettingsHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/settings", false, handlers.UpdateSettingsHandler).Methods("PUT")

	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/status", false, handlers.GetPlanStatusHandler).Methods("GET")

	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/tell", true, handlers.TellPlanHandler).Methods("POST")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/build", true, handlers.BuildPlanHandler).Methods("PATCH")

	HandlePlandexFn(r, prefix+"/custom_models", false, handlers.ListCustomModelsHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/custom_models", false, handlers.CreateCustomModelHandler).Methods("POST")
	HandlePlandexFn(r, prefix+"/custom_models/{modelId}", false, handlers.DeleteAvailableModelHandler).Methods("DELETE")
	HandlePlandexFn(r, prefix+"/custom_models/{modelId}", false, handlers.UpdateCustomModelHandler).Methods("PUT")

	HandlePlandexFn(r, prefix+"/model_sets", false, handlers.ListModelPacksHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/model_sets", false, handlers.CreateModelPackHandler).Methods("POST")
	HandlePlandexFn(r, prefix+"/model_sets/{setId}", false, handlers.DeleteModelPackHandler).Methods("DELETE")
	HandlePlandexFn(r, prefix+"/model_sets/{setId}", false, handlers.UpdateModelPackHandler).Methods("PUT")
	HandlePlandexFn(r, prefix+"/default_settings", false, handlers.GetDefaultSettingsHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/default_settings", false, handlers.UpdateDefaultSettingsHandler).Methods("PUT")

	HandlePlandexFn(r, prefix+"/file_map", false, handlers.GetFileMapHandler).Methods("POST")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/load_cached_file_map", false, handlers.LoadCachedFileMapHandler).Methods("POST")

	HandlePlandexFn(r, prefix+"/plans/{planId}/config", false, handlers.GetPlanConfigHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/plans/{planId}/config", false, handlers.UpdatePlanConfigHandler).Methods("PUT")

	HandlePlandexFn(r, prefix+"/default_plan_config", false, handlers.GetDefaultPlanConfigHandler).Methods("GET")
	HandlePlandexFn(r, prefix+"/default_plan_config", false, handlers.UpdateDefaultPlanConfigHandler).Methods("PUT")
}

func addProxyableApiRoutes(r *mux.Router, prefix string) {
	EnsureHandlePlandex()

	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/connect", true, handlers.ConnectPlanHandler).Methods("PATCH")
	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/stop", false, handlers.StopPlanHandler).Methods("DELETE")

	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/respond_missing_file", false, handlers.RespondMissingFileHandler).Methods("POST")

	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/auto_load_context", false, handlers.AutoLoadContextHandler).Methods("POST")

	HandlePlandexFn(r, prefix+"/plans/{planId}/{branch}/build_status", false, handlers.GetBuildStatusHandler).Methods("GET")
}
