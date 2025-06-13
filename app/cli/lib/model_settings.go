package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex-cli/api"
	"plandex-cli/fs"
	"plandex-cli/schema"
	"plandex-cli/term"
	shared "plandex-shared"

	"github.com/fatih/color"
)

var DefaultModelSettingsPath string

func init() {
	DefaultModelSettingsPath = filepath.Join(fs.HomePlandexDir, "default-model-settings.json")
}

func GetPlanModelSettingsPath(planId string) string {
	return filepath.Join(fs.HomePlandexDir, planId, "model-settings.json")
}

type ModelSettingsCheckLocalChangesResult struct {
	HasLocalChanges           bool
	LocalModelPackSchemaRoles *shared.ModelPackSchemaRoles
}

func ModelSettingsCheckLocalChanges(path string) (ModelSettingsCheckLocalChangesResult, error) {
	hashPath := path + ".hash"

	exists, err := fs.FileExists(path)
	if err != nil {
		return ModelSettingsCheckLocalChangesResult{}, fmt.Errorf("error checking model settings file: %v", err)
	}

	if !exists {
		return ModelSettingsCheckLocalChangesResult{}, nil
	}

	lastSavedHash, err := os.ReadFile(hashPath)
	if err != nil && !os.IsNotExist(err) {
		return ModelSettingsCheckLocalChangesResult{}, fmt.Errorf("error reading hash file: %v", err)
	}

	localJsonData, err := os.ReadFile(path)
	if err != nil {
		return ModelSettingsCheckLocalChangesResult{}, fmt.Errorf("error reading JSON file: %v", err)
	}

	var clientModelPackSchemaRoles *shared.ClientModelPackSchemaRoles
	err = json.Unmarshal(localJsonData, &clientModelPackSchemaRoles)
	if err != nil {
		return ModelSettingsCheckLocalChangesResult{}, fmt.Errorf("error unmarshalling JSON file: %v", err)
	}

	currentHash, err := clientModelPackSchemaRoles.ToModelPackSchemaRoles().Hash()
	if err != nil {
		return ModelSettingsCheckLocalChangesResult{}, fmt.Errorf("error hashing model pack: %v", err)
	}

	modelPackSchemaRoles := clientModelPackSchemaRoles.ToModelPackSchemaRoles()

	return ModelSettingsCheckLocalChangesResult{
		HasLocalChanges:           currentHash != string(lastSavedHash),
		LocalModelPackSchemaRoles: &modelPackSchemaRoles,
	}, nil
}

func WriteModelSettingsFile(path string, originalSettings *shared.PlanSettings) error {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	clientModelPackRoles := originalSettings.GetModelPack().ToModelPackSchema().ToClientModelPackSchemaRoles()

	bytes, err := json.MarshalIndent(clientModelPackRoles, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling model pack: %v", err)
	}

	err = os.WriteFile(path, bytes, 0644)
	if err != nil {
		return fmt.Errorf("error writing JSON file: %v", err)
	}

	return nil
}

func SaveModelPackRolesHash(basePath string, serverModelPack *shared.ModelPackSchemaRoles) error {
	hashPath := basePath + ".hash"

	hash, err := serverModelPack.Hash()
	if err != nil {
		return fmt.Errorf("error hashing model pack: %v", err)
	}

	err = os.WriteFile(hashPath, []byte(hash), 0644)
	if err != nil {
		return fmt.Errorf("error writing hash file: %v", err)
	}

	return nil
}

func ApplyModelSettings(path string, originalSettings *shared.PlanSettings) (*shared.PlanSettings, error) {
	jsonData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading JSON file: %v", err)
	}

	settings, err := originalSettings.DeepCopy()
	if err != nil {
		return nil, fmt.Errorf("error copying settings: %v", err)
	}

	clientModelPackRoles, err := schema.ValidateModelPackInlineJSON(jsonData)
	if err != nil {
		term.StopSpinner()
		color.New(color.Bold, term.ColorHiRed).Println("🚨 Error validating JSON file")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	modelPackRoles := clientModelPackRoles.ToModelPackSchemaRoles()

	modelPackSchema := shared.ModelPackSchema{
		Name:                 "custom",
		Description:          "Model pack with custom settings",
		ModelPackSchemaRoles: modelPackRoles,
	}
	modelPack := modelPackSchema.ToModelPack()
	settings.SetCustomModelPack(&modelPack)

	err = SaveModelPackRolesHash(path, &modelPackRoles)
	if err != nil {
		return nil, fmt.Errorf("error saving model settings hash: %v", err)
	}

	return settings, nil
}

func SaveLatestPlanModelSettingsIfNeeded() (bool, error) {
	path := GetPlanModelSettingsPath(CurrentPlanId)
	jsonData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("error reading JSON file: %v", err)
	}

	var clientModelPackSchemaRoles *shared.ClientModelPackSchemaRoles
	err = json.Unmarshal(jsonData, &clientModelPackSchemaRoles)
	if err != nil {
		return false, fmt.Errorf("error unmarshalling JSON file: %v", err)
	}

	modelPackSchemaRoles := clientModelPackSchemaRoles.ToModelPackSchemaRoles()

	localHash, err := modelPackSchemaRoles.Hash()
	if err != nil {
		return false, fmt.Errorf("error hashing model pack: %v", err)
	}

	settings, apiErr := api.Client.GetSettings(CurrentPlanId, CurrentBranch)
	if apiErr != nil {
		return false, fmt.Errorf("error getting settings: %v", apiErr)
	}

	serverHash, err := settings.GetModelPack().ToModelPackSchema().ModelPackSchemaRoles.Hash()
	if err != nil {
		return false, fmt.Errorf("error hashing model pack: %v", err)
	}

	if localHash == serverHash {
		return false, nil
	}

	err = WriteModelSettingsFile(path, settings)
	if err != nil {
		return false, fmt.Errorf("error writing model settings file: %v", err)
	}

	return true, nil
}

// save settings in file to server
func SyncPlanModelSettings() error {
	settings, err := api.Client.GetSettings(CurrentPlanId, CurrentBranch)
	if err != nil {
		return fmt.Errorf("error getting settings: %v", err)
	}

	updatedSettings, apiErr := ApplyModelSettings(GetPlanModelSettingsPath(CurrentPlanId), settings)
	if apiErr != nil {
		return fmt.Errorf("error applying model settings: %v", err)
	}

	res, updateErr := api.Client.UpdateSettings(CurrentPlanId, CurrentBranch, shared.UpdateSettingsRequest{
		ModelPackName: updatedSettings.ModelPackName,
		ModelPack:     updatedSettings.ModelPack,
	})

	if updateErr != nil {
		return fmt.Errorf("error updating settings: %v", err)
	}

	if res == nil {
		return nil
	}

	fmt.Println(res.Msg)

	return nil
}

func SyncDefaultModelSettings() error {
	settings, err := api.Client.GetOrgDefaultSettings()
	if err != nil {
		return fmt.Errorf("error getting settings: %v", err)
	}

	updatedSettings, apiErr := ApplyModelSettings(DefaultModelSettingsPath, settings)
	if apiErr != nil {
		return fmt.Errorf("error applying model settings: %v", err)
	}

	res, updateErr := api.Client.UpdateOrgDefaultSettings(shared.UpdateSettingsRequest{
		ModelPackName: updatedSettings.ModelPackName,
		ModelPack:     updatedSettings.ModelPack,
	})

	if updateErr != nil {
		return fmt.Errorf("error updating settings: %v", err)
	}

	if res == nil {
		return nil
	}

	fmt.Println(res.Msg)

	return nil
}
