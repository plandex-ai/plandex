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

func GetPlanSettings(plan *Plan) (settings *shared.PlanSettings, err error) {
	planDir := getPlanDir(plan.OrgId, plan.Id)
	settingsPath := filepath.Join(planDir, "settings.json")

	result, err := GetApiCustomModels(plan.OrgId)
	if err != nil {
		return nil, fmt.Errorf("error getting custom models: %v", err)
	}

	defer func() {
		if settings != nil {
			settings.Configure(result.CustomModelPacks, result.CustomModels, result.CustomProviders, os.Getenv("PLANDEX_CLOUD") != "")
		}
	}()

	bytes, err := os.ReadFile(settingsPath)

	if os.IsNotExist(err) || len(bytes) == 0 {
		log.Printf("GetPlanSettings - no settings file found for plan %s - checking org defaults", plan.Id)
		// see if org has default settings
		defaultSettings, err := GetOrgDefaultSettings(plan.OrgId)

		if err != nil {
			return nil, fmt.Errorf("error getting org default settings: %v", err)
		}

		if defaultSettings != nil {
			log.Printf("GetPlanSettings - found org default settings for plan %s", plan.Id)
			return defaultSettings, nil
		} else {
			log.Printf("GetPlanSettings - no org default settings found for plan %s", plan.Id)
		}

		log.Println("GetPlanSettings - no default settings found, returning default settings object")
		// if it doesn't exist, return default settings object
		settings = &shared.PlanSettings{
			UpdatedAt:     plan.CreatedAt,
			ModelPackName: shared.DefaultModelPack.Name,
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

	return settings, nil
}

func StorePlanSettings(plan *Plan, settings shared.PlanSettings) error {
	planDir := getPlanDir(plan.OrgId, plan.Id)
	settingsPath := filepath.Join(planDir, "settings.json")

	settings.UpdatedAt = time.Now()
	settings.CustomModelPacks = nil
	settings.CustomModels = nil
	settings.CustomModelsById = nil
	settings.CustomProviders = nil
	settings.UsesCustomProviderByModelId = nil
	settings.IsCloud = false
	settings.Configured = false

	bytes, err := json.Marshal(settings)

	if err != nil {
		return fmt.Errorf("error marshalling settings: %v", err)
	}

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

func GetOrgDefaultSettings(orgId string) (settings *shared.PlanSettings, err error) {
	result, err := GetApiCustomModels(orgId)
	if err != nil {
		return nil, fmt.Errorf("error getting custom models: %v", err)
	}

	defer func() {
		if settings != nil {
			settings.Configure(result.CustomModelPacks, result.CustomModels, result.CustomProviders, os.Getenv("PLANDEX_CLOUD") != "")
		}
	}()

	query := "SELECT * FROM default_plan_settings WHERE org_id = $1"

	var defaults DefaultPlanSettings

	err = Conn.Get(&defaults, query, orgId)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("GetOrgDefaultSettings - no rows - returning default settings")
			// if it doesn't exist, return default settings object
			settings := &shared.PlanSettings{
				UpdatedAt:     time.Time{},
				ModelPackName: shared.DefaultModelPack.Name,
			}
			return settings, nil
		}
		return nil, fmt.Errorf("error getting default plan settings: %v", err)
	}

	return &defaults.PlanSettings, nil
}

func GetOrgDefaultSettingsForUpdate(orgId string, tx *sqlx.Tx) (settings *shared.PlanSettings, err error) {
	result, err := GetApiCustomModels(orgId)
	if err != nil {
		return nil, fmt.Errorf("error getting custom models: %v", err)
	}

	defer func() {
		if settings != nil {
			settings.Configure(result.CustomModelPacks, result.CustomModels, result.CustomProviders, os.Getenv("PLANDEX_CLOUD") != "")
		}
	}()

	query := "SELECT * FROM default_plan_settings WHERE org_id = $1 FOR UPDATE"

	var defaults DefaultPlanSettings

	err = tx.Get(&defaults, query, orgId)

	if err != nil {
		if err == sql.ErrNoRows {
			// if it doesn't exist, return default settings object
			settings := &shared.PlanSettings{
				UpdatedAt:     time.Time{},
				ModelPackName: shared.DefaultModelPack.Name,
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

type GetCustomModelsResult struct {
	CustomModels     []*shared.CustomModel
	CustomProviders  []*shared.CustomProvider
	CustomModelPacks []*shared.ModelPack
}

func GetApiCustomModels(orgId string) (result *GetCustomModelsResult, err error) {
	var customModels []*CustomModel
	var customProviders []*CustomProvider
	var customModelPacks []*ModelPack

	errCh := make(chan error, 3)

	go func() {
		res, err := ListModelPacks(orgId)
		if err != nil {
			errCh <- fmt.Errorf("error getting custom model packs: %v", err)
		}
		customModelPacks = res
		errCh <- nil
	}()

	go func() {
		res, err := ListCustomModels(orgId)
		if err != nil {
			errCh <- fmt.Errorf("error getting custom models: %v", err)
		}
		customModels = res
		errCh <- nil
	}()

	go func() {
		res, err := ListCustomProviders(orgId)
		if err != nil {
			errCh <- fmt.Errorf("error getting custom providers: %v", err)
		}
		customProviders = res
		errCh <- nil
	}()

	for i := 0; i < 3; i++ {
		err := <-errCh
		if err != nil {
			return nil, err
		}
	}

	apiModelPacks := make([]*shared.ModelPack, len(customModelPacks))
	for i, modelPack := range customModelPacks {
		apiModelPacks[i] = modelPack.ToApi()
	}

	apiCustomModels := make([]*shared.CustomModel, len(customModels))
	for i, model := range customModels {
		apiCustomModels[i] = model.ToApi()
	}

	apiCustomProviders := make([]*shared.CustomProvider, len(customProviders))
	for i, provider := range customProviders {
		apiCustomProviders[i] = provider.ToApi()
	}

	result = &GetCustomModelsResult{
		CustomModels:     apiCustomModels,
		CustomProviders:  apiCustomProviders,
		CustomModelPacks: apiModelPacks,
	}

	return result, nil
}
