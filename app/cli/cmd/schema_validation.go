package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	shared "plandex-shared"

	"github.com/xeipuuv/gojsonschema"
)

//go:embed ../json-schemas/*.schema.json ../json-schemas/definitions/*.schema.json
var schemaFS embed.FS

// embeddedSchemaLoader implements a custom loader for embedded JSON schemas
// that can resolve references between schemas
type embeddedSchemaLoader struct {
	source string
	fs     embed.FS
}

// newEmbeddedSchemaLoader creates a new loader for embedded schemas
func newEmbeddedSchemaLoader(source string) embeddedSchemaLoader {
	return embeddedSchemaLoader{
		source: source,
		fs:     schemaFS,
	}
}

// JsonSource returns the source of the schema
func (l *embeddedSchemaLoader) JsonSource() interface{} {
	return l.source
}

// LoadJSON loads the schema from the embedded filesystem
func (l *embeddedSchemaLoader) LoadJSON() (interface{}, error) {
	// If the source is a reference to another schema
	if strings.HasPrefix(l.source, "definitions/") || strings.HasSuffix(l.source, ".schema.json") {
		// Construct the path to the schema in the embedded filesystem
		schemaPath := l.source
		if !strings.HasPrefix(schemaPath, "../json-schemas/") {
			schemaPath = filepath.Join("../json-schemas", schemaPath)
		}

		// Read the schema from the embedded filesystem
		schemaData, err := l.fs.ReadFile(schemaPath)
		if err != nil {
			return nil, fmt.Errorf("error reading embedded schema %s: %v", schemaPath, err)
		}

		// Parse the schema
		var schemaMap map[string]interface{}
		if err := json.Unmarshal(schemaData, &schemaMap); err != nil {
			return nil, fmt.Errorf("error parsing embedded schema %s: %v", schemaPath, err)
		}

		return schemaMap, nil
	}

	// If the source is a JSON string
	var schemaMap map[string]interface{}
	if err := json.Unmarshal([]byte(l.source), &schemaMap); err != nil {
		return nil, fmt.Errorf("error parsing schema JSON: %v", err)
	}

	return schemaMap, nil
}

// JsonReference returns a new loader for the referenced schema
func (l *embeddedSchemaLoader) JsonReference() gojsonschema.JSONLoader {
	return newEmbeddedSchemaLoader(l.source)
}

