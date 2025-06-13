package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/fs"
	"plandex-cli/schema"
	"plandex-cli/term"
	shared "plandex-shared"
	"strings"

	"github.com/fatih/color"
)

var CustomModelsDefaultPath string

type CustomModelsCheckLocalChangesResult struct {
	HasLocalChanges  bool
	LocalModelsInput shared.ModelsInput
}

func init() {
	CustomModelsDefaultPath = filepath.Join(fs.HomePlandexDir, "custom-models.json")
}

func GetServerModelsInput() (*shared.ModelsInput, error) {
	errCh := make(chan *shared.ApiError, 3)
	var (
		customModels     []*shared.CustomModel
		customProviders  []*shared.CustomProvider
		customModelPacks []*shared.ModelPackSchema
	)

	go func() {
		models, apiErr := api.Client.ListCustomModels()
		if apiErr != nil {
			errCh <- apiErr
			return
		}
		customModels = models
		errCh <- nil
	}()

	go func() {
		// custom providers are not supported on cloud
		if auth.Current.IsCloud {
			errCh <- nil
			return
		}
		providers, apiErr := api.Client.ListCustomProviders()
		if apiErr != nil {
			errCh <- apiErr
			return
		}
		customProviders = providers
		errCh <- nil
	}()

	go func() {
		modelPacks, apiErr := api.Client.ListModelPacks()
		if apiErr != nil {
			errCh <- apiErr
			return
		}

		schemas := make([]*shared.ModelPackSchema, len(modelPacks))
		for i, modelPack := range modelPacks {
			schemas[i] = modelPack.ToModelPackSchema()
		}

		customModelPacks = schemas
		errCh <- nil
	}()

	for i := 0; i < 3; i++ {
		err := <-errCh
		if err != nil {
			return nil, fmt.Errorf("error fetching custom models: %v", err.Msg)
		}
	}

	serverModelsInput := &shared.ModelsInput{
		CustomModels:     customModels,
		CustomProviders:  customProviders,
		CustomModelPacks: customModelPacks,
	}

	return serverModelsInput, nil
}

func CustomModelsCheckLocalChanges(path string) (CustomModelsCheckLocalChangesResult, error) {
	hashPath := path + ".hash"

	exists, err := fs.FileExists(path)
	if err != nil {
		return CustomModelsCheckLocalChangesResult{}, err
	}

	if !exists {
		return CustomModelsCheckLocalChangesResult{}, nil
	}

	localJsonData, err := os.ReadFile(path)
	if err != nil {
		return CustomModelsCheckLocalChangesResult{}, fmt.Errorf("error reading JSON file: %v", err)
	}

	var localClientModelsInput shared.ClientModelsInput
	err = json.Unmarshal(localJsonData, &localClientModelsInput)
	if err != nil {
		return CustomModelsCheckLocalChangesResult{}, fmt.Errorf("error unmarshalling JSON file: %v", err)
	}

	localModelsInput := localClientModelsInput.ToModelsInput()

	lastSavedHash, err := os.ReadFile(hashPath)
	if err != nil && !os.IsNotExist(err) {
		return CustomModelsCheckLocalChangesResult{}, fmt.Errorf("error reading hash file: %v", err)
	}

	currentHash, err := localModelsInput.Hash()
	if err != nil {
		return CustomModelsCheckLocalChangesResult{}, fmt.Errorf("error hashing models: %v", err)
	}

	return CustomModelsCheckLocalChangesResult{
		HasLocalChanges:  currentHash != string(lastSavedHash),
		LocalModelsInput: localModelsInput,
	}, nil
}

