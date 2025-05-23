package shared

import "time"

type SchemaUrl string

const SchemaUrlInputConfig SchemaUrl = "https://plandex.ai/schemas/models-input.schema.json"

type CustomModel struct {
	Id          string         `json:"id,omitempty"`
	ModelId     ModelId        `json:"modelId"`
	Publisher   ModelPublisher `json:"publisher"`
	Description string         `json:"description"`

	BaseModelShared

	Providers []BaseModelUsesProvider `json:"providers"`

	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type CustomProvider struct {
	Id      string `json:"id,omitempty"`
	Name    string `json:"name"`
	BaseUrl string `json:"baseUrl"`

	// for AWS Bedrock models
	HasAWSAuth bool `json:"hasAWSAuth,omitempty"`

	// for local models that don't require auth (ollama, etc.)
	SkipAuth bool `json:"skipAuth,omitempty"`

	ApiKeyEnvVar  string                       `json:"apiKeyEnvVar,omitempty"`
	ExtraAuthVars []ModelProviderExtraAuthVars `json:"extraAuthVars,omitempty"`

	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type ModelsInput struct {
	SchemaUrl        SchemaUrl         `json:"schemaUrl"`
	CustomModels     []CustomModel     `json:"customModels"`
	CustomProviders  []CustomProvider  `json:"customProviders"`
	CustomModelPacks []ModelPackSchema `json:"customModelPacks"`
}
