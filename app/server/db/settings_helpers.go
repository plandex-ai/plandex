package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/plandex/plandex/shared"
)

func GetPlanSettings(plan *Plan, fillDefaultModelSet bool) (*shared.PlanSettings, error) {
	planDir := getPlanDir(plan.OrgId, plan.Id)
	settingsPath := filepath.Join(planDir, "settings.json")

	var settings *shared.PlanSettings

	bytes, err := os.ReadFile(settingsPath)

	if os.IsNotExist(err) || len(bytes) == 0 {
		// if it doesn't exist, return default settings object
		settings = &shared.PlanSettings{
			UpdatedAt: plan.CreatedAt,
		}
		if settings.ModelSet == nil && fillDefaultModelSet {
			settings.ModelSet = &shared.DefaultModelSet
		}
		return settings, nil
	} else if err != nil {
		return nil, fmt.Errorf("error reading settings file: %v", err)
	}

	err = json.Unmarshal(bytes, &settings)

	if err != nil {
		return nil, fmt.Errorf("error unmarshalling settings: %v", err)
	}

	if settings.ModelSet == nil && fillDefaultModelSet {
		settings.ModelSet = &shared.DefaultModelSet
	}

	return settings, nil
}

func StorePlanSettings(plan *Plan, settings *shared.PlanSettings) error {
	planDir := getPlanDir(plan.OrgId, plan.Id)
	settingsPath := filepath.Join(planDir, "settings.json")

	bytes, err := json.Marshal(settings)

	if err != nil {
		return fmt.Errorf("error marshalling settings: %v", err)
	}

	settings.UpdatedAt = time.Now()

	err = os.WriteFile(settingsPath, bytes, 0644)

	if err != nil {
		return fmt.Errorf("error writing settings file: %v", err)
	}

	err = BumpPlanUpdatedAt(plan.Id, settings.UpdatedAt)

	if err != nil {
		return fmt.Errorf("error bumping plan updated at: %v", err)
	}

	return nil
}
