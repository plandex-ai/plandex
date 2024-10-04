package shared

import (
	"github.com/sashabaranov/go-openai"
)

const OpenAIEnvVar = "OPENAI_API_KEY"
const OpenAIV1BaseUrl = "https://api.openai.com/v1"

var fullCompatibility = ModelCompatibility{
	IsOpenAICompatible:        true,
	HasJsonResponseMode:       true,
	HasStreaming:              true,
	HasFunctionCalling:        true,
	HasStreamingFunctionCalls: true,
	HasImageSupport:           true,
}

var fullCompatibilityExceptImage = ModelCompatibility{
	IsOpenAICompatible:        true,
	HasJsonResponseMode:       true,
	HasStreaming:              true,
	HasFunctionCalling:        true,
	HasStreamingFunctionCalls: true,
	HasImageSupport:           false,
}

var AvailableModels = []*AvailableModel{
	{
		Description:                 "OpenAI's latest gpt-4o model",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 16384,
		BaseModelConfig: BaseModelConfig{
			Provider:           ModelProviderOpenAI,
			ModelName:          openai.GPT4o,
			MaxTokens:          128000,
			ApiKeyEnvVar:       OpenAIEnvVar,
			ModelCompatibility: fullCompatibility,
			BaseUrl:            OpenAIV1BaseUrl,
		},
	},
	{
		Description:                 "OpenAI's latest gpt-4o-mini model",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 16384,
		BaseModelConfig: BaseModelConfig{
			Provider:           ModelProviderOpenAI,
			ModelName:          "gpt-4o-mini",
			MaxTokens:          128000,
			ApiKeyEnvVar:       OpenAIEnvVar,
			ModelCompatibility: fullCompatibility,
			BaseUrl:            OpenAIV1BaseUrl,
		},
	},
	{
		Description:                 "Anthropic Claude 3.5 Sonnet via OpenRouter",
		DefaultMaxConvoTokens:       15000,
		DefaultReservedOutputTokens: 4096,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderOpenRouter,
			ModelName:    "anthropic/claude-3.5-sonnet",
			MaxTokens:    200000,
			ApiKeyEnvVar: ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility: ModelCompatibility{
				IsOpenAICompatible:        true,
				HasJsonResponseMode:       true,
				HasStreaming:              true,
				HasFunctionCalling:        true,
				HasStreamingFunctionCalls: false,
				HasImageSupport:           true,
			},
			BaseUrl: BaseUrlByProvider[ModelProviderOpenRouter],
		},
	},
	{
		Description:                 "Anthropic Claude 3 Haiku via OpenRouter",
		DefaultMaxConvoTokens:       15000,
		DefaultReservedOutputTokens: 4096,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderOpenRouter,
			ModelName:    "anthropic/claude-3-haiku",
			MaxTokens:    200000,
			ApiKeyEnvVar: ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility: ModelCompatibility{
				IsOpenAICompatible:        true,
				HasJsonResponseMode:       true,
				HasStreaming:              true,
				HasFunctionCalling:        true,
				HasStreamingFunctionCalls: false,
				HasImageSupport:           true,
			},
			BaseUrl: BaseUrlByProvider[ModelProviderOpenRouter],
		},
	},
	{
		Description:                 "Google Gemini Pro 1.5 via OpenRouter",
		DefaultMaxConvoTokens:       100000,
		DefaultReservedOutputTokens: 32768,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderOpenRouter,
			ModelName:    "google/gemini-pro-1.5",
			MaxTokens:    4000000,
			ApiKeyEnvVar: ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility: ModelCompatibility{
				IsOpenAICompatible:        true,
				HasJsonResponseMode:       true,
				HasStreaming:              true,
				HasFunctionCalling:        true,
				HasStreamingFunctionCalls: false,
				HasImageSupport:           true,
			},
			BaseUrl: BaseUrlByProvider[ModelProviderOpenRouter],
		},
	},
	{
		Description:                 "Google Gemini Flash 1.5 via OpenRouter",
		DefaultMaxConvoTokens:       100000,
		DefaultReservedOutputTokens: 32768,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderOpenRouter,
			ModelName:    "google/gemini-flash-1.5",
			MaxTokens:    4000000,
			ApiKeyEnvVar: ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility: ModelCompatibility{
				IsOpenAICompatible:        true,
				HasJsonResponseMode:       true,
				HasStreaming:              true,
				HasFunctionCalling:        true,
				HasStreamingFunctionCalls: false,
				HasImageSupport:           true,
			},
			BaseUrl: BaseUrlByProvider[ModelProviderOpenRouter],
		},
	},
	{
		Description:                 "DeepSeek V2.5 via OpenRouter",
		DefaultMaxConvoTokens:       15000,
		DefaultReservedOutputTokens: 4096,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderOpenRouter,
			ModelName:    "deepseek/deepseek-chat",
			MaxTokens:    128000,
			ApiKeyEnvVar: ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility: ModelCompatibility{
				IsOpenAICompatible:        true,
				HasJsonResponseMode:       true,
				HasStreaming:              true,
				HasFunctionCalling:        true,
				HasStreamingFunctionCalls: false,
				HasImageSupport:           false,
			},
			BaseUrl: BaseUrlByProvider[ModelProviderOpenRouter],
		},
	},
}

