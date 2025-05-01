package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/types"
	"strings"
	"time"

	shared "plandex-shared"

	"github.com/jmoiron/sqlx"
)

func Authenticate(w http.ResponseWriter, r *http.Request, requireOrg bool) *types.ServerAuth {
	return execAuthenticate(w, r, requireOrg, true)
}

func AuthenticateOptional(w http.ResponseWriter, r *http.Request, requireOrg bool) *types.ServerAuth {
	return execAuthenticate(w, r, requireOrg, false)
}

func GetAuthHeader(r *http.Request) (*shared.AuthHeader, error) {
	authHeader := r.Header.Get("Authorization")

	// check for a cookie as well for ui requests
	if authHeader == "" {
		log.Println("no auth header - checking for cookie")

		// Try to get auth token from a cookie as a fallback
		cookie, err := r.Cookie("authToken")
		if err != nil {
			if err == http.ErrNoCookie {
				log.Println("no auth cookie")
				return nil, nil
			}
			return nil, fmt.Errorf("error retrieving auth cookie: %v", err)
		}
		// Use the token from the cookie as the fallback authorization header
		authHeader = cookie.Value
		log.Println("got auth header from cookie")
	}

	if authHeader == "" {
		return nil, nil
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("invalid auth header")
	}

	// strip off the "Bearer " prefix
	encoded := strings.TrimPrefix(authHeader, "Bearer ")

	// decode the base64-encoded credentials
	bytes, err := base64.URLEncoding.DecodeString(encoded)

	if err != nil {
		return nil, fmt.Errorf("error decoding auth token: %v", err)
	}

	// parse the credentials
	var parsed shared.AuthHeader
	err = json.Unmarshal(bytes, &parsed)

	if err != nil {
		return nil, fmt.Errorf("error parsing auth token: %v", err)
	}

	return &parsed, nil
}

func ClearAuthCookieIfBrowser(w http.ResponseWriter, r *http.Request) error {
	acceptHeader := r.Header.Get("Accept")
	if acceptHeader == "" {
		// no accept header, not a browser request
		return nil
	}

	// Check for existing auth cookie
	_, err := r.Cookie("authToken")
	if err == http.ErrNoCookie {
		// No auth cookie, nothing to clear
		return nil
	}
	if err != nil {
		return fmt.Errorf("error retrieving auth cookie: %v", err)
	}

	var domain string
	if os.Getenv("GOENV") == "production" {
		domain = os.Getenv("APP_SUBDOMAIN") + ".plandex.ai"
	}

	// Clear the authToken cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "authToken",
		Path:     "/",
		Value:    "",
		MaxAge:   -1,
		Secure:   os.Getenv("GOENV") != "development",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Domain:   domain,
	})

	log.Println("cleared auth cookie")

	return nil
}

func ClearAccountFromCookies(w http.ResponseWriter, r *http.Request, userId string) error {
	// Get stored accounts
	storedAccounts, err := GetAccountsFromCookie(r)
	if err != nil {
		return fmt.Errorf("error getting accounts from cookie: %v", err)
	}

	// Remove the account with the given userId
	for i, account := range storedAccounts {
		if account.UserId == userId {
			storedAccounts = append(storedAccounts[:i], storedAccounts[i+1:]...)
			break
		}
	}

	// Marshal the updated accounts
	updatedAccountsBytes, err := json.Marshal(storedAccounts)
	if err != nil {
		return fmt.Errorf("error marshalling updated accounts: %v", err)
	}

	// Encode to base64
	encodedAccounts := base64.URLEncoding.EncodeToString(updatedAccountsBytes)

	// Set the updated accounts cookie
	var domain string
	if os.Getenv("GOENV") == "production" {
		domain = os.Getenv("APP_SUBDOMAIN") + ".plandex.ai"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "accounts",
		Path:     "/",
		Value:    encodedAccounts,
		Secure:   os.Getenv("GOENV") != "development",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Domain:   domain,
	})

	return nil
}

