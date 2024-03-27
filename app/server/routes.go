package main

import (
	"fmt"
	"net/http"
	"os"
	"plandex-server/handlers"

	"github.com/gorilla/mux"
)

func routes() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "OK")
	})

	r.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		// get version from version.txt
		bytes, err := os.ReadFile("version.txt")

		if err != nil {
			http.Error(w, "Error getting version", http.StatusInternalServerError)
			return
		}

		fmt.Fprint(w, string(bytes))
	})

	r.HandleFunc("/accounts/start_trial", handlers.StartTrialHandler).Methods("POST")
	r.HandleFunc("/accounts/email_verifications", handlers.CreateEmailVerificationHandler).Methods("POST")
	r.HandleFunc("/accounts/sign_in", handlers.SignInHandler).Methods("POST")
	r.HandleFunc("/accounts/sign_out", handlers.SignOutHandler).Methods("POST")
	r.HandleFunc("/accounts", handlers.CreateAccountHandler).Methods("POST")
	r.HandleFunc("/accounts/convert_trial", handlers.ConvertTrialHandler).Methods("POST")

	r.HandleFunc("/orgs/session", handlers.GetOrgSessionHandler).Methods("GET")
	r.HandleFunc("/orgs", handlers.ListOrgsHandler).Methods("GET")
	r.HandleFunc("/orgs", handlers.CreateOrgHandler).Methods("POST")

	r.HandleFunc("/users", handlers.ListUsersHandler).Methods("GET")
	r.HandleFunc("/orgs/users/{userId}", handlers.DeleteOrgUserHandler).Methods("DELETE")
	r.HandleFunc("/orgs/roles", handlers.ListOrgRolesHandler).Methods("GET")

	r.HandleFunc("/invites", handlers.InviteUserHandler).Methods("POST")
	r.HandleFunc("/invites/pending", handlers.ListPendingInvitesHandler).Methods("GET")
	r.HandleFunc("/invites/accepted", handlers.ListAcceptedInvitesHandler).Methods("GET")
	r.HandleFunc("/invites/all", handlers.ListAllInvitesHandler).Methods("GET")
	r.HandleFunc("/invites/{inviteId}", handlers.DeleteInviteHandler).Methods("DELETE")

	r.HandleFunc("/projects", handlers.CreateProjectHandler).Methods("POST")
	r.HandleFunc("/projects", handlers.ListProjectsHandler).Methods("GET")
	r.HandleFunc("/projects/{projectId}/set_plan", handlers.ProjectSetPlanHandler).Methods("PUT")
	r.HandleFunc("/projects/{projectId}/rename", handlers.RenameProjectHandler).Methods("PUT")

	r.HandleFunc("/projects/{projectId}/plans/current_branches", handlers.GetCurrentBranchByPlanIdHandler).Methods("POST")

	r.HandleFunc("/plans", handlers.ListPlansHandler).Methods("GET")
	r.HandleFunc("/plans/archive", handlers.ListArchivedPlansHandler).Methods("GET")
	r.HandleFunc("/plans/ps", handlers.ListPlansRunningHandler).Methods("GET")

	r.HandleFunc("/projects/{projectId}/plans", handlers.CreatePlanHandler).Methods("POST")

	r.HandleFunc("/projects/{projectId}/plans", handlers.CreatePlanHandler).Methods("DELETE")

	r.HandleFunc("/plans/{planId}", handlers.GetPlanHandler).Methods("GET")
	r.HandleFunc("/plans/{planId}", handlers.DeletePlanHandler).Methods("DELETE")

	r.HandleFunc("/plans/{planId}/{branch}/tell", handlers.TellPlanHandler).Methods("POST")

	r.HandleFunc("/plans/{planId}/{branch}/respond_missing_file", handlers.RespondMissingFileHandler).Methods("POST")

	r.HandleFunc("/plans/{planId}/{branch}/build", handlers.BuildPlanHandler).Methods("PATCH")
	r.HandleFunc("/plans/{planId}/{branch}/connect", handlers.ConnectPlanHandler).Methods("PATCH")
	r.HandleFunc("/plans/{planId}/{branch}/stop", handlers.StopPlanHandler).Methods("DELETE")

	r.HandleFunc("/plans/{planId}/{branch}/current_plan", handlers.CurrentPlanHandler).Methods("GET")
	r.HandleFunc("/plans/{planId}/{branch}/apply", handlers.ApplyPlanHandler).Methods("PATCH")
	r.HandleFunc("/plans/{planId}/{branch}/archive", handlers.ArchivePlanHandler).Methods("PATCH")
	r.HandleFunc("/plans/{planId}/{branch}/reject_all", handlers.RejectAllChangesHandler).Methods("PATCH")
	r.HandleFunc("/plans/{planId}/{branch}/reject_file", handlers.RejectFileHandler).Methods("PATCH")

	r.HandleFunc("/plans/{planId}/{branch}/context", handlers.ListContextHandler).Methods("GET")
	r.HandleFunc("/plans/{planId}/{branch}/context", handlers.LoadContextHandler).Methods("POST")
	r.HandleFunc("/plans/{planId}/{branch}/context", handlers.UpdateContextHandler).Methods("PUT")
	r.HandleFunc("/plans/{planId}/{branch}/context", handlers.DeleteContextHandler).Methods("DELETE")

	r.HandleFunc("/plans/{planId}/{branch}/convo", handlers.ListConvoHandler).Methods("GET")
	r.HandleFunc("/plans/{planId}/{branch}/rewind", handlers.RewindPlanHandler).Methods("PATCH")
	r.HandleFunc("/plans/{planId}/{branch}/logs", handlers.ListLogsHandler).Methods("GET")

	r.HandleFunc("/plans/{planId}/branches", handlers.ListBranchesHandler).Methods("GET")
	r.HandleFunc("/plans/{planId}/branches/{branch}", handlers.DeleteBranchHandler).Methods("DELETE")
	r.HandleFunc("/plans/{planId}/{branch}/branches", handlers.CreateBranchHandler).Methods("POST")

	r.HandleFunc("/plans/{planId}/{branch}/settings", handlers.GetSettingsHandler).Methods("GET")
	r.HandleFunc("/plans/{planId}/{branch}/settings", handlers.UpdateSettingsHandler).Methods("PUT")

	return r

}
