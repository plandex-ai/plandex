package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"plandex-server/db"
	"strings"

	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
)

func ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for ListUsersHandler")
	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	if auth.User.IsTrial {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeTrialActionNotAllowed,
			Status: http.StatusForbidden,
			Msg:    "Free trial user can't list users",
		})
		return
	}

	users, err := db.ListUsers(auth.OrgId)
	if err != nil {
		log.Println("Error listing users: ", err)
		http.Error(w, "Error listing users: "+err.Error(), http.StatusInternalServerError)
	}

	bytes, err := json.Marshal(users)

	if err != nil {
		log.Println("Error marshalling users: ", err)
		http.Error(w, "Error marshalling users: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully processed request for ListUsersHandler")

	w.Write(bytes)
}

func DeleteOrgUserHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for DeleteOrgUserHandler")
	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	if auth.User.IsTrial {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeTrialActionNotAllowed,
			Status: http.StatusForbidden,
			Msg:    "Free trial user can't delete users",
		})
		return
	}

	vars := mux.Vars(r)
	userId := vars["userId"]

	user, err := db.GetUser(userId)

	if err != nil {
		log.Printf("Error getting user: %v\n", err)
		http.Error(w, "Error getting user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// ensure current user can invite target user
	if !auth.HasPermission(strings.Join([]string{"remove_user", user.OrgRoleId}, "|")) {
		log.Printf("User does not have permission to invite user with role: %v\n", user.OrgRoleId)
		http.Error(w, "User does not have permission to invite user with role: "+user.OrgRoleId, http.StatusForbidden)
		return
	}

	// verify user is org member
	isMember, err := db.ValidateOrgMembership(userId, auth.OrgId)

	if err != nil {
		log.Printf("Error validating org membership: %v\n", err)
		http.Error(w, "Error validating org membership: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !isMember {
		log.Printf("User %s is not a member of org %s\n", userId, auth.OrgId)
		http.Error(w, "User "+userId+" is not a member of org "+auth.OrgId, http.StatusForbidden)
		return
	}

	// start a transaction
	tx, err := db.Conn.Begin()
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

	err = db.DeleteOrgUser(userId, auth.OrgId, tx)

	if err != nil {
		log.Println("Error deleting org user: ", err)
		http.Error(w, "Error deleting org user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	invite, err := db.GetInviteForOrgUser(auth.OrgId, userId)

	if err != nil {
		log.Println("Error getting invite for org user: ", err)
		http.Error(w, "Error getting invite for org user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if invite != nil {
		err = db.DeleteInvite(invite.Id, tx)

		if err != nil {
			log.Println("Error deleting invite: ", err)
			http.Error(w, "Error deleting invite: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	log.Println("Successfully processed request for DeleteOrgUserHandler")
}
