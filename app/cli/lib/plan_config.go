package lib

import (
	"os"
	"sort"

	"github.com/olekukonko/tablewriter"
	"github.com/plandex/plandex/shared"
)

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
