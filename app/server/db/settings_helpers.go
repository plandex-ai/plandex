package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	shared "plandex-shared"

	"github.com/jmoiron/sqlx"
)

func GetPlanSettings(plan *Plan, fillDefaultModelPack bool) (*shared.PlanSettings, error) {
	planDir := getPlanDir(plan.OrgId, plan.Id)
	settingsPath := filepath.Join(planDir, "settings.json")

	var settings *shared.PlanSettings

	bytes, err := os.ReadFile(settingsPath)

	if os.IsNotExist(err) || len(bytes) == 0 {
		log.Printf("GetPlanSettings - no settings file found for plan %s - checking org defaults", plan.Id)
		// see if org has default settings
		defaultSettings, err := GetOrgDefaultSettings(plan.OrgId, fillDefaultModelPack)

		if err != nil {
			return nil, fmt.Errorf("error getting org default settings: %v", err)
		}

		if defaultSettings != nil {
			log.Printf("GetPlanSettings - found org default settings for plan %s", plan.Id)
			return defaultSettings, nil
		} else {
			log.Printf("GetPlanSettings - no org default settings found for plan %s", plan.Id)
		}

		// if it doesn't exist, return default settings object
		settings = &shared.PlanSettings{
			UpdatedAt: plan.CreatedAt,
		}
		if settings.ModelPack == nil && fillDefaultModelPack {
			settings.ModelPack = shared.DefaultModelPack
		}
		return settings, nil
	} else if err != nil {
		return nil, fmt.Errorf("error reading settings file: %v", err)
	}

	log.Printf("GetPlanSettings - settings found in file")

	err = json.Unmarshal(bytes, &settings)

	if err != nil {
		return nil, fmt.Errorf("error unmarshalling settings: %v", err)
	}

	if settings.ModelPack == nil && fillDefaultModelPack {
		log.Printf("GetPlanSettings - filling default model pack")
		settings.ModelPack = shared.DefaultModelPack
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

func GetOrgDefaultSettings(orgId string, fillDefaultModelPack bool) (*shared.PlanSettings, error) {
	query := "SELECT * FROM default_plan_settings WHERE org_id = $1"

	var defaults DefaultPlanSettings

	err := Conn.Get(&defaults, query, orgId)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("GetOrgDefaultSettings - no rows - returning default settings")
			// if it doesn't exist, return default settings object
			settings := &shared.PlanSettings{
				UpdatedAt: time.Time{},
			}
			if settings.ModelPack == nil && fillDefaultModelPack {
				settings.ModelPack = shared.DefaultModelPack
			}
			return settings, nil
		}
		return nil, fmt.Errorf("error getting default plan settings: %v", err)
	}

	// fill in default WholeFileBuilder if not set
	if defaults.PlanSettings.ModelPack.WholeFileBuilder == nil {
		defaults.PlanSettings.ModelPack.WholeFileBuilder = shared.DefaultModelPack.WholeFileBuilder
	}

	return &defaults.PlanSettings, nil
}

func GetOrgDefaultSettingsForUpdate(orgId string, tx *sqlx.Tx, fillDefaultModelPack bool) (*shared.PlanSettings, error) {
	query := "SELECT * FROM default_plan_settings WHERE org_id = $1 FOR UPDATE"

	var defaults DefaultPlanSettings

	err := tx.Get(&defaults, query, orgId)

	if err != nil {
		if err == sql.ErrNoRows {
			// if it doesn't exist, return default settings object
			settings := &shared.PlanSettings{
				UpdatedAt: time.Time{},
			}
			if settings.ModelPack == nil && fillDefaultModelPack {
				settings.ModelPack = shared.DefaultModelPack
			}
			return settings, nil
		}
		return nil, fmt.Errorf("error getting default plan settings: %v", err)
	}

	return &defaults.PlanSettings, nil
}

func StoreOrgDefaultSettings(orgId string, settings *shared.PlanSettings, tx *sqlx.Tx) error {
	settings.UpdatedAt = time.Now()

	query := `INSERT INTO default_plan_settings (org_id, plan_settings) 
	VALUES ($1, $2) 
	ON CONFLICT (org_id) DO UPDATE SET plan_settings = excluded.plan_settings
	`

	_, err := tx.Exec(query, orgId, settings)

	if err != nil {
		return fmt.Errorf("error storing default plan settings: %v", err)
	}

	return nil
}
