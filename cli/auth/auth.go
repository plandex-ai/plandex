package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"plandex/fs"
	"plandex/types"
)

func MustResolveAuthWithOrg() {
	MustResolveAuth(true)
}

func MustResolveAuth(requireOrg bool) {
	if apiClient == nil {
		panic(fmt.Errorf("error resolving auth: api client not set"))
	}

	// load HomeAuthPath file into ClientAuth struct
	bytes, err := os.ReadFile(fs.HomeAuthPath)

	if err != nil {
		if os.IsNotExist(err) {
			err = promptInitialAuth()

			if err != nil {
				panic(fmt.Errorf("error resolving auth: %v", err))
			}

			return
		} else {
			panic(fmt.Errorf("error reading auth.json: %v", err))
		}
	}

	var auth types.ClientAuth
	err = json.Unmarshal(bytes, &auth)
	if err != nil {
		panic(fmt.Errorf("error unmarshalling auth.json: %v", err))
	}

	Current = &auth

	if requireOrg && Current.OrgId == "" {
		orgs, apiErr := apiClient.ListOrgs()

		if apiErr != nil {
			panic(fmt.Errorf("error listing orgs: %v", apiErr.Msg))
		}

		orgId, orgName, err := resolveOrgAuth(orgs)

		if err != nil {
			panic(fmt.Errorf("error resolving org: %v", err))
		}

		if orgId == "" {
			// still no org--exit now
			os.Exit(1)
		}

		Current.OrgId = orgId
		Current.OrgName = orgName

		err = writeCurrentAuth()

		if err != nil {
			panic(fmt.Errorf("error writing auth: %v", err))
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

		fmt.Printf("🚨 Account %s not found on %s\n", Current.Email, host)
		os.Exit(1)
	}

	return nil
}
