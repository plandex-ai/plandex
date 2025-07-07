package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"plandex-server/db"
	"plandex-server/email"
	"strings"

	shared "plandex-shared"
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

		user, err := db.GetUser(req.UserId)

		if err != nil {
			log.Printf("Error getting user: %v\n", err)
			http.Error(w, "Error getting user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if user == nil {
			log.Printf("User not found for id: %v\n", req.UserId)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		if user.Email != req.Email {
			log.Printf("User email does not match for id: %v\n", req.UserId)
			http.Error(w, "User email does not match", http.StatusBadRequest)
			return
		}
	}

	if req.RequireUser && !hasAccount {
		log.Printf("User not found for email: %v\n", req.Email)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if req.RequireNoUser && hasAccount {
		log.Printf("User already exists for email: %v\n", req.Email)
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	var res shared.CreateEmailVerificationResponse

	if !(os.Getenv("GOENV") == "development" && os.Getenv("LOCAL_MODE") == "1") {
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

		res = shared.CreateEmailVerificationResponse{
			HasAccount: hasAccount,
		}
	} else {
		res = shared.CreateEmailVerificationResponse{
			HasAccount:  hasAccount,
			IsLocalMode: true,
		}
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

func CheckEmailPinHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for VerifyEmailPinHandler")

	// read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var req shared.VerifyEmailPinRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Printf("Error unmarshalling request: %v\n", err)
		http.Error(w, "Error unmarshalling request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	req.Email = strings.ToLower(req.Email)

	_, err = db.ValidateEmailVerification(req.Email, req.Pin)

	if err != nil {
		if err.Error() == db.InvalidOrExpiredPinError {
			http.Error(w, "Invalid or expired pin", http.StatusNotFound)
			return
		}

		log.Printf("Error validating email verification: %v\n", err)
		http.Error(w, "Error validating email verification: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully verified email pin")
}

// sign in codes allow users to authenticate between different clients
// like UI to CLI or vice versa
func CreateSignInCodeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreateSignInCodeHandler")

	auth := Authenticate(w, r, true)

	if auth == nil {
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

	err = db.CreateSignInCode(auth.User.Id, auth.OrgId, pinHash)

	if err != nil {
		log.Printf("Error creating sign in code: %v\n", err)
		http.Error(w, "Error creating sign in code: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully created sign in code")

	// return the pin as a response
	w.Write(pinBytes)
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

	log.Println("Validating and signing in")
	resp, err := ValidateAndSignIn(w, r, req)

	if err != nil {
		log.Printf("Error signing in: %v\n", err)
		http.Error(w, "Error signing in: "+err.Error(), http.StatusInternalServerError)
		return
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

	auth := Authenticate(w, r, false)
	if auth == nil {
		return
	}

	_, err := db.Conn.Exec("UPDATE auth_tokens SET deleted_at = NOW() WHERE token_hash = $1", auth.AuthToken.TokenHash)

	if err != nil {
		log.Printf("Error deleting auth token: %v\n", err)
		http.Error(w, "Error deleting auth token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = ClearAuthCookieIfBrowser(w, r)

	if err != nil {
		log.Printf("Error clearing auth cookie: %v\n", err)
		http.Error(w, "Error clearing auth cookie: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = ClearAccountFromCookies(w, r, auth.User.Id)

	if err != nil {
		log.Printf("Error clearing account from cookies: %v\n", err)
		http.Error(w, "Error clearing account from cookies: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully signed out")
}

func GetOrgUserConfigHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for GetOrgUserConfigHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	orgUserConfig, err := db.GetOrgUserConfig(auth.User.Id, auth.OrgId)

	if err != nil {
		log.Printf("Error getting org user config: %v\n", err)
		http.Error(w, "Error getting org user config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(orgUserConfig)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(bytes)
}

func UpdateOrgUserConfigHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for UpdateOrgUserConfigHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var req shared.OrgUserConfig
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Printf("Error unmarshalling request: %v\n", err)
		http.Error(w, "Error unmarshalling request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = db.UpdateOrgUserConfig(auth.User.Id, auth.OrgId, &req)

	if err != nil {
		log.Printf("Error updating org user config: %v\n", err)
		http.Error(w, "Error updating org user config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully updated org user config")
}
