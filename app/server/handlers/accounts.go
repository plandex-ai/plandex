package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/types"
	"strings"

	shared "plandex-shared"

	"github.com/jmoiron/sqlx"
)

func CreateAccountHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreateAccountHandler")

	if os.Getenv("IS_CLOUD") != "" {
		log.Println("Creating accounts is not supported in cloud mode")
		http.Error(w, "Creating accounts is not supported in cloud mode", http.StatusNotImplemented)
		return
	}

	isLocalMode := (os.Getenv("GOENV") == "development" && os.Getenv("LOCAL_MODE") == "1")

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

	var emailVerificationId string

	// skipping email verification in dev/local mode
	if !isLocalMode {
		emailVerificationId, err = db.ValidateEmailVerification(req.Email, req.Pin)

		if err != nil {
			log.Printf("Error validating email verification: %v\n", err)
			http.Error(w, "Error validating email verification: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	var apiErr *shared.ApiError
	var user *db.User
	var userId string
	var token string
	var orgId string

	err = db.WithTx(r.Context(), "create account", func(tx *sqlx.Tx) error {
		res, err := db.CreateAccount(req.UserName, req.Email, emailVerificationId, tx)

		if err != nil {
			return fmt.Errorf("error creating account: %v", err)
		}

		user = res.User
		userId = user.Id
		token = res.Token
		orgId = res.OrgId

		_, apiErr = hooks.ExecHook(hooks.CreateAccount, hooks.HookParams{
			Auth: &types.ServerAuth{
				User:  user,
				OrgId: orgId,
			},
		})

		return nil
	})

	if apiErr != nil {
		writeApiError(w, *apiErr)
		return
	}

	if err != nil {
		log.Printf("Error creating account: %v\n", err)
		http.Error(w, "Error creating account: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// get orgs
	orgs, err := db.GetAccessibleOrgsForUser(user)

	if err != nil {
		log.Printf("Error getting orgs for user: %v\n", err)
		http.Error(w, "Error getting orgs for user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	apiOrgs, apiErr := toApiOrgs(orgs)

	if apiErr != nil {
		log.Printf("Error converting orgs to API orgs: %v\n", apiErr)
		writeApiError(w, *apiErr)
		return
	}

	resp := shared.SessionResponse{
		UserId:      userId,
		Token:       token,
		Email:       req.Email,
		UserName:    req.UserName,
		Orgs:        apiOrgs,
		IsLocalMode: os.Getenv("GOENV") == "development" && os.Getenv("LOCAL_MODE") == "1",
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
