package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"plandex/fs"
	"plandex/term"
	"plandex/types"
)

func MustResolveAuthWithOrg() {
	MustResolveAuth(true)
}

func MustResolveAuth(requireOrg bool) {
	if apiClient == nil {
		term.OutputErrorAndExit("error resolving auth: api client not set")
	}

	// load HomeAuthPath file into ClientAuth struct
	bytes, err := os.ReadFile(fs.HomeAuthPath)

	if err != nil {
		if os.IsNotExist(err) {
			err = promptInitialAuth()

			if err != nil {
				term.OutputErrorAndExit("error resolving auth: %v", err)
			}

			return
		} else {
			term.OutputErrorAndExit("error reading auth.json: %v", err)
		}
	}

	var auth types.ClientAuth
	err = json.Unmarshal(bytes, &auth)
	if err != nil {
		term.OutputErrorAndExit("error unmarshalling auth.json: %v", err)
	}

	Current = &auth

	if requireOrg && Current.OrgId == "" {
		term.StartSpinner("")
		orgs, apiErr := apiClient.ListOrgs()
		term.StopSpinner()

		if apiErr != nil {
			term.OutputErrorAndExit("Error listing orgs: %v", apiErr.Msg)
		}

		orgId, orgName, err := resolveOrgAuth(orgs)

		if err != nil {
			term.OutputErrorAndExit("Error resolving org: %v", err)
		}

		if orgId == "" {
			// still no org--exit now
			term.OutputErrorAndExit("No org")
		}

		Current.OrgId = orgId
		Current.OrgName = orgName

		err = writeCurrentAuth()

		if err != nil {
			term.OutputErrorAndExit("Error writing auth: %v", err)
		}
	}

}

func RefreshInvalidToken() error {
	if Current == nil {
		return fmt.Errorf("error refreshing token: auth not loaded")
	}

	hasAccount, pin, err := verifyEmail(Current.Email, Current.Host)

	if err != nil {
		return fmt.Errorf("error verifying email: %v", err)
	}

	if hasAccount {
		return signIn(Current.Email, pin, Current.Host)
	} else {
		host := Current.Host
		if host == "" {
			host = "Plandex Cloud"
		}

		term.OutputErrorAndExit("Account %s not found on %s", Current.Email, host)
	}

	return nil
}
