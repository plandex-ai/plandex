package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
)

func ListOrgsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListOrgsHandler")

	auth := authenticate(w, r, false)
	if auth == nil {
		return
	}

	orgs, err := db.GetAccessibleOrgsForUser(auth.User)

	if err != nil {
		log.Printf("Error listing orgs: %v\n", err)
		http.Error(w, "Error listing orgs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiOrgs []*shared.Org
	for _, org := range orgs {
		apiOrgs = append(apiOrgs, org.ToApi())
	}

	bytes, err := json.Marshal(apiOrgs)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully listed orgs")

	w.Write(bytes)
}

func CreateOrgHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreateOrgHandler")

	auth := authenticate(w, r, false)
	if auth == nil {
		return
	}

	if auth.User.IsTrial {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeTrialActionNotAllowed,
			Status: http.StatusForbidden,
			Msg:    "Anonymous trial user can't create org",
		})
		return
	}

	// read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var req shared.CreateOrgRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Printf("Error unmarshalling request: %v\n", err)
		http.Error(w, "Error unmarshalling request: "+err.Error(), http.StatusInternalServerError)
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

	var domain *string
	if req.AutoAddDomainUsers {
		if shared.IsEmailServiceDomain(auth.User.Domain) {
			log.Printf("Invalid domain: %v\n", auth.User.Domain)
			http.Error(w, "Invalid domain: "+auth.User.Domain, http.StatusBadRequest)
			return
		}

		domain = &auth.User.Domain
	}

	// create a new org
	org, err := db.CreateOrg(&req, auth.AuthToken.UserId, domain, tx)

	if err != nil {
		log.Printf("Error creating org: %v\n", err)
		http.Error(w, "Error creating org: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if org.AutoAddDomainUsers && org.Domain != nil {
		err = db.AddOrgDomainUsers(org.Id, *org.Domain, tx)

		if err != nil {
			log.Printf("Error adding org domain users: %v\n", err)
			http.Error(w, "Error adding org domain users: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	err = tx.Commit()

	if err != nil {
		log.Printf("Error committing transaction: %v\n", err)
		http.Error(w, "Error committing transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := shared.CreateOrgResponse{
		Id: org.Id,
	}

	bytes, err := json.Marshal(resp)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully created org")

	w.Write(bytes)
}

func GetOrgSessionHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for GetOrgSessionHandler")

	auth := authenticate(w, r, true)

	if auth == nil {
		return
	}

	log.Println("Successfully got org session")
}

func ListOrgRolesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListOrgRolesHandler")

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	if auth.User.IsTrial {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeTrialActionNotAllowed,
			Status: http.StatusForbidden,
			Msg:    "Anonymous trial user can't list org roles",
		})
		return
	}

	if !auth.HasPermission(types.PermissionListOrgRoles) {
		log.Println("User cannot list org roles")
		http.Error(w, "User cannot list org roles", http.StatusForbidden)
		return
	}

	roles, err := db.ListOrgRoles(auth.OrgId)

	if err != nil {
		log.Printf("Error listing org roles: %v\n", err)
		http.Error(w, "Error listing org roles: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiRoles []*shared.OrgRole
	for _, role := range roles {
		apiRoles = append(apiRoles, role.ToApi())
	}

	bytes, err := json.Marshal(apiRoles)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully listed org roles")

	w.Write(bytes)
}
