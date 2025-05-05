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

	ModelProviderAmazonBedrock ModelProvider = "aws-bedrock"

	ModelProviderOllama ModelProvider = "ollama"

	ModelProviderCustom ModelProvider = "custom"
)

var ModelProviderToLiteLLMId = map[ModelProvider]string{
	ModelProviderGoogleAIStudio: "google_ai_studio",
	ModelProviderGoogleVertex:   "vertex_ai",
	ModelProviderAzureOpenAI:    "azure",
	ModelProviderAmazonBedrock:  "bedrock",
}

var AllModelProviders = []string{
	string(ModelProviderOpenAI),
	string(ModelProviderOpenRouter),
	string(ModelProviderAnthropic),
	string(ModelProviderGoogleAIStudio),
	string(ModelProviderGoogleVertex),
	string(ModelProviderAzureOpenAI),
	string(ModelProviderDeepSeek),
	string(ModelProviderPerplexity),
	string(ModelProviderAmazonBedrock),
	string(ModelProviderCustom),
}

// var BaseUrlByProvider = map[ModelProvider]string{
// 	ModelProviderOpenAI:     OpenAIV1BaseUrl,
// 	ModelProviderOpenRouter: OpenRouterBaseUrl,

// 	// apart from openai and openrouter, the rest are supported by liteLLM
// 	ModelProviderAnthropic:      LiteLLMBaseUrl,
// 	ModelProviderGoogleAIStudio: LiteLLMBaseUrl,
// 	ModelProviderGoogleVertex:   LiteLLMBaseUrl,
// 	ModelProviderAzureOpenAI:    LiteLLMBaseUrl,
// 	ModelProviderDeepSeek:       LiteLLMBaseUrl,
// 	ModelProviderPerplexity:     LiteLLMBaseUrl,
// 	ModelProviderAmazonBedrock:  LiteLLMBaseUrl,
// }

// var ApiKeyByProvider = map[ModelProvider]string{
// 	ModelProviderOpenAI:     OpenAIEnvVar,
// 	ModelProviderOpenRouter: OpenRouterApiKeyEnvVar,

// 	ModelProviderAnthropic:      AnthropicApiKeyEnvVar,
// 	ModelProviderGoogleAIStudio: GoogleAIStudioApiKeyEnvVar,
// 	ModelProviderGoogleVertex:   GoogleVertexApiKeyEnvVar,
// 	ModelProviderAzureOpenAI:    AzureOpenAIEnvVar,
// 	ModelProviderDeepSeek:       DeepSeekApiKeyEnvVar,
// 	ModelProviderPerplexity:     PerplexityApiKeyEnvVar,
// }

type ModelProviderExtraAuthVars struct {
	Var        string `json:"var"`
	IsFilePath bool   `json:"isFilePath,omitempty"`
	Required   bool   `json:"required,omitempty"`
	Default    string `json:"default,omitempty"`
}

type ModelProviderConfigSchema struct {
	Provider       ModelProvider `json:"provider"`
	CustomProvider *string       `json:"customProvider,omitempty"`
	BaseUrl        string        `json:"baseUrl"`

	// for AWS Bedrock models
	HasAWSAuth bool `json:"hasAWSAuth,omitempty"`

	// for local models that don't require auth (ollama, etc.)
	SkipAuth bool `json:"skipAuth,omitempty"`

	ApiKeyEnvVar  string                       `json:"apiKeyEnvVar,omitempty"`
	ExtraAuthVars []ModelProviderExtraAuthVars `json:"extraAuthVars,omitempty"`
}

const DefaultAzureApiVersion = "2024-10-21"
const AnthropicMaxReasoningBudget = 32000

var BuiltInModelProviderConfigs = map[ModelProvider]ModelProviderConfigSchema{
	ModelProviderOpenAI: {
		Provider:     ModelProviderOpenAI,
		BaseUrl:      OpenAIV1BaseUrl,
		ApiKeyEnvVar: OpenAIEnvVar,
	},
	ModelProviderOpenRouter: {
		Provider:     ModelProviderOpenRouter,
		BaseUrl:      OpenRouterBaseUrl,
		ApiKeyEnvVar: OpenRouterApiKeyEnvVar,
	},
	ModelProviderAnthropic: {
		Provider:     ModelProviderAnthropic,
		BaseUrl:      LiteLLMBaseUrl,
		ApiKeyEnvVar: AnthropicApiKeyEnvVar,
	},
	ModelProviderGoogleAIStudio: {
		Provider:     ModelProviderGoogleAIStudio,
		BaseUrl:      LiteLLMBaseUrl,
		ApiKeyEnvVar: GoogleAIStudioApiKeyEnvVar,
	},
	ModelProviderGoogleVertex: {
		Provider: ModelProviderGoogleVertex,
		BaseUrl:  LiteLLMBaseUrl,
		ExtraAuthVars: []ModelProviderExtraAuthVars{
			{
				Var:        "GOOGLE_APPLICATION_CREDENTIALS",
				IsFilePath: true,
				Required:   true,
			},
			{
				Var:      "VERTEXAI_PROJECT",
				Required: true,
			},
			{
				Var:      "VERTEXAI_LOCATION",
				Required: true,
			},
		},
	},
	ModelProviderAzureOpenAI: {
		Provider: ModelProviderAzureOpenAI,
		BaseUrl:  LiteLLMBaseUrl,
		ExtraAuthVars: []ModelProviderExtraAuthVars{
			{
				Var:      "AZURE_OPENAI_API_KEY",
				Required: true,
			},
			{
				Var:      "AZURE_API_BASE",
				Required: true,
			},
			{
				Var:      "AZURE_API_VERSION",
				Required: false,
				Default:  DefaultAzureApiVersion,
			},
		},
	},
	ModelProviderDeepSeek: {
		Provider:     ModelProviderDeepSeek,
		BaseUrl:      LiteLLMBaseUrl,
		ApiKeyEnvVar: DeepSeekApiKeyEnvVar,
	},
	ModelProviderPerplexity: {
		Provider:     ModelProviderPerplexity,
		BaseUrl:      LiteLLMBaseUrl,
		ApiKeyEnvVar: PerplexityApiKeyEnvVar,
	},
	ModelProviderAmazonBedrock: {
		Provider:   ModelProviderAmazonBedrock,
		BaseUrl:    LiteLLMBaseUrl,
		HasAWSAuth: true,
		ExtraAuthVars: []ModelProviderExtraAuthVars{
			{Var: "AWS_ACCESS_KEY_ID", Required: false},
			{Var: "AWS_SECRET_ACCESS_KEY", Required: false},
			{Var: "AWS_REGION", Required: false},
		},
	},
	ModelProviderOllama: {
		Provider: ModelProviderOllama,
		BaseUrl:  LiteLLMBaseUrl,
		SkipAuth: true,
	},
}
