package shared

import (
	"github.com/sashabaranov/go-openai"
)

var AvailableModels = []*AvailableModel{
	// Direct OpenAI models
	{
		Description:                 "OpenAI gpt-4o",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 16384,
		BaseModelConfig: BaseModelConfig{
			Provider:                   ModelProviderOpenAI,
			ModelName:                  openai.GPT4o,
			MaxTokens:                  128000,
			ApiKeyEnvVar:               OpenAIEnvVar,
			ModelCompatibility:         fullCompatibility,
			BaseUrl:                    OpenAIV1BaseUrl,
			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
			PredictedOutputEnabled:     true,
		},
	},
	{
		Description:                 "OpenAI gpt-4o-mini",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 16384,
		BaseModelConfig: BaseModelConfig{
			Provider:                   ModelProviderOpenAI,
			ModelName:                  "gpt-4o-mini",
			MaxTokens:                  128000,
			ApiKeyEnvVar:               OpenAIEnvVar,
			ModelCompatibility:         fullCompatibility,
			BaseUrl:                    OpenAIV1BaseUrl,
			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
			PredictedOutputEnabled:     true,
		},
	},
	{
		Description:                 "OpenAI o1-mini",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 65536,
		BaseModelConfig: BaseModelConfig{
			Provider:                   ModelProviderOpenAI,
			ModelName:                  "o1-mini",
			MaxTokens:                  128000,
			ApiKeyEnvVar:               OpenAIEnvVar,
			ModelCompatibility:         fullCompatibility,
			BaseUrl:                    OpenAIV1BaseUrl,
			PreferredModelOutputFormat: ModelOutputFormatXml,
			SystemPromptDisabled:       true,
			RoleParamsDisabled:         true,
		},
	},
	{
		Description:                 "OpenAI o3-mini",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 100000,
		BaseModelConfig: BaseModelConfig{
			Provider:                   ModelProviderOpenAI,
			ModelName:                  "o3-mini",
			MaxTokens:                  200000,
			ApiKeyEnvVar:               OpenAIEnvVar,
			ModelCompatibility:         fullCompatibility,
			BaseUrl:                    OpenAIV1BaseUrl,
			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
			RoleParamsDisabled:         true,
		},
	},
	{
		Description:                 "OpenAI o1",
		DefaultMaxConvoTokens:       15000,
		DefaultReservedOutputTokens: 100000,
		BaseModelConfig: BaseModelConfig{
			Provider:                   ModelProviderOpenAI,
			ModelName:                  "o1",
			MaxTokens:                  200000,
			ApiKeyEnvVar:               OpenAIEnvVar,
			ModelCompatibility:         fullCompatibility,
			BaseUrl:                    OpenAIV1BaseUrl,
			PreferredModelOutputFormat: ModelOutputFormatXml,
			SystemPromptDisabled:       true,
			RoleParamsDisabled:         true,
		},
	},

	// OpenRouter models
	{
		Description:                 "Anthropic Claude 3.5 Sonnet via OpenRouter",
		DefaultMaxConvoTokens:       15000,
		DefaultReservedOutputTokens: 8192,
		BaseModelConfig: BaseModelConfig{
			Provider:                     ModelProviderOpenRouter,
			ModelName:                    "anthropic/claude-3.5-sonnet",
			MaxTokens:                    200000,
			ApiKeyEnvVar:                 ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility:           fullCompatibility,
			BaseUrl:                      BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat:   ModelOutputFormatXml,
			PreferredOpenRouterProviders: DefaultOpenRouterProvidersByFamily[OpenRouterFamilyAnthropic],
			OpenRouterAllowFallbacks:     DefaultOpenRouterAllowFallbacksByFamily[OpenRouterFamilyAnthropic],
		},
	},
	{
		Description:                 "Anthropic Claude 3.5 Haiku via OpenRouter",
		DefaultMaxConvoTokens:       15000,
		DefaultReservedOutputTokens: 8192,
		BaseModelConfig: BaseModelConfig{
			Provider:                     ModelProviderOpenRouter,
			ModelName:                    "anthropic/claude-3.5-haiku",
			MaxTokens:                    200000,
			ApiKeyEnvVar:                 ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility:           fullCompatibility,
			BaseUrl:                      BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat:   ModelOutputFormatXml,
			PreferredOpenRouterProviders: DefaultOpenRouterProvidersByFamily[OpenRouterFamilyAnthropic],
			OpenRouterAllowFallbacks:     DefaultOpenRouterAllowFallbacksByFamily[OpenRouterFamilyAnthropic],
		},
	},
	{
		Description:                 "Google Gemini Pro 1.5 via OpenRouter",
		DefaultMaxConvoTokens:       100000,
		DefaultReservedOutputTokens: 8192,
		BaseModelConfig: BaseModelConfig{
			Provider:                     ModelProviderOpenRouter,
			ModelName:                    "google/gemini-pro-1.5",
			MaxTokens:                    2000000,
			ApiKeyEnvVar:                 ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility:           fullCompatibility,
			BaseUrl:                      BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat:   ModelOutputFormatXml,
			PreferredOpenRouterProviders: DefaultOpenRouterProvidersByFamily[OpenRouterFamilyGoogle],
			OpenRouterAllowFallbacks:     DefaultOpenRouterAllowFallbacksByFamily[OpenRouterFamilyGoogle],
		},
	},
	{
		Description:                 "Google Gemini Flash 1.5 via OpenRouter",
		DefaultMaxConvoTokens:       75000,
		DefaultReservedOutputTokens: 8192,
		BaseModelConfig: BaseModelConfig{
			Provider:                     ModelProviderOpenRouter,
			ModelName:                    "google/gemini-flash-1.5",
			MaxTokens:                    1000000,
			ApiKeyEnvVar:                 ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility:           fullCompatibility,
			BaseUrl:                      BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat:   ModelOutputFormatXml,
			PreferredOpenRouterProviders: DefaultOpenRouterProvidersByFamily[OpenRouterFamilyGoogle],
			OpenRouterAllowFallbacks:     DefaultOpenRouterAllowFallbacksByFamily[OpenRouterFamilyGoogle],
		},
	},
	{
		Description:                 "Google Gemini Flash 2.0 Experimental via OpenRouter",
		DefaultMaxConvoTokens:       75000,
		DefaultReservedOutputTokens: 8192,
		BaseModelConfig: BaseModelConfig{
			Provider:                     ModelProviderOpenRouter,
			ModelName:                    "google/gemini-2.0-flash-exp:free",
			MaxTokens:                    1000000,
			ApiKeyEnvVar:                 ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility:           fullCompatibility,
			BaseUrl:                      BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat:   ModelOutputFormatXml,
			PreferredOpenRouterProviders: DefaultOpenRouterProvidersByFamily[OpenRouterFamilyGoogle],
			OpenRouterAllowFallbacks:     DefaultOpenRouterAllowFallbacksByFamily[OpenRouterFamilyGoogle],
		},
	},

	{
		Description:                 "DeepSeek V3 via OpenRouter",
		DefaultMaxConvoTokens:       7500,
		DefaultReservedOutputTokens: 8192,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderOpenRouter,
			ModelName:    "deepseek/deepseek-chat",
			MaxTokens:    64000,
			ApiKeyEnvVar: ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility: ModelCompatibility{
				HasImageSupport: false,
			},
			BaseUrl:                      BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat:   ModelOutputFormatXml,
			PreferredOpenRouterProviders: DefaultOpenRouterProvidersByFamily[OpenRouterFamilyDeepSeek],
			OpenRouterAllowFallbacks:     DefaultOpenRouterAllowFallbacksByFamily[OpenRouterFamilyDeepSeek],
		},
	},

	{
		Description:                 "DeepSeek R1 via OpenRouter",
		DefaultMaxConvoTokens:       7500,
		DefaultReservedOutputTokens: 8192,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderOpenRouter,
			ModelName:    "deepseek/deepseek-r1",
			MaxTokens:    64000,
			ApiKeyEnvVar: ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility: ModelCompatibility{
				HasImageSupport: false,
			},
			BaseUrl:                      BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat:   ModelOutputFormatXml,
			PreferredOpenRouterProviders: DefaultOpenRouterProvidersByFamily[OpenRouterFamilyDeepSeek],
			OpenRouterAllowFallbacks:     DefaultOpenRouterAllowFallbacksByFamily[OpenRouterFamilyDeepSeek],
		},
	},

	{
		Description:                 "DeepSeek R1 Distill Llama 70B via OpenRouter/DeepInfra",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 131072,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderOpenRouter,
			ModelName:    "deepseek/deepseek-r1-distill-llama-70b",
			MaxTokens:    131072,
			ApiKeyEnvVar: ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility: ModelCompatibility{
				HasImageSupport: false,
			},
			BaseUrl:                      BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat:   ModelOutputFormatXml,
			PreferredOpenRouterProviders: []OpenRouterProvider{OpenRouterProviderDeepInfra},
			OpenRouterAllowFallbacks:     false,
		},
	},

	{
		Description:                 "DeepSeek R1 Distill Qwen 32B via OpenRouter/Fireworks",
		DefaultMaxConvoTokens:       12500,
		DefaultReservedOutputTokens: 81920,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderOpenRouter,
			ModelName:    "deepseek/deepseek-r1-distill-qwen-32b",
			MaxTokens:    163840,
			ApiKeyEnvVar: ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility: ModelCompatibility{
				HasImageSupport: false,
			},
			BaseUrl:                      BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat:   ModelOutputFormatXml,
			PreferredOpenRouterProviders: []OpenRouterProvider{OpenRouterProviderFireworks},
			OpenRouterAllowFallbacks:     false,
		},
	},

	{
		Description:                 "Qwen 2.5 Coder 32B via OpenRouter/Hyperbolic",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 8192,
		BaseModelConfig: BaseModelConfig{
			Provider:                     ModelProviderOpenRouter,
			ModelName:                    "qwen/qwen-2.5-coder-32b-instruct",
			MaxTokens:                    128000,
			ApiKeyEnvVar:                 ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility:           fullCompatibility,
			BaseUrl:                      BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat:   ModelOutputFormatXml,
			PreferredOpenRouterProviders: DefaultOpenRouterProvidersByFamily[OpenRouterFamilyQwen],
			OpenRouterAllowFallbacks:     DefaultOpenRouterAllowFallbacksByFamily[OpenRouterFamilyQwen],
		},
	},

	// OpenAI models via OpenRouter
	{
		Description:                 "OpenAI gpt-4o via OpenRouter",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 16384,
		BaseModelConfig: BaseModelConfig{
			Provider:                     ModelProviderOpenRouter,
			ModelName:                    "openai/gpt-4o",
			MaxTokens:                    128000,
			ApiKeyEnvVar:                 ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility:           fullCompatibility,
			BaseUrl:                      BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat:   ModelOutputFormatToolCallJson,
			PredictedOutputEnabled:       true,
			PreferredOpenRouterProviders: DefaultOpenRouterProvidersByFamily[OpenRouterFamilyOpenAI],
			OpenRouterAllowFallbacks:     DefaultOpenRouterAllowFallbacksByFamily[OpenRouterFamilyOpenAI],
		},
	},
	{
		Description:                 "OpenAI gpt-4o-mini via OpenRouter",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 16384,
		BaseModelConfig: BaseModelConfig{
			Provider:                     ModelProviderOpenRouter,
			ModelName:                    "openai/gpt-4o-mini",
			MaxTokens:                    128000,
			ApiKeyEnvVar:                 ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility:           fullCompatibility,
			BaseUrl:                      BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat:   ModelOutputFormatToolCallJson,
			PredictedOutputEnabled:       true,
			PreferredOpenRouterProviders: DefaultOpenRouterProvidersByFamily[OpenRouterFamilyOpenAI],
			OpenRouterAllowFallbacks:     DefaultOpenRouterAllowFallbacksByFamily[OpenRouterFamilyOpenAI],
		},
	},

	// These only have OpenAI as a provider for now and o1/o3-mini both require adding OpenAI API keys in OpenRouter; better to use the direct OpenAI models until this changes
	// {
	// 	Description:                 "OpenAI o1-mini via OpenRouter",
	// 	DefaultMaxConvoTokens:       10000,
	// 	DefaultReservedOutputTokens: 65536,
	// 	BaseModelConfig: BaseModelConfig{
	// 		Provider:                     ModelProviderOpenRouter,
	// 		ModelName:                    "openai/o1-mini",
	// 		MaxTokens:                    128000,
	// 		ApiKeyEnvVar:                 ApiKeyByProvider[ModelProviderOpenRouter],
	// 		ModelCompatibility:           fullCompatibility,
	// 		BaseUrl:                      BaseUrlByProvider[ModelProviderOpenRouter],
	// 		PreferredModelOutputFormat:   ModelOutputFormatXml,
	// 		SystemPromptDisabled:         true,
	// 		RoleParamsDisabled:           true,
	// 		PreferredOpenRouterProviders: DefaultOpenRouterProvidersByFamily[OpenRouterFamilyOpenAI],
	// 		OpenRouterAllowFallbacks:     DefaultOpenRouterAllowFallbacksByFamily[OpenRouterFamilyOpenAI],
	// 	},
	// },
	// {
	// 	Description:                 "OpenAI o3-mini via OpenRouter",
	// 	DefaultMaxConvoTokens:       10000,
	// 	DefaultReservedOutputTokens: 100000,
	// 	BaseModelConfig: BaseModelConfig{
	// 		Provider:                     ModelProviderOpenRouter,
	// 		ModelName:                    "openai/o3-mini",
	// 		MaxTokens:                    200000,
	// 		ApiKeyEnvVar:                 ApiKeyByProvider[ModelProviderOpenRouter],
	// 		ModelCompatibility:           fullCompatibility,
	// 		BaseUrl:                      BaseUrlByProvider[ModelProviderOpenRouter],
	// 		PreferredModelOutputFormat:   ModelOutputFormatXml,
	// 		SystemPromptDisabled:         true,
	// 		RoleParamsDisabled:           true,
	// 		PreferredOpenRouterProviders: DefaultOpenRouterProvidersByFamily[OpenRouterFamilyOpenAI],
	// 		OpenRouterAllowFallbacks:     DefaultOpenRouterAllowFallbacksByFamily[OpenRouterFamilyOpenAI],
	// 	},
	// },
	// {
	// 	Description:                 "OpenAI o1 via OpenRouter",
	// 	DefaultMaxConvoTokens:       15000,
	// 	DefaultReservedOutputTokens: 100000,
	// 	BaseModelConfig: BaseModelConfig{
	// 		Provider:                     ModelProviderOpenRouter,
	// 		ModelName:                    "openai/o1",
	// 		MaxTokens:                    200000,
	// 		ApiKeyEnvVar:                 ApiKeyByProvider[ModelProviderOpenRouter],
	// 		ModelCompatibility:           fullCompatibility,
	// 		BaseUrl:                      BaseUrlByProvider[ModelProviderOpenRouter],
	// 		PreferredModelOutputFormat:   ModelOutputFormatXml,
	// 		SystemPromptDisabled:         true,
	// 		RoleParamsDisabled:           true,
	// 		PreferredOpenRouterProviders: DefaultOpenRouterProvidersByFamily[OpenRouterFamilyOpenAI],
	// 		OpenRouterAllowFallbacks:     DefaultOpenRouterAllowFallbacksByFamily[OpenRouterFamilyOpenAI],
	// 	},
	// },
}

var AvailableModelsByComposite = map[string]*AvailableModel{}

func init() {
	for _, model := range AvailableModels {
		compositeKey := string(model.Provider) + "/" + model.ModelName
		AvailableModelsByComposite[compositeKey] = model
	}
}

func GetAvailableModel(provider ModelProvider, modelName string) *AvailableModel {
	compositeKey := string(provider) + "/" + modelName
	return AvailableModelsByComposite[compositeKey]
}
