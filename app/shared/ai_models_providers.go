package shared

const OpenAIEnvVar = "OPENAI_API_KEY"
const OpenAIV1BaseUrl = "https://api.openai.com/v1"
const OpenRouterApiKeyEnvVar = "OPENROUTER_API_KEY"
const OpenRouterBaseUrl = "https://openrouter.ai/api/v1"

type ModelProvider string

const (
	ModelProviderOpenRouter ModelProvider = "openrouter"
	ModelProviderOpenAI     ModelProvider = "openai"
	ModelProviderJules      ModelProvider = "jules"
	ModelProviderCustom     ModelProvider = "custom"
)

var AllModelProviders = []string{
	string(ModelProviderOpenAI),
	string(ModelProviderOpenRouter),
	string(ModelProviderJules),
	// string(ModelProviderTogether),
	string(ModelProviderCustom),
}

var BaseUrlByProvider = map[ModelProvider]string{
	ModelProviderOpenAI:     OpenAIV1BaseUrl,
	ModelProviderOpenRouter: OpenRouterBaseUrl,
	ModelProviderJules:      "https://mock-jules-api.example.com/v1",
}

var ApiKeyByProvider = map[ModelProvider]string{
	ModelProviderOpenAI:     OpenAIEnvVar,
	ModelProviderOpenRouter: OpenRouterApiKeyEnvVar,
	ModelProviderJules:      "JULES_API_KEY",
}
