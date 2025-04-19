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
