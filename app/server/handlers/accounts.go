package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"plandex-server/db"
	"strings"

	"github.com/plandex/plandex/shared"
)

func StartTrialHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for StartTrialHandler")

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

	b, err := shared.GetRandomAlphanumeric(6)
	if err != nil {
		log.Printf("Error generating random tag: %v\n", err)
		http.Error(w, "Error generating random tag: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tag := fmt.Sprintf("%x", b)
	tag = strings.ToLower(tag)

	user := &db.User{
		Name:    "Trial User " + tag,
		Email:   tag + "@trial.plandex.ai",
		Domain:  "trial.plandex.ai",
		IsTrial: true,
	}
	err = db.CreateUser(user, tx)

	if err != nil {
		log.Printf("Error creating user: %v\n", err)
		http.Error(w, "Error creating user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userId := user.Id

	// create a new org
	var orgId string
	orgName := "Trial Org " + tag
	err = tx.QueryRow("INSERT INTO orgs (name, owner_id, is_trial) VALUES ($1, $2, true) RETURNING id", orgName, userId).Scan(&orgId)

	if err != nil {
		log.Printf("Error creating org: %v\n", err)
		http.Error(w, "Error creating org: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// get org owner role id
	orgOwnerRoleId, err := db.GetOrgOwnerRoleId()

	if err != nil {
		log.Printf("Error getting org owner role: %v\n", err)
		http.Error(w, "Error getting org owner role: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// insert org user
	err = db.CreateOrgUser(orgId, userId, orgOwnerRoleId, tx)
	if err != nil {
		log.Printf("Error inserting org user: %v\n", err)
		http.Error(w, "Error inserting org user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// create auth token
	token, _, err := db.CreateAuthToken(userId, true, tx)

	if err != nil {
		log.Printf("Error creating auth token: %v\n", err)
		http.Error(w, "Error creating auth token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// commit transaction
	err = tx.Commit()
	if err != nil {
		log.Printf("Error committing transaction: %v\n", err)
		http.Error(w, "Error committing transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := shared.StartTrialResponse{
		UserId:   userId,
		OrgId:    orgId,
		Token:    token,
		UserName: user.Name,
		OrgName:  orgName,
		Email:    user.Email,
	}

	bytes, err := json.Marshal(resp)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully started trial")

	w.Write(bytes)
}

func CreateAccountHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreateAccountHandler")

	// read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var req shared.CreateAccountRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Printf("Error unmarshalling request: %v\n", err)
		http.Error(w, "Error unmarshalling request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	req.Email = strings.ToLower(req.Email)

	emailVerificationId, err := db.ValidateEmailVerification(req.Email, req.Pin)

	if err != nil {
		log.Printf("Error validating email verification: %v\n", err)
		http.Error(w, "Error validating email verification: "+err.Error(), http.StatusInternalServerError)
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

	// create user
	emailSplit := strings.Split(req.Email, "@")
	if len(emailSplit) != 2 {
		log.Printf("Invalid email: %v\n", req.Email)
		http.Error(w, "Invalid email: "+req.Email, http.StatusBadRequest)
		return
	}
	domain := emailSplit[1]

	user := db.User{
		Name:   req.UserName,
		Email:  req.Email,
		Domain: domain,
	}
	err = db.CreateUser(&user, tx)

	if err != nil {
		if db.IsNonUniqueErr(err) {
			log.Printf("User already exists for email: %v\n", req.Email)
			http.Error(w, "User already exists for email: "+req.Email, http.StatusConflict)
			return
		}

		log.Printf("Error creating user: %v\n", err)
		http.Error(w, "Error creating user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userId := user.Id

	// create auth token
	token, authTokenId, err := db.CreateAuthToken(userId, false, tx)

	if err != nil {
		log.Printf("Error creating auth token: %v\n", err)
		http.Error(w, "Error creating auth token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// update email verification with user and auth token ids
	_, err = tx.Exec("UPDATE email_verifications SET user_id = $1, auth_token_id = $2 WHERE id = $3", userId, authTokenId, emailVerificationId)

	if err != nil {
		log.Printf("Error updating email verification: %v\n", err)
		http.Error(w, "Error updating email verification: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// add to org matching domain if one exists and auto add domain users is true for that org
	org, err := db.GetOrgForDomain(domain)

	if err != nil {
		log.Printf("Error getting org for domain: %v\n", err)
		http.Error(w, "Error getting org for domain: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if org != nil && org.AutoAddDomainUsers {
		// get org owner role id
		orgOwnerRoleId, err := db.GetOrgOwnerRoleId()

		if err != nil {
			log.Printf("Error getting org owner role: %v\n", err)
			http.Error(w, "Error getting org owner role: "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = db.CreateOrgUser(org.Id, userId, orgOwnerRoleId, tx)

		if err != nil {
			log.Printf("Error adding org user: %v\n", err)
			http.Error(w, "Error adding org user: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// commit transaction
	err = tx.Commit()
	if err != nil {
		log.Printf("Error committing transaction: %v\n", err)
		http.Error(w, "Error committing transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// get orgs
	orgs, err := db.GetAccessibleOrgsForUser(&user)

	if err != nil {
		log.Printf("Error getting orgs for user: %v\n", err)
		http.Error(w, "Error getting orgs for user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiOrgs []*shared.Org
	for _, org := range orgs {
		apiOrgs = append(apiOrgs, org.ToApi())
	}

	resp := shared.SessionResponse{
		UserId:   userId,
		Token:    token,
		Email:    req.Email,
		UserName: req.UserName,
		Orgs:     apiOrgs,
	}

	bytes, err := json.Marshal(resp)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully created account")

	w.Write(bytes)
}

func ConvertTrialHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ConvertTrialHandler")

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	if !auth.User.IsTrial {
		log.Println("Trial isn't active")
		http.Error(w, "Trial isn't active", http.StatusBadRequest)
		return
	}

	var req shared.ConvertTrialRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Error unmarshalling request: %v\n", err)
		http.Error(w, "Error unmarshalling request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	emailVerificationId, err := db.ValidateEmailVerification(req.Email, req.Pin)

	if err != nil {
		log.Printf("Error validating email verification: %v\n", err)
		http.Error(w, "Error validating email verification: "+err.Error(), http.StatusInternalServerError)
		return
	}

	emailSplit := strings.Split(req.Email, "@")
	if len(emailSplit) != 2 {
		log.Printf("Invalid email: %v\n", req.Email)
		http.Error(w, "Invalid email: "+req.Email, http.StatusBadRequest)
		return
	}
	userDomain := emailSplit[1]
	var domain *string
	if req.OrgAutoAddDomainUsers {
		if shared.IsEmailServiceDomain(userDomain) {
			log.Printf("Invalid domain: %v\n", userDomain)
			http.Error(w, "Invalid domain: "+userDomain, http.StatusBadRequest)
			return
		}

		domain = &userDomain
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

	// clear current auth token
	_, err = db.Conn.Exec("UPDATE auth_tokens SET deleted_at = NOW() WHERE token_hash = $1", auth.AuthToken.TokenHash)

	if err != nil {
		log.Printf("Error deleting auth token: %v\n", err)
		http.Error(w, "Error deleting auth token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("UPDATE users SET name = $1, email = $2, domain = $3, is_trial = false WHERE id = $4", req.UserName, req.Email, userDomain, auth.User.Id)

	if err != nil {
		if db.IsNonUniqueErr(err) {
			log.Printf("User already exists for email: %v\n", req.Email)
			http.Error(w, "User already exists for email: "+req.Email, http.StatusConflict)
			return
		}
		log.Printf("Error updating user: %v\n", err)
		http.Error(w, "Error updating user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// create auth token
	token, authTokenId, err := db.CreateAuthToken(auth.User.Id, false, tx)

	if err != nil {
		log.Printf("Error creating auth token: %v\n", err)
		http.Error(w, "Error creating auth token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// update email verification with user and auth token ids
	_, err = tx.Exec("UPDATE email_verifications SET user_id = $1, auth_token_id = $2 WHERE id = $3", auth.User.Id, authTokenId, emailVerificationId)

	if err != nil {
		log.Printf("Error updating email verification: %v\n", err)
		http.Error(w, "Error updating email verification: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// update org
	_, err = tx.Exec("UPDATE orgs SET name = $1, domain = $2, auto_add_domain_users = $3, is_trial = false WHERE id = $4", req.OrgName, domain, req.OrgAutoAddDomainUsers, auth.OrgId)

	if err != nil {
		if db.IsNonUniqueErr(err) {
			log.Printf("Org already exists for domain: %v\n", userDomain)
			http.Error(w, "Org already exists for domain: "+userDomain, http.StatusConflict)
			return
		}
		log.Printf("Error updating org: %v\n", err)
		http.Error(w, "Error updating org: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// commit transaction
	err = tx.Commit()
	if err != nil {
		log.Printf("Error committing transaction: %v\n", err)
		http.Error(w, "Error committing transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// get orgs
	orgs, err := db.GetAccessibleOrgsForUser(auth.User)

	if err != nil {
		log.Printf("Error getting orgs for user: %v\n", err)
		http.Error(w, "Error getting orgs for user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiOrgs []*shared.Org
	for _, org := range orgs {
		apiOrgs = append(apiOrgs, org.ToApi())
	}

	resp := shared.SessionResponse{
		UserId:   auth.User.Id,
		Token:    token,
		Email:    req.Email,
		UserName: req.UserName,
		Orgs:     apiOrgs,
	}

	bytes, err := json.Marshal(resp)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully converted trial")

	w.Write(bytes)
}
