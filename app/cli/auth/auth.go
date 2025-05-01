package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"plandex-cli/fs"
	"plandex-cli/term"

	shared "plandex-shared"
)

var openUnauthenticatedCloudURL func(msg, path string)
var openAuthenticatedURL func(msg, path string)

func SetOpenUnauthenticatedCloudURLFn(fn func(msg, path string)) {
	openUnauthenticatedCloudURL = fn
}

func SetOpenAuthenticatedURLFn(fn func(msg, path string)) {
	openAuthenticatedURL = fn
}

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

	var auth shared.ClientAuth
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

		org, err := resolveOrgAuth(orgs, Current.IsLocalMode)

		if err != nil {
			term.OutputErrorAndExit("Error resolving org: %v", err)
		}

		if org.Id == "" {
			// still no org--exit now
			term.OutputErrorAndExit("No org")
		}

		Current.OrgId = org.Id
		Current.OrgName = org.Name
		Current.IntegratedModelsMode = org.IntegratedModelsMode

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
	res, err := verifyEmail(Current.Email, Current.Host)

	if err != nil {
		return fmt.Errorf("error verifying email: %v", err)
	}

	if res.hasAccount {
		return signIn(Current.Email, res.pin, Current.Host)
	} else {
		host := Current.Host
		if host == "" {
			host = "Plandex Cloud"
		}

		term.OutputErrorAndExit("Account %s not found on %s", Current.Email, host)
	}

	return nil
}

func RefreshAuth() error {
	if Current == nil {
		return fmt.Errorf("error refreshing auth: auth not loaded")
	}

	org, apiErr := apiClient.GetOrgSession()

	if apiErr != nil {
		return fmt.Errorf("error getting org session: %v", apiErr.Msg)
	}

	Current.OrgName = org.Name
	Current.OrgIsTrial = org.IsTrial
	Current.IntegratedModelsMode = org.IntegratedModelsMode

	err := writeCurrentAuth()

	if err != nil {
		return fmt.Errorf("error writing auth: %v", err)
	}

	return nil
}
