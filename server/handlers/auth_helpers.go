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
		writeApiError(w, shared.ApiErr{
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

	if isMember {
		return &types.ServerAuth{
			AuthToken: authToken,
			User:      user,
			OrgId:     parsed.OrgId,
		}
	} else {
		// check if there's an invite for this user and accept it if so (adds the user to the org)
		invite, err := db.GetInviteForOrgUser(parsed.OrgId, authToken.UserId)

		if err != nil {
			log.Printf("error getting invite for org user: %v\n", err)
			http.Error(w, "error getting invite for org user", http.StatusInternalServerError)
			return nil
		}

		if invite != nil {
			err := db.AcceptInvite(invite, authToken.UserId)

			if err != nil {
				log.Printf("error accepting invite: %v\n", err)
				http.Error(w, "error accepting invite", http.StatusInternalServerError)
				return nil
			}

			return &types.ServerAuth{
				AuthToken: authToken,
				User:      user,
				OrgId:     parsed.OrgId,
			}
		} else {
			log.Println("user is not a member of the org")
			http.Error(w, "not a member of org", http.StatusUnauthorized)
			return nil
		}
	}

}

func authorizeProject(w http.ResponseWriter, projectId, userId, orgId string) bool {
	log.Println("authorizing project")

	hasProjectAccess, err := db.ValidateProjectAccess(projectId, userId, orgId)

	if err != nil {
		log.Printf("error validating project membership: %v\n", err)
		http.Error(w, "error validating project membership", http.StatusInternalServerError)
		return false
	}

	if !hasProjectAccess {
		log.Println("user is not a member of the project")
		http.Error(w, "not a member of project", http.StatusUnauthorized)
		return false
	}

	return true
}

func authorizePlan(w http.ResponseWriter, planId, userId, orgId string) *db.Plan {
	log.Println("authorizing plan")

	plan, err := db.ValidatePlanAccess(planId, userId, orgId)

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
