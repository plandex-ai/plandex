package handlers

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/types"
	"strings"

	"github.com/plandex/plandex/shared"
)

func authenticate(w http.ResponseWriter, r *http.Request, requireOrg bool) *types.ServerAuth {
	log.Println("authenticating request")

	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		log.Println("no auth header")
		http.Error(w, "no auth header", http.StatusUnauthorized)
		return nil
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		log.Println("invalid auth header")
		http.Error(w, "invalid auth header", http.StatusUnauthorized)
		return nil
	}

	// strip off the "Bearer " prefix
	encoded := strings.TrimPrefix(authHeader, "Bearer ")

	// decode the base64-encoded credentials
	bytes, err := base64.StdEncoding.DecodeString(encoded)

	if err != nil {
		log.Printf("error decoding auth token: %v\n", err)
		http.Error(w, "error decoding auth token", http.StatusUnauthorized)
		return nil
	}

	// parse the credentials
	var parsed shared.AuthHeader
	err = json.Unmarshal(bytes, &parsed)

	if err != nil {
		log.Printf("error parsing auth token: %v\n", err)
		http.Error(w, "error parsing auth token", http.StatusUnauthorized)
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
	permissionsMap := make(map[types.Permission]bool)
	for _, permission := range permissions {
		permissionsMap[types.Permission(permission)] = true
	}

	log.Printf("UserId: %s, Email: %s, OrgId: %s\n", authToken.UserId, user.Email, parsed.OrgId)

	return &types.ServerAuth{
		AuthToken:   authToken,
		User:        user,
		OrgId:       parsed.OrgId,
		Permissions: permissionsMap,
	}

}

func authorizeProject(w http.ResponseWriter, projectId string, auth *types.ServerAuth) bool {
	log.Println("authorizing project")

	projectExists, err := db.ProjectExists(auth.OrgId, projectId)

	if err != nil {
		log.Printf("error validating project: %v\n", err)
		http.Error(w, "error validating project", http.StatusInternalServerError)
		return false
	}

	if !projectExists {
		log.Println("project does not exist in org")
		http.Error(w, "project does not exist in org", http.StatusNotFound)
		return false
	}

	return true
}

func authorizeProjectRename(w http.ResponseWriter, projectId string, auth *types.ServerAuth) bool {
	if !authorizeProject(w, projectId, auth) {
		return false
	}

	if !auth.HasPermission(types.PermissionRenameAnyProject) {
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

	if !auth.HasPermission(types.PermissionDeleteAnyProject) {
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

	if plan.OwnerId != auth.User.Id && !auth.HasPermission(types.PermissionUpdateAnyPlan) {
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

	if plan.OwnerId != auth.User.Id && !auth.HasPermission(types.PermissionDeleteAnyPlan) {
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

	if plan.OwnerId != auth.User.Id && !auth.HasPermission(types.PermissionRenameAnyPlan) {
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

	if plan.OwnerId != auth.User.Id && !auth.HasPermission(types.PermissionArchiveAnyPlan) {
		log.Println("User does not have permission to archive plan")
		http.Error(w, "User does not have permission to archive plan", http.StatusForbidden)
		return nil
	}

	return plan
}
