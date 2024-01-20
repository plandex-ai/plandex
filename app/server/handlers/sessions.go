package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/email"
	"strings"

	"github.com/plandex/plandex/shared"
)

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
	req.Email = strings.ToLower(req.Email)

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
	err = db.CreateEmailVerification(req.Email, req.UserId, pinHash)

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

	var hasAccount bool
	if req.UserId == "" {
		user, err := db.GetUserByEmail(req.Email)

		if err != nil {
			log.Printf("Error getting user: %v\n", err)
			http.Error(w, "Error getting user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		hasAccount = user != nil
	} else {
		hasAccount = true
	}

	res := shared.CreateEmailVerificationResponse{
		HasAccount: hasAccount,
	}

	bytes, err := json.Marshal(res)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully created email verification")

	w.Write(bytes)
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
	req.Email = strings.ToLower(req.Email)

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

	if user != nil && user.IsTrial {
		log.Printf("Trial user can't sign in: %v\n", req.Email)
		http.Error(w, "Trial user can't sign in", http.StatusForbidden)
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

	// create auth token
	token, authTokenId, err := db.CreateAuthToken(user.Id, false, tx)

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
	orgs, err := db.GetAccessibleOrgsForUser(user)

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

	_, err := db.Conn.Exec("UPDATE auth_tokens SET deleted_at = NOW() WHERE token_hash = $1", auth.AuthToken.TokenHash)

	if err != nil {
		log.Printf("Error deleting auth token: %v\n", err)
		http.Error(w, "Error deleting auth token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully signed out")
}
