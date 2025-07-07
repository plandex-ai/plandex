package lib

import (
	"plandex-cli/api"
	"plandex-cli/term"

	shared "plandex-shared"
)

var cachedOrgUserConfig *shared.OrgUserConfig

func MustGetOrgUserConfig() *shared.OrgUserConfig {
	if cachedOrgUserConfig != nil {
		return cachedOrgUserConfig
	}

	orgUserConfig, err := api.Client.GetOrgUserConfig()
	if err != nil {
		term.OutputErrorAndExit("Error getting org user config: %v", err)
	}
	cachedOrgUserConfig = orgUserConfig
	return orgUserConfig
}

func MustUpdateOrgUserConfig(orgUserConfig shared.OrgUserConfig) {
	err := api.Client.UpdateOrgUserConfig(orgUserConfig)
	if err != nil {
		term.OutputErrorAndExit("Error updating org user config: %v", err)
	}
	SetCachedOrgUserConfig(&orgUserConfig)
}

func SetCachedOrgUserConfig(orgUserConfig *shared.OrgUserConfig) {
	cachedOrgUserConfig = orgUserConfig
}
