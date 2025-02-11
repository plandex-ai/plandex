package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"plandex-server/db"
	"plandex-server/email"
	"strings"

	shared "plandex-shared"

	"github.com/gorilla/mux"
)

func InviteUserHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for InviteUserHandler")

	if os.Getenv("GOENV") == "development" && os.Getenv("LOCAL_MODE") == "1" {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusForbidden,
			Msg:    "Local mode is not supported for invites",
		})
		return
	}

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	org, err := db.GetOrg(auth.OrgId)

	if err != nil {
		log.Printf("Error getting org: %v\n", err)
		http.Error(w, "Error getting org: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if org.IsTrial {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeTrialActionNotAllowed,
			Status: http.StatusForbidden,
			Msg:    "Trial user can't invite other users",
		})

		return
	}

	currentUserId := auth.User.Id

	var req shared.InviteRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Error unmarshalling request: %v\n", err)
		http.Error(w, "Error unmarshalling request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	req.Email = strings.ToLower(req.Email)

	// ensure current user can invite target user
	permission := shared.Permission(strings.Join([]string{string(shared.PermissionInviteUser), req.OrgRoleId}, "|"))

	if !auth.HasPermission(permission) {
		log.Printf("User does not have permission to invite user with role: %v\n", req.OrgRoleId)
		http.Error(w, "User does not have permission to invite user with role: "+req.OrgRoleId, http.StatusForbidden)
		return
	}

	// ensure user doesn't already have access to org via domain
	split := strings.Split(req.Email, "@")
	if len(split) != 2 {
		log.Printf("Invalid email: %v\n", req.Email)
		http.Error(w, "Invalid email: "+req.Email, http.StatusBadRequest)
		return
	}
	domain := &split[1]

	if org.AutoAddDomainUsers && org.Domain == domain {
		log.Printf("User already has access to org via domain: %v\n", domain)
		http.Error(w, "User already has access to org via domain: "+*domain, http.StatusBadRequest)
	}

	// ensure user with this email isn't already in the org
	user, err := db.GetUserByEmail(req.Email)

	if err != nil {
		log.Printf("Error getting user: %v\n", err)
		http.Error(w, "Error getting user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if user != nil {
		isMember, err := db.ValidateOrgMembership(user.Id, auth.OrgId)

		if err != nil {
			log.Printf("Error validating org membership: %v\n", err)
			http.Error(w, "Error validating org membership: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if isMember {
			log.Println("User is already a member of org")
			http.Error(w, "User is already a member of org", http.StatusBadRequest)
			return
		}
	}

	// ensure invite isn't already active
	invite, err := db.GetActiveInviteByEmail(auth.OrgId, req.Email)

	if err != nil {
		log.Printf("Error getting invite: %v\n", err)
		http.Error(w, "Error getting invite: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if invite != nil {
		log.Println("Invite already exists")
		http.Error(w, "Invite already exists", http.StatusBadRequest)
		return
	}

	// start a transaction
	tx, err := db.Conn.Beginx()
	if err != nil {
		log.Printf("Error starting transaction: %v\n", err)
		http.Error(w, "Error starting transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Ensure that rollback is attempted in case of failure
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("transaction rollback error: %v\n", rbErr)
			} else {
				log.Println("transaction rolled back")
			}
		}
	}()

	err = db.CreateInvite(&db.Invite{
		OrgId:     auth.OrgId,
		OrgRoleId: req.OrgRoleId,
		Email:     req.Email,
		Name:      req.Name,
		InviterId: currentUserId,
	}, tx)

	if err != nil {
		log.Printf("Error creating invite: %v\n", err)
		http.Error(w, "Error creating invite: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = email.SendInviteEmail(req.Email, req.Name, auth.User.Name, org.Name)

	if err != nil {
		log.Printf("Error sending invite email: %v\n", err)
		http.Error(w, "Error sending invite email: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// commit transaction
	err = tx.Commit()
	if err != nil {
		log.Printf("Error committing transaction: %v\n", err)
		http.Error(w, "Error committing transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully created invite")
}

func ListPendingInvitesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for ListInvitesHandler")

	if os.Getenv("GOENV") == "development" && os.Getenv("LOCAL_MODE") == "1" {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusForbidden,
			Msg:    "Local mode is not supported for invites",
		})
		return
	}

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	org, err := db.GetOrg(auth.OrgId)
	if err != nil {
		log.Printf("Error getting org: %v\n", err)
		http.Error(w, "Error getting org: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if org.IsTrial {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeTrialActionNotAllowed,
			Status: http.StatusForbidden,
			Msg:    "Trial user can't list invites",
		})
		return
	}

	invites, err := db.ListPendingInvites(auth.OrgId)

	if err != nil {
		log.Printf("Error listing invites: %v\n", err)
		http.Error(w, "Error listing invites: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiInvites []*shared.Invite
	for _, invite := range invites {
		apiInvites = append(apiInvites, invite.ToApi())
	}

	bytes, err := json.Marshal(apiInvites)

	if err != nil {
		log.Printf("Error marshalling invites: %v\n", err)
		http.Error(w, "Error marshalling invites: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(bytes)
	log.Println("Successfully processed request for ListPendingInvitesHandler")
}

func ListAcceptedInvitesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for ListAcceptedInvitesHandler")

	if os.Getenv("GOENV") == "development" && os.Getenv("LOCAL_MODE") == "1" {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusForbidden,
			Msg:    "Local mode is not supported for invites",
		})
		return
	}

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	org, err := db.GetOrg(auth.OrgId)
	if err != nil {
		log.Printf("Error getting org: %v\n", err)
		http.Error(w, "Error getting org: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if org.IsTrial {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeTrialActionNotAllowed,
			Status: http.StatusForbidden,
			Msg:    "Trial user can't list invites",
		})
		return
	}

	invites, err := db.ListAcceptedInvites(auth.OrgId)

	if err != nil {
		log.Printf("Error listing invites: %v\n", err)
		http.Error(w, "Error listing invites: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiInvites []*shared.Invite
	for _, invite := range invites {
		apiInvites = append(apiInvites, invite.ToApi())
	}

	bytes, err := json.Marshal(apiInvites)

	if err != nil {
		log.Printf("Error marshalling invites: %v\n", err)
		http.Error(w, "Error marshalling invites: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(bytes)
	log.Println("Successfully processed request for ListAcceptedInvitesHandler")
}

func ListAllInvitesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for ListAllInvitesHandler")

	if os.Getenv("GOENV") == "development" && os.Getenv("LOCAL_MODE") == "1" {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusForbidden,
			Msg:    "Local mode is not supported for invites",
		})
		return
	}

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	org, err := db.GetOrg(auth.OrgId)
	if err != nil {
		log.Printf("Error getting org: %v\n", err)
		http.Error(w, "Error getting org: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if org.IsTrial {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeTrialActionNotAllowed,
			Status: http.StatusForbidden,
			Msg:    "Trial user can't list invites",
		})
		return
	}

	invites, err := db.ListAllInvites(auth.OrgId)

	if err != nil {
		log.Printf("Error listing invites: %v\n", err)
		http.Error(w, "Error listing invites: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiInvites []*shared.Invite
	for _, invite := range invites {
		apiInvites = append(apiInvites, invite.ToApi())
	}

	bytes, err := json.Marshal(apiInvites)

	if err != nil {
		log.Printf("Error marshalling invites: %v\n", err)
		http.Error(w, "Error marshalling invites: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(bytes)
	log.Println("Successfully processed request for ListAllInvitesHandler")
}

func DeleteInviteHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for DeleteInviteHandler")

	if os.Getenv("GOENV") == "development" && os.Getenv("LOCAL_MODE") == "1" {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusForbidden,
			Msg:    "Local mode is not supported for invites",
		})
		return
	}

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	org, err := db.GetOrg(auth.OrgId)
	if err != nil {
		log.Printf("Error getting org: %v\n", err)
		http.Error(w, "Error getting org: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if org.IsTrial {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeTrialActionNotAllowed,
			Status: http.StatusForbidden,
			Msg:    "Trial user can't delete invites",
		})
		return
	}

	vars := mux.Vars(r)
	inviteId := vars["inviteId"]

	invite, err := db.GetInvite(inviteId)

	if err != nil {
		log.Printf("Error getting invite: %v\n", err)
		http.Error(w, "Error getting invite: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if invite == nil || invite.OrgId != auth.OrgId {
		log.Printf("Invite not found: %v\n", inviteId)
		http.Error(w, "Invite not found: "+inviteId, http.StatusNotFound)
		return
	}

	// ensure current user can remove target invite
	removePermission := shared.Permission(strings.Join([]string{string(shared.PermissionRemoveUser), invite.OrgRoleId}, "|"))

	invitePermission := shared.Permission(strings.Join([]string{string(shared.PermissionInviteUser), invite.OrgRoleId}, "|"))

	if !(auth.HasPermission(removePermission) ||
		(auth.User.Id == invite.InviterId && auth.HasPermission(invitePermission))) {
		log.Printf("User does not have permission to remove invite with role: %v\n", invite.OrgRoleId)
		http.Error(w, "User does not have permission to remove invite with role: "+invite.OrgRoleId, http.StatusForbidden)
		return
	}

	err = db.DeleteInvite(inviteId, nil)

	if err != nil {
		log.Printf("Error deleting invite: %v\n", err)
		http.Error(w, "Error deleting invite: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully deleted invite")
}
