package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/hooks"
	"strings"

	"github.com/plandex/plandex/shared"
)

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

	var orgId string
	if org != nil {
		orgId = org.Id
	}

	err = hooks.ExecHook(hooks.CreateAccount, hooks.HookParams{
		W:     w,
		User:  &user,
		OrgId: orgId,
	})
	if err != nil {
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