func SetAuthCookieIfBrowser(w http.ResponseWriter, r *http.Request, user *db.User, token, orgId string) error {
	log.Println("setting auth cookie if browser")

	acceptHeader := r.Header.Get("Accept")
	if acceptHeader == "" {
		// no accept header, not a browser request
		log.Println("not a browser request")
		return nil
	}

	log.Println("is browser - setting auth cookie")

	if token == "" {
		authHeader, err := GetAuthHeader(r)
		if err != nil {
			return fmt.Errorf("error getting auth header: %v", err)
		}
		token = authHeader.Token
	}

	if token == "" {
		return fmt.Errorf("no token")
	}

	// set authToken cookie
	authHeader := shared.AuthHeader{
		Token: token,
		OrgId: orgId,
	}

	bytes, err := json.Marshal(authHeader)

	if err != nil {
		return fmt.Errorf("error marshalling auth header: %v", err)
	}

	// base64 encode
	token = base64.URLEncoding.EncodeToString(bytes)

	var domain string
	if os.Getenv("GOENV") == "production" {
		domain = os.Getenv("APP_SUBDOMAIN") + ".plandex.ai"
	}

	cookie := &http.Cookie{
		Name:     "authToken",
		Path:     "/",
		Value:    "Bearer " + token,
		Secure:   os.Getenv("GOENV") != "development",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Domain:   domain,
		Expires:  time.Now().Add(time.Hour * 24 * 90),
	}

	log.Println("setting auth cookie", cookie)

	http.SetCookie(w, cookie)

	storedAccounts, err := GetAccountsFromCookie(r)

	if err != nil {
		return fmt.Errorf("error getting accounts from cookie: %v", err)
	}

	found := false
	for _, account := range storedAccounts {
		if account.UserId == user.Id {
			found = true

			account.Token = token
			account.Email = user.Email
			account.UserName = user.Name
			break
		}
	}

	if !found {
		storedAccounts = append(storedAccounts, &shared.ClientAccount{
			Email:    user.Email,
			UserName: user.Name,
			UserId:   user.Id,
			Token:    token,
		})
	}

	bytes, err = json.Marshal(storedAccounts)

	if err != nil {
		return fmt.Errorf("error marshalling accounts: %v", err)
	}

	// base64 encode
	accounts := base64.URLEncoding.EncodeToString(bytes)

	http.SetCookie(w, &http.Cookie{
		Name:     "accounts",
		Path:     "/",
		Value:    accounts,
		Secure:   os.Getenv("GOENV") != "development",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Domain:   domain,
		Expires:  time.Now().Add(time.Hour * 24 * 90),
	})

	return nil
}

func GetAccountsFromCookie(r *http.Request) ([]*shared.ClientAccount, error) {
	accountsCookie, err := r.Cookie("accounts")

	if err == http.ErrNoCookie {
		return []*shared.ClientAccount{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("error getting accounts cookie: %v", err)
	}

	bytes, err := base64.URLEncoding.DecodeString(accountsCookie.Value)
	if err != nil {
		return nil, fmt.Errorf("error decoding accounts cookie: %v", err)
	}

	var accounts []*shared.ClientAccount
	err = json.Unmarshal(bytes, &accounts)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling accounts cookie: %v", err)
	}

	return accounts, nil
}

