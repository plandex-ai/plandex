package lib

import (
	"os"
	"plandex-cli/api"
	"plandex-cli/term"
	"sort"

	shared "plandex-shared"

	"github.com/olekukonko/tablewriter"
)

var cachedPlanConfig *shared.PlanConfig

func MustGetCurrentPlanConfig() *shared.PlanConfig {
	if cachedPlanConfig != nil {
		return cachedPlanConfig
	}

	planConfig, apiErr := api.Client.GetPlanConfig(CurrentPlanId)
	if apiErr != nil {
		term.OutputErrorAndExit("Error getting plan config: %v", apiErr)
	}
	cachedPlanConfig = planConfig
	return planConfig
}

func SetCachedPlanConfig(planConfig *shared.PlanConfig) {
	cachedPlanConfig = planConfig
}

func ShowPlanConfig(config *shared.PlanConfig, key string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(true)
	table.SetHeader([]string{"Name", "Value", "Description"})

	numVisibleSettings := 0
	for k, setting := range shared.ConfigSettingsByKey {
		if key != "" && k != key {
			continue
		}

		if setting.Visible == nil || setting.Visible(config) {
			numVisibleSettings++
		}
	}
	numOutput := 0

	sortedSettings := make([][]string, 0, len(shared.ConfigSettingsByKey))

	for k, setting := range shared.ConfigSettingsByKey {
		if key != "" && k != key {
			continue
		}

		if setting.Visible == nil || setting.Visible(config) {
			var sortKey string
			if setting.SortKey != "" {
				sortKey = setting.SortKey
			} else {
				sortKey = k
			}
			sortedSettings = append(sortedSettings, []string{sortKey, setting.Name, setting.Getter(config), setting.Desc})
		}
	}

	sort.Slice(sortedSettings, func(i, j int) bool {
		return sortedSettings[i][0] < sortedSettings[j][0]
	})

	for _, row := range sortedSettings {
		table.Append(row[1:])
		numOutput++
		if numOutput < numVisibleSettings {
			table.Append([]string{"", "", ""})
		}
	}

	table.Render()
}