var AvailableModelsByName = map[string]*AvailableModel{}

var OpenRouterClaude3Dot5SonnetGPT4oModelPack ModelPack
var OpenRouterClaude3Dot5SonnetModelPack ModelPack
var Gpt4oLatestModelPack ModelPack
var Gpt4o8062024ModelPack ModelPack
var GoogleGeminiModelPack ModelPack

var BuiltInModelPacks = []*ModelPack{
	&Gpt4o8062024ModelPack,
	&Gpt4oLatestModelPack,
	&OpenRouterClaude3Dot5SonnetModelPack,
	&OpenRouterClaude3Dot5SonnetGPT4oModelPack,
	&GoogleGeminiModelPack,
}

var DefaultModelPack *ModelPack = &Gpt4oLatestModelPack

func getPlannerModelConfig(name string) PlannerModelConfig {
	return PlannerModelConfig{
		MaxConvoTokens:       AvailableModelsByName[name].DefaultMaxConvoTokens,
		ReservedOutputTokens: AvailableModelsByName[name].DefaultReservedOutputTokens,
	}
}

var DefaultConfigByRole = map[ModelRole]ModelRoleConfig{
	ModelRolePlanner: {
		Temperature: 0.3,
		TopP:        0.3,
	},
	ModelRolePlanSummary: {
		Temperature: 0.2,
		TopP:        0.2,
	},
	ModelRoleBuilder: {
		Temperature: 0.1,
		TopP:        0.1,
	},
	ModelRoleName: {
		Temperature: 0.8,
		TopP:        0.5,
	},
	ModelRoleCommitMsg: {
		Temperature: 0.8,
		TopP:        0.5,
	},
	ModelRoleExecStatus: {
		Temperature: 0.1,
		TopP:        0.1,
	},
	ModelRoleVerifier: {
		Temperature: 0.2,
		TopP:        0.2,
	},
	ModelRoleAutoFix: {
		Temperature: 0.1,
		TopP:        0.1,
	},
}

var RequiredCompatibilityByRole = map[ModelRole]ModelCompatibility{
	ModelRolePlanner: {
		IsOpenAICompatible: true,
		HasStreaming:       true,
	},
	ModelRolePlanSummary: {
		IsOpenAICompatible: true,
		HasStreaming:       true,
	},
	ModelRoleBuilder: {
		IsOpenAICompatible: true,
		HasStreaming:       true,
		HasFunctionCalling: true,
	},
	ModelRoleName: {
		IsOpenAICompatible: true,
	},
	ModelRoleCommitMsg: {
		IsOpenAICompatible: true,
		HasFunctionCalling: true,
	},
	ModelRoleExecStatus: {
		IsOpenAICompatible: true,
		HasFunctionCalling: true,
	},
	ModelRoleVerifier: {
		IsOpenAICompatible: true,
		HasStreaming:       true,
		HasFunctionCalling: true,
	},
	ModelRoleAutoFix: {
		IsOpenAICompatible: true,
		HasStreaming:       true,
		HasFunctionCalling: true,
	},
}