func WriteCustomModelsFile(path string, modelsInput *shared.ModelsInput, saveHash bool) error {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	clientModelsInput := modelsInput.ToClientModelsInput()
	clientModelsInput.PrepareUpdate()

	jsonData, err := json.MarshalIndent(clientModelsInput, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling models: %v", err)
	}

	err = os.WriteFile(path, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	if saveHash {
		err = SaveCustomModelsHash(path, modelsInput)
		if err != nil {
			return fmt.Errorf("error saving hash file: %v", err)
		}
	}

	return nil
}

func SaveCustomModelsHash(basePath string, modelsInput *shared.ModelsInput) error {
	hashPath := basePath + ".hash"

	hash, err := modelsInput.Hash()
	if err != nil {
		return fmt.Errorf("error hashing models: %v", err)
	}

	err = os.WriteFile(hashPath, []byte(hash), 0644)
	if err != nil {
		return fmt.Errorf("error writing hash file: %v", err)
	}

	return nil
}

func MustSyncCustomModels(path string, serverModelsInput *shared.ModelsInput, saveHash bool) bool {
	term.StartSpinner("")

	jsonData, err := os.ReadFile(path)
	if err != nil {
		term.OutputErrorAndExit("Error reading custom models file: %v", err)
		return false
	}

	clientModelsInput, err := schema.ValidateModelsInputJSON(jsonData)
	if err != nil {
		term.StopSpinner()
		color.New(color.Bold, term.ColorHiRed).Println("🚨 Error validating custom models file")
		fmt.Println(err.Error())
		return false
	}

	modelsInput := clientModelsInput.ToModelsInput()

	noDuplicates, errMsg := modelsInput.CheckNoDuplicates()
	if !noDuplicates {
		term.StopSpinner()
		color.New(color.Bold, term.ColorHiRed).Println("🚨 Some items in custom models file are duplicated:")
		fmt.Println()
		fmt.Println(errMsg)
		return false
	}

	if modelsInput.Equals(*serverModelsInput) {
		term.StopSpinner()
		return false
	}

	apiErr := api.Client.CreateCustomModels(&modelsInput)
	if apiErr != nil {
		term.OutputErrorAndExit("Error importing models: %v", apiErr.Msg)
		return false
	}

	if saveHash {
		err := SaveCustomModelsHash(path, &modelsInput)
		if err != nil {
			term.OutputErrorAndExit("Error saving hash file: %v", err)
			return false
		}
	}

	inputModelIds := map[string]bool{}
	inputProviderNames := map[string]bool{}
	inputModelPackNames := map[string]bool{}
	for _, model := range clientModelsInput.CustomModels {
		inputModelIds[string(model.ModelId)] = true
	}
	for _, provider := range clientModelsInput.CustomProviders {
		inputProviderNames[provider.Name] = true
	}
	for _, modelPack := range clientModelsInput.CustomModelPacks {
		inputModelPackNames[modelPack.Name] = true
	}

	updatedModelsInput := modelsInput.FilterUnchanged(serverModelsInput)

	customModels := serverModelsInput.CustomModels
	customProviders := serverModelsInput.CustomProviders
	customModelPacks := serverModelsInput.CustomModelPacks

	term.StopSpinner()

	added := strings.Builder{}
	updated := strings.Builder{}
	deleted := strings.Builder{}

	existsById := map[string]bool{}
	for _, model := range customModels {
		existsById[string(model.ModelId)] = true
	}
	for _, provider := range customProviders {
		existsById[provider.Name] = true
	}
	for _, modelPack := range customModelPacks {
		existsById[modelPack.Name] = true
	}

	for _, provider := range updatedModelsInput.CustomProviders {
		action := "✅ Added"
		builder := &added
		if existsById[provider.Name] {
			action = "🔄 Updated"
			builder = &updated
		}
		builder.WriteString(fmt.Sprintf("%s custom %s → %s\n",
			action,
			color.New(term.ColorHiCyan).Sprint("provider"),
			color.New(color.Bold, term.ColorHiGreen).Sprint(provider.Name)))
	}
	for _, provider := range customProviders {
		if !inputProviderNames[provider.Name] {
			deleted.WriteString(fmt.Sprintf("❌ Removed custom %s → %s\n",
				color.New(term.ColorHiCyan).Sprint("provider"),
				color.New(color.Bold, term.ColorHiRed).Sprint(provider.Name)))
		}
	}

	for _, model := range updatedModelsInput.CustomModels {
		action := "✅ Added"
		builder := &added
		if existsById[string(model.ModelId)] {
			action = "🔄 Updated"
			builder = &updated
		}
		builder.WriteString(fmt.Sprintf("%s custom %s → %s\n",
			action,
			color.New(term.ColorHiCyan).Sprint("model"),
			color.New(color.Bold, term.ColorHiGreen).Sprint(string(model.ModelId))))
	}
	for _, model := range customModels {
		if !inputModelIds[string(model.ModelId)] {
			deleted.WriteString(fmt.Sprintf("❌ Removed custom %s → %s\n",
				color.New(term.ColorHiCyan).Sprint("model"),
				color.New(color.Bold, term.ColorHiRed).Sprint(string(model.ModelId))))
		}
	}

	for _, modelPack := range updatedModelsInput.CustomModelPacks {
		action := "✅ Added"
		builder := &added
		if existsById[modelPack.Name] {
			action = "🔄 Updated"
			builder = &updated
		}
		builder.WriteString(fmt.Sprintf("%s custom %s → %s\n",
			action,
			color.New(term.ColorHiCyan).Sprint("model pack"),
			color.New(color.Bold, term.ColorHiGreen).Sprint(modelPack.Name)))
	}
	for _, modelPack := range customModelPacks {
		if !inputModelPackNames[modelPack.Name] {
			deleted.WriteString(fmt.Sprintf("❌ Removed custom %s → %s\n",
				color.New(term.ColorHiCyan).Sprint("model pack"),
				color.New(color.Bold, term.ColorHiRed).Sprint(modelPack.Name)))
		}
	}

	if updated.Len()+added.Len()+deleted.Len() == 0 {
		return false
	}

	fmt.Print(added.String())
	fmt.Print(updated.String())
	fmt.Print(deleted.String())

	return true
}

func SyncCustomModels() error {
	serverModelsInput, err := GetServerModelsInput()
	if err != nil {
		return fmt.Errorf("error getting server models input: %v", err)
	}

	MustSyncCustomModels(CustomModelsDefaultPath, serverModelsInput, true)

	return nil
}
