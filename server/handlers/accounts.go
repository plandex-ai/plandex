package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/email"
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

	name := "Trial User " + tag
	email := tag + "@trial.plandex.ai"

	var userId string
	err = tx.QueryRow("INSERT INTO users (name, email, is_trial) VALUES ($1, $2, true) RETURNING id", name, email).Scan(&userId)

	if err != nil {
		log.Printf("Error creating user: %v\n", err)
		http.Error(w, "Error creating user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// create a new org
	var orgId string
	orgName := "Trial Org " + tag
	err = tx.QueryRow("INSERT INTO orgs (name, owner_id, is_trial) VALUES ($1, $2, true) RETURNING id", orgName, userId).Scan(&orgId)

	if err != nil {
		log.Printf("Error creating org: %v\n", err)
		http.Error(w, "Error creating org: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// insert org user
	_, err = tx.Exec("INSERT INTO orgs_users (org_id, user_id) VALUES ($1, $2)", orgId, userId)
	if err != nil {
		log.Printf("Error inserting org user: %v\n", err)
		http.Error(w, "Error inserting org user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// create auth token
	token, _, err := db.CreateAuthToken(userId, tx)

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
		UserName: name,
		OrgName:  orgName,
		Email:    email,
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

func CreateEmailVerificationHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreateEmailVerificationHandler")

	// read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var req shared.CreateEmailVerificationRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Printf("Error unmarshalling request: %v\n", err)
		http.Error(w, "Error unmarshalling request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// create pin - 6 alphanumeric characters
	pinBytes, err := shared.GetRandomAlphanumeric(6)
	if err != nil {
		log.Printf("Error generating random pin: %v\n", err)
		http.Error(w, "Error generating random pin: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// get sha256 hash of pin
	hashBytes := sha256.Sum256(pinBytes)
	pinHash := hex.EncodeToString(hashBytes[:])

	// create verification
	err = db.CreateEmailVerification(req.Email, pinHash, req.UserId)

	if err != nil {
		log.Printf("Error creating email verification: %v\n", err)
		http.Error(w, "Error creating email verification: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = email.SendVerificationEmail(req.Email, string(pinBytes))

	if err != nil {
		log.Printf("Error sending verification email: %v\n", err)
		http.Error(w, "Error sending verification email: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully created email verification")

}

func SignInHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for SignInHandler")

	// read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var req shared.SignInRequest
	err = json.Unmarshal(body, &req)
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

	user, err := db.GetUserByEmail(req.Email)

	if err != nil {
		log.Printf("Error getting user: %v\n", err)
		http.Error(w, "Error getting user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if user == nil {
		log.Printf("User not found for email: %v\n", req.Email)
		http.Error(w, "Not found", http.StatusNotFound)
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

	// create auth token
	token, authTokenId, err := db.CreateAuthToken(user.Id, tx)

	if err != nil {
		log.Printf("Error creating auth token: %v\n", err)
		http.Error(w, "Error creating auth token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// update email verification with user and auth token ids
	_, err = tx.Exec("UPDATE email_verifications SET user_id = $1, auth_token_id = $2 WHERE id = $3", user.Id, authTokenId, emailVerificationId)

	if err != nil {
		log.Printf("Error updating email verification: %v\n", err)
		http.Error(w, "Error updating email verification: "+err.Error(), http.StatusInternalServerError)
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
	orgs, err := db.GetOrgsForUser(user.Id)

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
		UserId:   user.Id,
		Token:    token,
		Email:    user.Email,
		UserName: user.Name,
		Orgs:     apiOrgs,
	}

	bytes, err := json.Marshal(resp)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully signed in")

	w.Write(bytes)
}

func SignOutHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for SignOutHandler")

	auth := authenticate(w, r, false)
	if auth == nil {
		return
	}

	_, err := db.Conn.Exec("UPDATE auth_tokens SET deleted_at = NOW() WHERE token_hash = $1", auth.TokenHash)

	if err != nil {
		log.Printf("Error deleting auth token: %v\n", err)
		http.Error(w, "Error deleting auth token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully signed out")
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
	var userId string
	err = tx.QueryRow("INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id", req.UserName, req.Email).Scan(&userId)

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

	// create auth token
	token, authTokenId, err := db.CreateAuthToken(userId, tx)

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

	// commit transaction
	err = tx.Commit()
	if err != nil {
		log.Printf("Error committing transaction: %v\n", err)
		http.Error(w, "Error committing transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := shared.SessionResponse{
		UserId:   userId,
		Token:    token,
		Email:    req.Email,
		UserName: req.UserName,
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

func CreateOrgHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreateOrgHandler")

	auth := authenticate(w, r, false)
	if auth == nil {
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

	if req.Domain != "" {
		// ensure user email matches domain
		user, err := db.GetUser(auth.UserId)

		if err != nil {
			log.Printf("Error getting user: %v\n", err)
			http.Error(w, "Error getting user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		split := strings.Split(user.Email, "@")
		userDomain := split[1]

		if userDomain != req.Domain {
			log.Printf("User email domain does not match request domain: %v\n", req.Domain)
			http.Error(w, "User email domain does not match request domain: "+req.Domain, http.StatusBadRequest)
			return
		}
	}

	// create a new org
	org, err := db.CreateOrg(&req, auth.UserId)

	if err != nil {
		log.Printf("Error creating org: %v\n", err)
		http.Error(w, "Error creating org: "+err.Error(), http.StatusInternalServerError)
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
