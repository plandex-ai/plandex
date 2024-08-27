package handlers

import (
	"log"
	"plandex-server/db"
	"plandex-server/hooks"

	"github.com/plandex/plandex/shared"
)

func toApiOrgs(orgs []*db.Org) ([]*shared.Org, *shared.ApiError) {
	var orgIds []string
	for _, org := range orgs {
		orgIds = append(orgIds, org.Id)
	}

	hookRes, apiErr := hooks.ExecHook(hooks.GetApiOrgs, hooks.HookParams{
		GetApiOrgIds: orgIds,
	})

	if apiErr != nil {
		log.Printf("Error getting integrated models mode by org id: %v\n", apiErr)
		return nil, apiErr
	}

	var apiOrgs []*shared.Org
	for _, org := range orgs {
		if hookRes.ApiOrgsById != nil {
			hookApiOrg := hookRes.ApiOrgsById[org.Id]
			apiOrgs = append(apiOrgs, hookApiOrg)
		} else {
			apiOrgs = append(apiOrgs, org.ToApi())
		}
	}

	return apiOrgs, nil
}

func getApiOrg(orgId string) (*shared.Org, *shared.ApiError) {
	org, err := db.GetOrg(orgId)
	if err != nil {
		log.Printf("Error getting org: %v\n", err)
		return nil, &shared.ApiError{
			Type: shared.ApiErrorTypeOther,
			Msg:  "Error getting org",
		}
	}

	hookRes, apiErr := hooks.ExecHook(hooks.GetApiOrgs, hooks.HookParams{
		GetApiOrgIds: []string{org.Id},
	})

	if apiErr != nil {
		log.Printf("Error getting integrated models mode by org id: %v\n", apiErr)
		return nil, apiErr
	}

	if hookRes.ApiOrgsById != nil {
		return hookRes.ApiOrgsById[org.Id], nil
	}

	return org.ToApi(), nil
}
