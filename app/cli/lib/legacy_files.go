package lib

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"plandex-cli/fs"
	"plandex-cli/term"
	"plandex-cli/types"
)

func MigrateLegacyProjectFile(currentUserId string) {
	if fs.PlandexDir == "" {
		return
	}

	if currentUserId == "" {
		return
	}

	// Migrate project.json to projects-v2.json
	// New formats map to a user id so we can handle multiple accounts in the same plandex dir
	projectPath := filepath.Join(fs.PlandexDir, "project.json")
	if _, err := os.Stat(projectPath); err == nil {
		var settings types.CurrentProjectSettings
		bytes, err := os.ReadFile(projectPath)
		if err != nil {
			term.OutputErrorAndExit("error reading project.json: %v", err)
		}

		err = json.Unmarshal(bytes, &settings)
		if err != nil {
			term.OutputErrorAndExit("error unmarshalling project.json: %v", err)
		}

		v2Path := filepath.Join(fs.PlandexDir, "projects-v2.json")
		settingsByAccount := types.CurrentProjectSettingsByAccount{
			currentUserId: &settings,
		}
		bytes, err = json.Marshal(settingsByAccount)
		if err != nil {
			term.OutputErrorAndExit("error marshalling projects-v2.json: %v", err)
		}

		err = os.WriteFile(v2Path, bytes, 0644)
		if err != nil {
			term.OutputErrorAndExit("error writing projects-v2.json: %v", err)
		}

		// Delete the v1 file after successful migration
		if err := os.Remove(projectPath); err != nil {
			term.OutputErrorAndExit("could not delete old project.json: %v", err)
		}
		log.Println("Migrated project.json to projects-v2.json")

	} else if !os.IsNotExist(err) {
		term.OutputErrorAndExit("error checking for project.json: %v", err)
	}
}

func MigrateLegacyCurrentPlanFile(currentUserId string) {
	if fs.PlandexDir == "" {
		return
	}

	if currentUserId == "" {
		return
	}

	if CurrentProjectId == "" {
		return
	}

	// Migrate current_plan.json to current-plans-v2.json
	planPath := filepath.Join(fs.HomePlandexDir, CurrentProjectId, "current_plan.json")

	if _, err := os.Stat(planPath); err == nil {
		var settings types.CurrentPlanSettings
		bytes, err := os.ReadFile(planPath)
		if err != nil {
			term.OutputErrorAndExit("error reading current_plan.json: %v", err)
		}

		err = json.Unmarshal(bytes, &settings)
		if err != nil {
			term.OutputErrorAndExit("error unmarshalling current_plan.json: %v", err)
		}

		v2Path := filepath.Join(fs.HomePlandexDir, CurrentProjectId, "current-plans-v2.json")
		settingsByAccount := types.CurrentPlanSettingsByAccount{
			currentUserId: &settings,
		}
		bytes, err = json.Marshal(settingsByAccount)
		if err != nil {
			term.OutputErrorAndExit("error marshalling current-plans-v2.json: %v", err)
		}

		err = os.WriteFile(v2Path, bytes, 0644)
		if err != nil {
			term.OutputErrorAndExit("error writing current-plans-v2.json: %v", err)
		}

		// Delete the v1 file after successful migration
		if err := os.Remove(planPath); err != nil {
			term.OutputErrorAndExit("could not delete old current_plan.json: %v", err)
		}
		log.Println("Migrated current_plan.json to current-plans-v2.json")
	} else if !os.IsNotExist(err) {
		term.OutputErrorAndExit("error checking for current_plan.json: %v", err)
	}
}

func MigrateLegacyPlanSettingsFile(currentUserId string) {
	if fs.PlandexDir == "" {
		return
	}

	if currentUserId == "" {
		return
	}

	if CurrentPlanId == "" {
		return
	}

	// Migrate settings.json to settings-v2.json for current plan
	settingsPath := filepath.Join(fs.HomePlandexDir, CurrentProjectId, CurrentPlanId, "settings.json")
	if _, err := os.Stat(settingsPath); err == nil {
		var settings types.PlanSettings
		bytes, err := os.ReadFile(settingsPath)
		if err != nil {
			term.OutputErrorAndExit("error reading settings.json: %v", err)
		}

		err = json.Unmarshal(bytes, &settings)
		if err != nil {
			term.OutputErrorAndExit("error unmarshalling settings.json: %v", err)
		}

		v2Path := filepath.Join(fs.HomePlandexDir, CurrentProjectId, CurrentPlanId, "settings-v2.json")
		settingsByAccount := types.PlanSettingsByAccount{
			currentUserId: &settings,
		}
		bytes, err = json.Marshal(settingsByAccount)
		if err != nil {
			term.OutputErrorAndExit("error marshalling settings-v2.json: %v", err)
		}

		err = os.WriteFile(v2Path, bytes, 0644)
		if err != nil {
			term.OutputErrorAndExit("error writing settings-v2.json: %v", err)
		}

		// Delete the v1 file after successful migration
		if err := os.Remove(settingsPath); err != nil {
			term.OutputErrorAndExit("could not delete old settings.json: %v", err)
		}
		log.Println("Migrated settings.json to settings-v2.json")
	} else if !os.IsNotExist(err) {
		term.OutputErrorAndExit("error checking for settings.json: %v", err)
	}
}