func ValidateAndSignIn(w http.ResponseWriter, r *http.Request, req shared.SignInRequest) (*shared.SessionResponse, error) {
	var user *db.User
	var emailVerificationId string
	var signInCodeId string
	var signInCodeOrgId string
	var err error

	isLocalMode := (os.Getenv("GOENV") == "development" && os.Getenv("LOCAL_MODE") == "1")

	if req.IsSignInCode {
		res, err := db.ValidateSignInCode(req.Pin)

		if err != nil {
			log.Printf("Error validating sign in code: %v\n", err)
			return nil, fmt.Errorf("error validating sign in code: %v", err)
		}

		user, err = db.GetUser(res.UserId)

		if err != nil {
			log.Printf("Error getting user: %v\n", err)
			return nil, fmt.Errorf("error getting user: %v", err)
		}

		if user == nil {
			log.Printf("User not found for id: %v\n", res.UserId)
			return nil, fmt.Errorf("user not found")
		}

		signInCodeId = res.Id
		signInCodeOrgId = res.OrgId
	} else {
		req.Email = strings.ToLower(req.Email)
		user, err = db.GetUserByEmail(req.Email)

		if err != nil {
			log.Printf("Error getting user: %v\n", err)
			return nil, fmt.Errorf("error getting user: %v", err)
		}

		if user == nil {
			log.Printf("User not found for email: %v\n", req.Email)
			return nil, fmt.Errorf("not found")
		}

		// only validate email in non-local mode
		if !isLocalMode {
			emailVerificationId, err = db.ValidateEmailVerification(req.Email, req.Pin)

			if err != nil {
				log.Printf("Error validating email verification: %v\n", err)
				return nil, fmt.Errorf("error validating email verification: %v", err)
			}

			log.Println("Email verification successful")
		}
	}

	var token string
	var authTokenId string

	err = db.WithTx(r.Context(), "validate and sign in", func(tx *sqlx.Tx) error {
		var err error
		// create auth token
		token, authTokenId, err = db.CreateAuthToken(user.Id, tx)

		if err != nil {
			log.Printf("Error creating auth token: %v\n", err)
			return fmt.Errorf("error creating auth token: %v", err)
		}

		if req.IsSignInCode {
			// update sign in code with auth token id
			_, err = tx.Exec("UPDATE sign_in_codes SET auth_token_id = $1 WHERE id = $2", authTokenId, signInCodeId)

			if err != nil {
				log.Printf("Error updating sign in code: %v\n", err)
				return fmt.Errorf("error updating sign in code: %v", err)
			}
		} else if !isLocalMode { // only update email verification in non-local mode
			// update email verification with user and auth token ids
			_, err = tx.Exec("UPDATE email_verifications SET user_id = $1, auth_token_id = $2 WHERE id = $3", user.Id, authTokenId, emailVerificationId)

			if err != nil {
				log.Printf("Error updating email verification: %v\n", err)
				return fmt.Errorf("error updating email verification: %v", err)
			}

			log.Println("Email verification updated")
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error validating and signing in: %v", err)
	}

	// get orgs
	orgs, err := db.GetAccessibleOrgsForUser(user)

	if err != nil {
		log.Printf("Error getting orgs for user: %v\n", err)
		return nil, fmt.Errorf("error getting orgs for user: %v", err)
	}

	if req.IsSignInCode {
		filteredOrgs := []*db.Org{}
		for _, org := range orgs {
			if org.Id == signInCodeOrgId {
				filteredOrgs = append(filteredOrgs, org)
			}
		}
		orgs = filteredOrgs
	}

	// with a single org, set the orgId in the cookie
	// otherwise, the user will be prompted to select an org
	var orgId string
	if len(orgs) == 1 {
		orgId = orgs[0].Id
	}

	log.Println("Setting auth cookie if browser")
	err = SetAuthCookieIfBrowser(w, r, user, token, orgId)
	if err != nil {
		log.Printf("Error setting auth cookie: %v\n", err)
		return nil, fmt.Errorf("error setting auth cookie: %v", err)
	}

	apiOrgs, apiErr := toApiOrgs(orgs)

	if apiErr != nil {
		log.Printf("Error converting orgs to api orgs: %v\n", apiErr)
		return nil, fmt.Errorf("error converting orgs to api orgs: %v", apiErr)
	}

	resp := shared.SessionResponse{
		UserId:      user.Id,
		Token:       token,
		Email:       user.Email,
		UserName:    user.Name,
		Orgs:        apiOrgs,
		IsLocalMode: os.Getenv("GOENV") == "development" && os.Getenv("LOCAL_MODE") == "1",
	}

	return &resp, nil
}

func execAuthenticate(w http.ResponseWriter, r *http.Request, requireOrg bool, raiseErr bool) *types.ServerAuth {
	log.Println("authenticating request")

	parsed, err := GetAuthHeader(r)

	if err != nil {
		log.Printf("error getting auth header: %v\n", err)
		if raiseErr {
			http.Error(w, "error getting auth header", http.StatusInternalServerError)
		}
		return nil
	}

	if parsed == nil {
		log.Println("no auth header")
		if raiseErr {
			http.Error(w, "no auth header", http.StatusUnauthorized)
		}
		return nil
	}

	// validate the token
	authToken, err := db.ValidateAuthToken(parsed.Token)

	if err != nil {
		log.Printf("error validating auth token: %v\n", err)

		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeInvalidToken,
			Status: http.StatusUnauthorized,
			Msg:    "Invalid auth token",
		})
		return nil
	}

	user, err := db.GetUser(authToken.UserId)

	if err != nil {
		log.Printf("error getting user: %v\n", err)
		if raiseErr {
			http.Error(w, "error getting user", http.StatusInternalServerError)
		}
		return nil
	}

	if !requireOrg {
		return &types.ServerAuth{
			AuthToken: authToken,
			User:      user,
		}
	}

	if parsed.OrgId == "" {
		log.Println("no org id")
		if raiseErr {
			http.Error(w, "no org id", http.StatusUnauthorized)
		}
		return nil
	}

	// validate the org membership
	isMember, err := db.ValidateOrgMembership(authToken.UserId, parsed.OrgId)

	if err != nil {
		log.Printf("error validating org membership: %v\n", err)
		if raiseErr {
			http.Error(w, "error validating org membership", http.StatusInternalServerError)
		}
		return nil
	}

	if !isMember {
		// check if there's an invite for this user and accept it if so (adds the user to the org)
		invite, err := db.GetActiveInviteByEmail(parsed.OrgId, user.Email)

		if err != nil {
			log.Printf("error getting invite for org user: %v\n", err)
			if raiseErr {
				http.Error(w, "error getting invite for org user", http.StatusInternalServerError)
			}
			return nil
		}

		if invite != nil {
			log.Println("accepting invite")

			err := db.AcceptInvite(r.Context(), invite, authToken.UserId)

			if err != nil {
				log.Printf("error accepting invite: %v\n", err)
				if raiseErr {
					http.Error(w, "error accepting invite", http.StatusInternalServerError)
				}
				return nil
			}

		} else {
			log.Println("user is not a member of the org")
			if raiseErr {
				http.Error(w, "not a member of org", http.StatusUnauthorized)
			}
			return nil
		}
	}

	// get user permissions
	permissions, err := db.GetUserPermissions(authToken.UserId, parsed.OrgId)

	if err != nil {
		log.Printf("error getting user permissions: %v\n", err)
		if raiseErr {
			http.Error(w, "error getting user permissions", http.StatusInternalServerError)
		}
		return nil
	}

	// build the permissions map
	permissionsMap := make(shared.Permissions)
	for _, permission := range permissions {
		permissionsMap[permission] = true
	}

	auth := &types.ServerAuth{
		AuthToken:   authToken,
		User:        user,
		OrgId:       parsed.OrgId,
		Permissions: permissionsMap,
	}

	// don't send hash for org-session requests
	var hash string
	if r.URL.Path != "/orgs/session" {
		hash = parsed.Hash
	}

	_, apiErr := hooks.ExecHook(hooks.Authenticate, hooks.HookParams{
		Auth: auth,
		AuthenticateHookRequestParams: &hooks.AuthenticateHookRequestParams{
			Path: r.URL.Path,
			Hash: hash,
		},
	})

	if apiErr != nil {
		writeApiError(w, *apiErr)
		return nil
	}

	log.Printf("UserId: %s, Email: %s, OrgId: %s\n", authToken.UserId, user.Email, parsed.OrgId)

	return auth

}

