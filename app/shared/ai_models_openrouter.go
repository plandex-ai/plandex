package shared

type OpenRouterFamily string

const (
	OpenRouterFamilyAnthropic OpenRouterFamily = "anthropic"
	OpenRouterFamilyGoogle    OpenRouterFamily = "google"
	OpenRouterFamilyOpenAI    OpenRouterFamily = "openai"
	OpenRouterFamilyQwen      OpenRouterFamily = "qwen"
	OpenRouterFamilyDeepSeek  OpenRouterFamily = "deepseek"
)

type OpenRouterProvider string

const (
	OpenRouterProviderAnthropic OpenRouterProvider = "Anthropic"
	OpenRouterProviderGoogle    OpenRouterProvider = "Google Vertex"
	OpenRouterProviderOpenAI    OpenRouterProvider = "OpenAI"
	OpenRouterProviderDeepSeek  OpenRouterProvider = "DeepSeek"
	OpenRouterProviderQwen      OpenRouterProvider = "Hyperbolic"
	OpenRouterProviderDeepInfra OpenRouterProvider = "DeepInfra"
	OpenRouterProviderFireworks OpenRouterProvider = "Fireworks"
)

var DefaultOpenRouterProvidersByFamily = map[OpenRouterFamily][]OpenRouterProvider{
	OpenRouterFamilyAnthropic: {OpenRouterProviderAnthropic},
	OpenRouterFamilyGoogle:    {OpenRouterProviderGoogle},
	OpenRouterFamilyOpenAI:    {OpenRouterProviderOpenAI},
	OpenRouterFamilyQwen:      {OpenRouterProviderQwen},
	OpenRouterFamilyDeepSeek:  {OpenRouterProviderDeepSeek},
}

// open source models don't have fallbacks enabled for now because pricing and context limits aren't predictable across providers
var DefaultOpenRouterAllowFallbacksByFamily = map[OpenRouterFamily]bool{
	OpenRouterFamilyAnthropic: false,
	OpenRouterFamilyGoogle:    false,
	OpenRouterFamilyOpenAI:    false,
}
