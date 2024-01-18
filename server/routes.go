package main

import (
	"fmt"
	"net/http"
	"plandex-server/handlers"

	"github.com/gorilla/mux"
)

func InitRoutes() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
	})

	r.HandleFunc("/accounts/start_trial", handlers.StartTrialHandler).Methods("POST")
	r.HandleFunc("/accounts/email_verifications", handlers.CreateEmailVerificationHandler).Methods("POST")
	r.HandleFunc("/accounts/sign_in", handlers.SignInHandler).Methods("POST")
	r.HandleFunc("/accounts/sign_out", handlers.SignOutHandler).Methods("POST")
	r.HandleFunc("/accounts", handlers.CreateAccountHandler).Methods("POST")
	r.HandleFunc("/accounts/convert_trial", handlers.ConvertTrialHandler).Methods("POST")

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
	r.HandleFunc("/projects/{projectId}/plans", handlers.ListPlansHandler).Methods("GET")
	r.HandleFunc("/projects/{projectId}/plans/archive", handlers.ListArchivedPlansHandler).Methods("GET")
	r.HandleFunc("/projects/{projectId}/plans/ps", handlers.ListPlansRunningHandler).Methods("GET")

	r.HandleFunc("/projects/{projectId}/plans", handlers.CreatePlanHandler).Methods("POST")
	r.HandleFunc("/projects/{projectId}/plans", handlers.CreatePlanHandler).Methods("DELETE")
	r.HandleFunc("/plans/{planId}", handlers.GetPlanHandler).Methods("GET")
	r.HandleFunc("/plans/{planId}", handlers.DeletePlanHandler).Methods("DELETE")

	r.HandleFunc("/plans/{planId}/tell", handlers.TellPlanHandler).Methods("POST")
	r.HandleFunc("/plans/{planId}/build", handlers.BuildPlanHandler).Methods("PATCH")
	r.HandleFunc("/plans/{planId}/connect", handlers.ConnectPlanHandler).Methods("PATCH")
	r.HandleFunc("/plans/{planId}/stop", handlers.StopPlanHandler).Methods("DELETE")

	r.HandleFunc("/plans/{planId}/current_plan", handlers.CurrentPlanHandler).Methods("GET")
	r.HandleFunc("/plans/{planId}/apply", handlers.ApplyPlanHandler).Methods("PATCH")
	r.HandleFunc("/plans/{planId}/archive", handlers.ArchivePlanHandler).Methods("PATCH")
	r.HandleFunc("/plans/{planId}/reject_all", handlers.RejectAllChangesHandler).Methods("PATCH")
	r.HandleFunc("/plans/{planId}/results/{resultId}/reject", handlers.RejectResultHandler).Methods("PATCH")
	r.HandleFunc("/plans/{planId}/results/{resultId}/replacements/{replacementId}/reject", handlers.RejectReplacementHandler).Methods("PATCH")

	r.HandleFunc("/plans/{planId}/context", handlers.ListContextHandler).Methods("GET")
	r.HandleFunc("/plans/{planId}/context", handlers.LoadContextHandler).Methods("POST")
	r.HandleFunc("/plans/{planId}/context", handlers.UpdateContextHandler).Methods("PUT")
	r.HandleFunc("/plans/{planId}/context", handlers.DeleteContextHandler).Methods("DELETE")

	r.HandleFunc("/plans/{planId}/convo", handlers.ListConvoHandler).Methods("GET")
	r.HandleFunc("/plans/{planId}/rewind", handlers.RewindPlanHandler).Methods("PATCH")
	r.HandleFunc("/plans/{planId}/logs", handlers.ListLogsHandler).Methods("GET")

	return r
}