func init() {
	for _, model := range AvailableModels {
		AvailableModelsByName[model.ModelName] = model
	}

	Gpt4oLatestModelPack = ModelPack{
		Name:        "gpt-4o-latest",
		Description: "Uses OpenAI's latest gpt-4o model for heavy lifting, and latest version of gpt-4o-mini for lighter tasks.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig: ModelRoleConfig{
				Role:            ModelRolePlanner,
				BaseModelConfig: AvailableModelsByName[openai.GPT4o].BaseModelConfig,
				Temperature:     DefaultConfigByRole[ModelRolePlanner].Temperature,
				TopP:            DefaultConfigByRole[ModelRolePlanner].TopP,
			},
			PlannerModelConfig: getPlannerModelConfig(openai.GPT4o),
		},
		PlanSummary: ModelRoleConfig{
			Role:            ModelRolePlanSummary,
			BaseModelConfig: AvailableModelsByName[openai.GPT4o].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRolePlanSummary].Temperature,
			TopP:            DefaultConfigByRole[ModelRolePlanSummary].TopP,
		},
		Builder: ModelRoleConfig{
			Role:            ModelRoleBuilder,
			BaseModelConfig: AvailableModelsByName[openai.GPT4o].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleBuilder].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleBuilder].TopP,
		},
		Namer: ModelRoleConfig{
			Role:            ModelRoleName,
			BaseModelConfig: AvailableModelsByName["gpt-4o-mini"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleName].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleName].TopP,
		},
		CommitMsg: ModelRoleConfig{
			Role:            ModelRoleCommitMsg,
			BaseModelConfig: AvailableModelsByName["gpt-4o-mini"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleCommitMsg].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleCommitMsg].TopP,
		},
		ExecStatus: ModelRoleConfig{
			Role:            ModelRoleExecStatus,
			BaseModelConfig: AvailableModelsByName[openai.GPT4o].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleExecStatus].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleExecStatus].TopP,
		},
		Verifier: &ModelRoleConfig{
			Role:            ModelRoleVerifier,
			BaseModelConfig: AvailableModelsByName[openai.GPT4o].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleVerifier].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleVerifier].TopP,
		},
		AutoFix: &ModelRoleConfig{
			Role:            ModelRoleAutoFix,
			BaseModelConfig: AvailableModelsByName[openai.GPT4o].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleAutoFix].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleAutoFix].TopP,
		},
	}

	OpenRouterClaude3Dot5SonnetGPT4oModelPack = ModelPack{
		Name:        "anthropic-claude-3.5-sonnet-gpt-4o",
		Description: "Uses Anthropic's Claude 3.5 Sonnet model (via OpenRouter) for planning, summarization, and auto-continue, OpenAI gpt-4o for builds, and gpt-4o-mini for lighter tasks.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig: ModelRoleConfig{
				Role:            ModelRolePlanner,
				BaseModelConfig: AvailableModelsByName["anthropic/claude-3.5-sonnet"].BaseModelConfig,
				Temperature:     DefaultConfigByRole[ModelRolePlanner].Temperature,
				TopP:            DefaultConfigByRole[ModelRolePlanner].TopP,
			},
			PlannerModelConfig: getPlannerModelConfig("anthropic/claude-3.5-sonnet"),
		},
		PlanSummary: ModelRoleConfig{
			Role:            ModelRolePlanSummary,
			BaseModelConfig: AvailableModelsByName["anthropic/claude-3.5-sonnet"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRolePlanSummary].Temperature,
			TopP:            DefaultConfigByRole[ModelRolePlanSummary].TopP,
		},
		Builder: ModelRoleConfig{
			Role:            ModelRoleBuilder,
			BaseModelConfig: AvailableModelsByName[openai.GPT4o].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleBuilder].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleBuilder].TopP,
		},
		Namer: ModelRoleConfig{
			Role:            ModelRoleName,
			BaseModelConfig: AvailableModelsByName["gpt-4o-mini"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleName].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleName].TopP,
		},
		CommitMsg: ModelRoleConfig{
			Role:            ModelRoleCommitMsg,
			BaseModelConfig: AvailableModelsByName["gpt-4o-mini"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleCommitMsg].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleCommitMsg].TopP,
		},
		ExecStatus: ModelRoleConfig{
			Role:            ModelRoleExecStatus,
			BaseModelConfig: AvailableModelsByName[openai.GPT4o].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleExecStatus].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleExecStatus].TopP,
		},
		Verifier: &ModelRoleConfig{
			Role:            ModelRoleVerifier,
			BaseModelConfig: AvailableModelsByName[openai.GPT4o].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleVerifier].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleVerifier].TopP,
		},
		AutoFix: &ModelRoleConfig{
			Role:            ModelRoleAutoFix,
			BaseModelConfig: AvailableModelsByName[openai.GPT4o].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleAutoFix].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleAutoFix].TopP,
		},
	}

	OpenRouterClaude3Dot5SonnetModelPack = ModelPack{
		Name:        "anthropic-claude-3.5-sonnet",
		Description: "Uses Anthropic's Claude 3.5 Sonnet model (via OpenRouter) for planning, builds, and auto-continue, and summarization, and Claude 3 Haiku for lighter tasks.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig: ModelRoleConfig{
				Role:            ModelRolePlanner,
				BaseModelConfig: AvailableModelsByName["anthropic/claude-3.5-sonnet"].BaseModelConfig,
				Temperature:     DefaultConfigByRole[ModelRolePlanner].Temperature,
				TopP:            DefaultConfigByRole[ModelRolePlanner].TopP,
			},
			PlannerModelConfig: getPlannerModelConfig("anthropic/claude-3.5-sonnet"),
		},
		PlanSummary: ModelRoleConfig{
			Role:            ModelRolePlanSummary,
			BaseModelConfig: AvailableModelsByName["anthropic/claude-3.5-sonnet"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRolePlanSummary].Temperature,
			TopP:            DefaultConfigByRole[ModelRolePlanSummary].TopP,
		},
		Builder: ModelRoleConfig{
			Role:            ModelRoleBuilder,
			BaseModelConfig: AvailableModelsByName["anthropic/claude-3.5-sonnet"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleBuilder].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleBuilder].TopP,
		},
		Namer: ModelRoleConfig{
			Role:            ModelRoleName,
			BaseModelConfig: AvailableModelsByName["anthropic/claude-3-haiku"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleName].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleName].TopP,
		},
		CommitMsg: ModelRoleConfig{
			Role:            ModelRoleCommitMsg,
			BaseModelConfig: AvailableModelsByName["anthropic/claude-3-haiku"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleName].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleName].TopP,
		},
		ExecStatus: ModelRoleConfig{
			Role:            ModelRoleExecStatus,
			BaseModelConfig: AvailableModelsByName["anthropic/claude-3.5-sonnet"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleBuilder].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleBuilder].TopP,
		},
	}

	GoogleGeminiModelPack = ModelPack{
		Name:        "google-gemini",
		Description: "Uses Google's Gemini Pro 1.5 model for heavy lifting and Gemini Flash 1.5 for lighter tasks.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig: ModelRoleConfig{
				Role:            ModelRolePlanner,
				BaseModelConfig: AvailableModelsByName["google/gemini-pro-1.5"].BaseModelConfig,
				Temperature:     DefaultConfigByRole[ModelRolePlanner].Temperature,
				TopP:            DefaultConfigByRole[ModelRolePlanner].TopP,
			},
			PlannerModelConfig: getPlannerModelConfig("google/gemini-pro-1.5"),
		},
		PlanSummary: ModelRoleConfig{
			Role:            ModelRolePlanSummary,
			BaseModelConfig: AvailableModelsByName["google/gemini-pro-1.5"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRolePlanSummary].Temperature,
			TopP:            DefaultConfigByRole[ModelRolePlanSummary].TopP,
		},
		Builder: ModelRoleConfig{
			Role:            ModelRoleBuilder,
			BaseModelConfig: AvailableModelsByName["google/gemini-pro-1.5"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleBuilder].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleBuilder].TopP,
		},
		Namer: ModelRoleConfig{
			Role:            ModelRoleName,
			BaseModelConfig: AvailableModelsByName["google/gemini-flash-1.5"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleName].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleName].TopP,
		},
		CommitMsg: ModelRoleConfig{
			Role:            ModelRoleCommitMsg,
			BaseModelConfig: AvailableModelsByName["google/gemini-flash-1.5"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleCommitMsg].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleCommitMsg].TopP,
		},
		ExecStatus: ModelRoleConfig{
			Role:            ModelRoleExecStatus,
			BaseModelConfig: AvailableModelsByName["google/gemini-pro-1.5"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleExecStatus].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleExecStatus].TopP,
		},
		Verifier: &ModelRoleConfig{
			Role:            ModelRoleVerifier,
			BaseModelConfig: AvailableModelsByName["google/gemini-pro-1.5"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleVerifier].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleVerifier].TopP,
		},
		AutoFix: &ModelRoleConfig{
			Role:            ModelRoleAutoFix,
			BaseModelConfig: AvailableModelsByName["google/gemini-pro-1.5"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleAutoFix].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleAutoFix].TopP,
		},
	}
}

func FilterCompatibleModels(models []*AvailableModel, role ModelRole) []*AvailableModel {
	required := RequiredCompatibilityByRole[role]
	var compatibleModels []*AvailableModel

	for _, model := range models {
		if required.IsOpenAICompatible && !model.ModelCompatibility.IsOpenAICompatible {
			continue
		}
		if required.HasJsonResponseMode && !model.ModelCompatibility.HasJsonResponseMode {
			continue
		}
		if required.HasStreaming && !model.ModelCompatibility.HasStreaming {
			continue
		}
		if required.HasFunctionCalling && !model.ModelCompatibility.HasFunctionCalling {
			continue
		}
		if required.HasStreamingFunctionCalls && !model.ModelCompatibility.HasStreamingFunctionCalls {
			continue
		}

		compatibleModels = append(compatibleModels, model)
	}

	return compatibleModels
}