func authorizeProject(w http.ResponseWriter, projectId string, auth *types.ServerAuth) bool {
	return authorizeProjectOptional(w, projectId, auth, true)
}

func authorizeProjectOptional(w http.ResponseWriter, projectId string, auth *types.ServerAuth, shouldErr bool) bool {
	log.Println("authorizing project")

	projectExists, err := db.ProjectExists(auth.OrgId, projectId)

	if err != nil {
		log.Printf("error validating project: %v\n", err)
		http.Error(w, "error validating project", http.StatusInternalServerError)
		return false
	}

	if !projectExists && shouldErr {
		log.Println("project does not exist in org")
		http.Error(w, "project does not exist in org", http.StatusNotFound)
		return false
	}

	return projectExists
}

func authorizeProjectRename(w http.ResponseWriter, projectId string, auth *types.ServerAuth) bool {
	if !authorizeProject(w, projectId, auth) {
		return false
	}

	if !auth.HasPermission(shared.PermissionRenameAnyProject) {
		log.Println("User does not have permission to rename project")
		http.Error(w, "User does not have permission to rename project", http.StatusForbidden)
		return false
	}

	return true
}

func authorizeProjectDelete(w http.ResponseWriter, projectId string, auth *types.ServerAuth) bool {
	if !authorizeProject(w, projectId, auth) {
		return false
	}

	if !auth.HasPermission(shared.PermissionDeleteAnyProject) {
		log.Println("User does not have permission to delete project")
		http.Error(w, "User does not have permission to delete project", http.StatusForbidden)
		return false
	}

	return true
}