// ValidateModelConfigJSON validates a JSON file against the model configuration schema
// and returns a CustomModel if validation succeeds.
func ValidateModelConfigJSON(jsonFilePath string) (*shared.CustomModel, error) {
	// Read the JSON file
	jsonData, err := schemaFS.ReadFile(jsonFilePath)
	if err != nil {
		// Fall back to reading from the filesystem if not found in embedded FS
		jsonData, err = schemaFS.ReadFile(jsonFilePath)
		if err != nil {
			return nil, fmt.Errorf("error reading JSON file: %v", err)
		}
	}

	// Create a schema loader for the embedded schema
	schemaLoader := newEmbeddedSchemaLoader("../json-schemas/model-config.schema.json")
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("schema validation error: %v", err)
	}

	// Check validation result
	if !result.Valid() {
		var errorMessages []string
		for _, desc := range result.Errors() {
			errorMessages = append(errorMessages, fmt.Sprintf("- %s", desc))
		}
		return nil, fmt.Errorf("JSON validation failed:\n%s", strings.Join(errorMessages, "\n"))
	}

	// Parse the validated JSON into a CustomModel
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &jsonMap); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	// Create the model
	model := &shared.CustomModel{
		BaseModelShared: shared.BaseModelShared{
			ModelCompatibility: shared.ModelCompatibility{},
		},
	}

	// Extract required fields
	if modelId, ok := jsonMap["modelId"].(string); ok {
		model.ModelId = shared.ModelId(modelId)
	} else {
		return nil, fmt.Errorf("missing or invalid 'modelId' in JSON")
	}

	// Extract optional fields
	if description, ok := jsonMap["description"].(string); ok {
		model.Description = description
	}

	// Extract numeric fields
	if maxTokens, ok := jsonMap["maxTokens"].(float64); ok {
		model.MaxTokens = int(maxTokens)
	} else {
		return nil, fmt.Errorf("missing or invalid 'maxTokens' in JSON")
	}

	if maxConvoTokens, ok := jsonMap["defaultMaxConvoTokens"].(float64); ok {
		model.DefaultMaxConvoTokens = int(maxConvoTokens)
	} else {
		return nil, fmt.Errorf("missing or invalid 'defaultMaxConvoTokens' in JSON")
	}

	if maxOutputTokens, ok := jsonMap["maxOutputTokens"].(float64); ok {
		model.MaxOutputTokens = int(maxOutputTokens)
	} else {
		return nil, fmt.Errorf("missing or invalid 'maxOutputTokens' in JSON")
	}

	if reservedOutputTokens, ok := jsonMap["reservedOutputTokens"].(float64); ok {
		model.ReservedOutputTokens = int(reservedOutputTokens)
	} else {
		return nil, fmt.Errorf("missing or invalid 'reservedOutputTokens' in JSON")
	}

	// Extract output format
	if outputFormat, ok := jsonMap["preferredOutputFormat"].(string); ok {
		model.PreferredOutputFormat = shared.ModelOutputFormat(outputFormat)
	} else {
		return nil, fmt.Errorf("missing or invalid 'preferredOutputFormat' in JSON")
	}

	// Extract boolean fields
	if hasImageSupport, ok := jsonMap["hasImageSupport"].(bool); ok {
		model.HasImageSupport = hasImageSupport
	}

	if systemPromptDisabled, ok := jsonMap["systemPromptDisabled"].(bool); ok {
		model.SystemPromptDisabled = systemPromptDisabled
	}

	if roleParamsDisabled, ok := jsonMap["roleParamsDisabled"].(bool); ok {
		model.RoleParamsDisabled = roleParamsDisabled
	}

	if stopDisabled, ok := jsonMap["stopDisabled"].(bool); ok {
		model.StopDisabled = stopDisabled
	}

	if predictedOutputEnabled, ok := jsonMap["predictedOutputEnabled"].(bool); ok {
		model.PredictedOutputEnabled = predictedOutputEnabled
	}

	if includeReasoning, ok := jsonMap["includeReasoning"].(bool); ok {
		model.IncludeReasoning = includeReasoning
	}

	if reasoningEffortEnabled, ok := jsonMap["reasoningEffortEnabled"].(bool); ok {
		model.ReasoningEffortEnabled = reasoningEffortEnabled
	}

	if reasoningEffort, ok := jsonMap["reasoningEffort"].(string); ok {
		model.ReasoningEffort = reasoningEffort
	}

	if supportsCacheControl, ok := jsonMap["supportsCacheControl"].(bool); ok {
		model.SupportsCacheControl = supportsCacheControl
	}

	if singleMessageNoSystemPrompt, ok := jsonMap["singleMessageNoSystemPrompt"].(bool); ok {
		model.SingleMessageNoSystemPrompt = singleMessageNoSystemPrompt
	}

	if tokenEstimatePaddingPct, ok := jsonMap["tokenEstimatePaddingPct"].(float64); ok {
		model.TokenEstimatePaddingPct = tokenEstimatePaddingPct
	}

	if reasoningBudget, ok := jsonMap["reasoningBudget"].(float64); ok {
		model.ReasoningBudget = int(reasoningBudget)
	}

	// Extract providers
	if providersArray, ok := jsonMap["providers"].([]interface{}); ok {
		for _, providerItem := range providersArray {
			if providerMap, ok := providerItem.(map[string]interface{}); ok {
				provider := shared.ModelProvider{}

				if providerName, ok := providerMap["provider"].(string); ok {
					provider.Provider = providerName
				} else {
					return nil, fmt.Errorf("missing or invalid 'provider' in providers array")
				}

				if modelName, ok := providerMap["modelName"].(string); ok {
					provider.ModelName = modelName
				} else {
					return nil, fmt.Errorf("missing or invalid 'modelName' in providers array")
				}

				if customProvider, ok := providerMap["customProvider"].(string); ok {
					provider.CustomProvider = customProvider
				}

				model.Providers = append(model.Providers, provider)
			}
		}
	}

	return model, nil
}
