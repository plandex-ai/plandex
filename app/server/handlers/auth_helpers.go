package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/types"
	"strings"

	"github.com/plandex/plandex/shared"
)

func Authenticate(w http.ResponseWriter, r *http.Request, requireOrg bool) *types.ServerAuth {
	log.Println("authenticating request")

	parsed, err := GetAuthHeader(r)

	if err != nil {
		log.Printf("error getting auth header: %v\n", err)
		http.Error(w, "error getting auth header", http.StatusInternalServerError)
		return nil
	}

	if parsed == nil {
		log.Println("no auth header")
		http.Error(w, "no auth header", http.StatusUnauthorized)
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
		http.Error(w, "error getting user", http.StatusInternalServerError)
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
		http.Error(w, "no org id", http.StatusUnauthorized)
		return nil
	}

	// validate the org membership
	isMember, err := db.ValidateOrgMembership(authToken.UserId, parsed.OrgId)

	if err != nil {
		log.Printf("error validating org membership: %v\n", err)
		http.Error(w, "error validating org membership", http.StatusInternalServerError)
		return nil
	}

	if !isMember {
		// check if there's an invite for this user and accept it if so (adds the user to the org)
		invite, err := db.GetActiveInviteByEmail(parsed.OrgId, user.Email)

		if err != nil {
			log.Printf("error getting invite for org user: %v\n", err)
			http.Error(w, "error getting invite for org user", http.StatusInternalServerError)
			return nil
		}

		if invite != nil {
			log.Println("accepting invite")

			err := db.AcceptInvite(invite, authToken.UserId)

			if err != nil {
				log.Printf("error accepting invite: %v\n", err)
				http.Error(w, "error accepting invite", http.StatusInternalServerError)
				return nil
			}

		} else {
			log.Println("user is not a member of the org")
			http.Error(w, "not a member of org", http.StatusUnauthorized)
			return nil
		}
	}

	// get user permissions
	permissions, err := db.GetUserPermissions(authToken.UserId, parsed.OrgId)

	if err != nil {
		log.Printf("error getting user permissions: %v\n", err)
		http.Error(w, "error getting user permissions", http.StatusInternalServerError)
		return nil
	}

	// build the permissions map
	permissionsMap := make(shared.Permissions)
	for _, permission := range permissions {
		permissionsMap[permission] = true
	}

	log.Printf("UserId: %s, Email: %s, OrgId: %s\n", authToken.UserId, user.Email, parsed.OrgId)

	return &types.ServerAuth{
		AuthToken:   authToken,
		User:        user,
		OrgId:       parsed.OrgId,
		Permissions: permissionsMap,
	}

}

func GetAuthHeader(r *http.Request) (*shared.AuthHeader, error) {
	authHeader := r.Header.Get("Authorization")

	// check for a cookie as well for ui requests
	if authHeader == "" {
		// Try to get auth token from a cookie as a fallback
		cookie, err := r.Cookie("authToken")
		if err != nil {
			if err == http.ErrNoCookie {
				return nil, nil
			}
			return nil, fmt.Errorf("error retrieving auth cookie: %v", err)
		}
		// Use the token from the cookie as the fallback authorization header
		authHeader = cookie.Value
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
	bytes, err := base64.StdEncoding.DecodeString(encoded)

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

func SetAuthCookieIfBrowser(w http.ResponseWriter, r *http.Request, user *db.User, token, orgId string) error {
	isBrowser := false
	acceptHeader := r.Header.Get("Accept")
	if strings.Contains(acceptHeader, "text/html") {
		isBrowser = true
	}
	if !isBrowser {
		return nil
	}

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
	token = base64.StdEncoding.EncodeToString(bytes)

	http.SetCookie(w, &http.Cookie{
		Name:     "authToken",
		Value:    "Bearer " + token,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

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
	accounts := base64.StdEncoding.EncodeToString(bytes)

	http.SetCookie(w, &http.Cookie{
		Name:     "accounts",
		Value:    accounts,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
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

	bytes, err := base64.StdEncoding.DecodeString(accountsCookie.Value)
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
