package shared

import "time"

type CustomModel struct {
	Id          string         `json:"id"`
	ModelId     ModelId        `json:"modelId"`
	Publisher   ModelPublisher `json:"publisher"`
	Description string         `json:"description"`

	BaseModelShared

	Providers []BaseModelUsesProvider `json:"providers"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CustomProvider struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	BaseUrl string `json:"baseUrl"`

	// for AWS Bedrock models
	HasAWSAuth bool `json:"hasAWSAuth,omitempty"`

	// for local models that don't require auth (ollama, etc.)
	SkipAuth bool `json:"skipAuth,omitempty"`

	ApiKeyEnvVar  string                       `json:"apiKeyEnvVar,omitempty"`
	ExtraAuthVars []ModelProviderExtraAuthVars `json:"extraAuthVars,omitempty"`
}
