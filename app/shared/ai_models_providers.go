package shared

const OpenAIV1BaseUrl = "https://api.openai.com/v1"
const OpenRouterBaseUrl = "https://openrouter.ai/api/v1"
const LiteLLMBaseUrl = "http://localhost:4000/v1" // runs in the same container alongside the plandex server

const OpenAIEnvVar = "OPENAI_API_KEY"
const OpenRouterApiKeyEnvVar = "OPENROUTER_API_KEY"
const AnthropicApiKeyEnvVar = "ANTHROPIC_API_KEY"
const GoogleAIStudioApiKeyEnvVar = "GEMINI_API_KEY"
const GoogleVertexApiKeyEnvVar = "VERTEX_API_KEY"
const AzureOpenAIEnvVar = "AZURE_OPENAI_API_KEY"
const DeepSeekApiKeyEnvVar = "DEEPSEEK_API_KEY"
const PerplexityApiKeyEnvVar = "PERPLEXITY_API_KEY"

type ModelProvider string

const (
	ModelProviderOpenRouter ModelProvider = "openrouter"
	ModelProviderOpenAI     ModelProvider = "openai"

	ModelProviderAnthropic      ModelProvider = "anthropic"
	ModelProviderGoogleAIStudio ModelProvider = "google-ai-studio"
	ModelProviderGoogleVertex   ModelProvider = "google-vertex"
	ModelProviderAzureOpenAI    ModelProvider = "azure-openai"
	ModelProviderDeepSeek       ModelProvider = "deepseek"
	ModelProviderPerplexity     ModelProvider = "perplexity"

	ModelProviderCustom ModelProvider = "custom"
)

var AllModelProviders = []string{
	string(ModelProviderOpenAI),
	string(ModelProviderOpenRouter),
	string(ModelProviderAnthropic),
	string(ModelProviderGoogleAIStudio),
	string(ModelProviderGoogleVertex),
	string(ModelProviderAzureOpenAI),
	string(ModelProviderDeepSeek),
	string(ModelProviderPerplexity),
	string(ModelProviderCustom),
}

var BaseUrlByProvider = map[ModelProvider]string{
	ModelProviderOpenAI:     OpenAIV1BaseUrl,
	ModelProviderOpenRouter: OpenRouterBaseUrl,

	// apart from openai and openrouter, the rest are supported by liteLLM
	ModelProviderAnthropic:      LiteLLMBaseUrl,
	ModelProviderGoogleAIStudio: LiteLLMBaseUrl,
	ModelProviderGoogleVertex:   LiteLLMBaseUrl,
	ModelProviderAzureOpenAI:    LiteLLMBaseUrl,
	ModelProviderDeepSeek:       LiteLLMBaseUrl,
	ModelProviderPerplexity:     LiteLLMBaseUrl,
}

var ApiKeyByProvider = map[ModelProvider]string{
	ModelProviderOpenAI:     OpenAIEnvVar,
	ModelProviderOpenRouter: OpenRouterApiKeyEnvVar,

	ModelProviderAnthropic:      AnthropicApiKeyEnvVar,
	ModelProviderGoogleAIStudio: GoogleAIStudioApiKeyEnvVar,
	ModelProviderGoogleVertex:   GoogleVertexApiKeyEnvVar,
	ModelProviderAzureOpenAI:    AzureOpenAIEnvVar,
	ModelProviderDeepSeek:       DeepSeekApiKeyEnvVar,
	ModelProviderPerplexity:     PerplexityApiKeyEnvVar,
}

// these types can come in via JSON config which uses JSON schema with discriminated unions — thus the need for the `omitempty` and pointers on all optional fields
type ModelProviderExtraAuthVars struct {
	Var        string `json:"var"`
	IsFilePath *bool  `json:"isFilePath,omitempty"`
	Required   *bool  `json:"required,omitempty"`
}

type ModelProviderConfig struct {
	SchemaVersion  string        `json:"schemaVersion"`
	Provider       ModelProvider `json:"provider"`
	CustomProvider *string       `json:"customProvider,omitempty"`
	BaseUrl        string        `json:"baseUrl"`

	// for local models that don't require auth (ollama, etc.)
	SkipAuth *bool `json:"skipAuth,omitempty"`

	ApiKeyEnvVar  *string                       `json:"apiKeyEnvVar,omitempty"`
	ExtraAuthVars *[]ModelProviderExtraAuthVars `json:"extraAuthVars,omitempty"`
}

var BuiltInModelProviderConfigs = map[ModelProvider]ModelProviderConfig{
	ModelProviderOpenAI: {
		Provider: ModelProviderOpenAI,
		BaseUrl:  OpenAIV1BaseUrl,

		SkipAuth:      &[]bool{false}[0],
		ApiKeyEnvVar:  &[]string{OpenAIEnvVar}[0],
		ExtraAuthVars: &[]ModelProviderExtraAuthVars{},
	},
	ModelProviderOpenRouter: {
		Provider: ModelProviderOpenRouter,
		BaseUrl:  OpenRouterBaseUrl,

		SkipAuth:      &[]bool{true}[0],
		ApiKeyEnvVar:  &[]string{OpenRouterApiKeyEnvVar}[0],
		ExtraAuthVars: &[]ModelProviderExtraAuthVars{},
	},
}