func authorizePlan(w http.ResponseWriter, planId string, auth *types.ServerAuth) *db.Plan {
	log.Println("authorizing plan")

	plan, err := db.ValidatePlanAccess(planId, auth.User.Id, auth.OrgId)

	if err != nil {
		log.Printf("error validating plan membership: %v\n", err)
		http.Error(w, "error validating plan membership", http.StatusInternalServerError)
		return nil
	}

	if plan == nil {
		log.Println("user doesn't have access the plan")
		http.Error(w, "no access to plan", http.StatusUnauthorized)
		return nil
	}

	return plan
}

func authorizePlanUpdate(w http.ResponseWriter, planId string, auth *types.ServerAuth) *db.Plan {
	plan := authorizePlan(w, planId, auth)

	if plan == nil {
		return nil
	}

	if plan.OwnerId != auth.User.Id && !auth.HasPermission(shared.PermissionUpdateAnyPlan) {
		log.Println("User does not have permission to update plan")
		http.Error(w, "User does not have permission to update plan", http.StatusForbidden)
		return nil
	}

	return plan
}

func authorizePlanDelete(w http.ResponseWriter, planId string, auth *types.ServerAuth) *db.Plan {
	plan := authorizePlan(w, planId, auth)

	if plan == nil {
		return nil
	}

	if plan.OwnerId != auth.User.Id && !auth.HasPermission(shared.PermissionDeleteAnyPlan) {
		log.Println("User does not have permission to delete plan")
		http.Error(w, "User does not have permission to delete plan", http.StatusForbidden)
		return nil
	}

	return plan
}

func authorizePlanRename(w http.ResponseWriter, planId string, auth *types.ServerAuth) *db.Plan {
	plan := authorizePlan(w, planId, auth)

	if plan == nil {
		return nil
	}

	if plan.OwnerId != auth.User.Id && !auth.HasPermission(shared.PermissionRenameAnyPlan) {
		log.Println("User does not have permission to rename plan")
		http.Error(w, "User does not have permission to rename plan", http.StatusForbidden)
		return nil
	}

	return plan
}

func authorizePlanArchive(w http.ResponseWriter, planId string, auth *types.ServerAuth) *db.Plan {
	plan := authorizePlan(w, planId, auth)

	if plan == nil {
		return nil
	}

	if plan.OwnerId != auth.User.Id && !auth.HasPermission(shared.PermissionArchiveAnyPlan) {
		log.Println("User does not have permission to archive plan")
		http.Error(w, "User does not have permission to archive plan", http.StatusForbidden)
		return nil
	}

	return plan
}
