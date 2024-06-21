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
		Description:                 "OpenAI's latest gpt-4o model, first released on 2024-05-13",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 4096,
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
		Description:                 "OpenAI's gpt-4o model, pinned to version released on 2024-05-13",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 4096,
		BaseModelConfig: BaseModelConfig{
			Provider:           ModelProviderOpenAI,
			ModelName:          "gpt-4o-2024-05-13",
			MaxTokens:          128000,
			ApiKeyEnvVar:       OpenAIEnvVar,
			ModelCompatibility: fullCompatibility,
			BaseUrl:            OpenAIV1BaseUrl,
		},
	},
	{
		Description:                 "OpenAI's latest gpt-4-turbo model, first released on 2024-04-09",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 4096,
		BaseModelConfig: BaseModelConfig{
			Provider:           ModelProviderOpenAI,
			ModelName:          openai.GPT4Turbo,
			MaxTokens:          128000,
			ApiKeyEnvVar:       OpenAIEnvVar,
			ModelCompatibility: fullCompatibilityExceptImage,
			BaseUrl:            OpenAIV1BaseUrl,
		}},

	{
		Description:                 "OpenAI's gpt-4-turbo, pinned to version released on 2024-04-09",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 4096,
		BaseModelConfig: BaseModelConfig{
			Provider:           ModelProviderOpenAI,
			ModelName:          openai.GPT4Turbo20240409,
			MaxTokens:          128000,
			ApiKeyEnvVar:       OpenAIEnvVar,
			ModelCompatibility: fullCompatibilityExceptImage,
			BaseUrl:            OpenAIV1BaseUrl,
		},
	},
	{
		Description:                 "OpenAI's gpt-4 model",
		DefaultMaxConvoTokens:       2500,
		DefaultReservedOutputTokens: 1000,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderOpenAI,
			ModelName:    openai.GPT4,
			MaxTokens:    8000,
			ApiKeyEnvVar: OpenAIEnvVar,
			ModelCompatibility: ModelCompatibility{
				IsOpenAICompatible:        true,
				HasJsonResponseMode:       false,
				HasStreaming:              true,
				HasFunctionCalling:        true,
				HasStreamingFunctionCalls: true,
			},
			BaseUrl: OpenAIV1BaseUrl,
		},
	},
	{
		Description:                 "OpenAI's latest gpt-3.5-turbo model",
		DefaultMaxConvoTokens:       5000,
		DefaultReservedOutputTokens: 2000,
		BaseModelConfig: BaseModelConfig{
			Provider:           ModelProviderOpenAI,
			ModelName:          openai.GPT3Dot5Turbo,
			MaxTokens:          16385,
			ApiKeyEnvVar:       OpenAIEnvVar,
			ModelCompatibility: fullCompatibilityExceptImage,
			BaseUrl:            OpenAIV1BaseUrl,
		},
	},
	{
		Description:                 "OpenAI's gpt-3.5-turbo, pinned to version released on 2024-01-25",
		DefaultMaxConvoTokens:       5000,
		DefaultReservedOutputTokens: 2000,
		BaseModelConfig: BaseModelConfig{
			Provider:           ModelProviderOpenAI,
			ModelName:          openai.GPT3Dot5Turbo0125,
			MaxTokens:          16385,
			ApiKeyEnvVar:       OpenAIEnvVar,
			ModelCompatibility: fullCompatibilityExceptImage,
			BaseUrl:            OpenAIV1BaseUrl,
		},
	},
	{
		Description:                 "OpenAI's gpt-3.5-turbo, pinned to version released on 2023-11-06",
		DefaultMaxConvoTokens:       5000,
		DefaultReservedOutputTokens: 2000,
		BaseModelConfig: BaseModelConfig{
			Provider:           ModelProviderOpenAI,
			ModelName:          openai.GPT3Dot5Turbo1106,
			MaxTokens:          16385,
			ApiKeyEnvVar:       OpenAIEnvVar,
			ModelCompatibility: fullCompatibilityExceptImage,
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
		Description:                 "Anthropic Claude 3 Opus via OpenRouter",
		DefaultMaxConvoTokens:       15000,
		DefaultReservedOutputTokens: 4096,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderOpenRouter,
			ModelName:    "anthropic/claude-3-opus",
			MaxTokens:    200000,
			ApiKeyEnvVar: ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility: ModelCompatibility{
				IsOpenAICompatible:        true,
				HasJsonResponseMode:       true,
				HasStreaming:              true,
				HasFunctionCalling:        true,
				HasStreamingFunctionCalls: false,
			},
			BaseUrl: BaseUrlByProvider[ModelProviderOpenRouter],
		},
	},
	{
		Description:                 "Anthropic Claude 3 Sonnet via OpenRouter",
		DefaultMaxConvoTokens:       15000,
		DefaultReservedOutputTokens: 4096,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderOpenRouter,
			ModelName:    "anthropic/claude-3-sonnet",
			MaxTokens:    200000,
			ApiKeyEnvVar: ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility: ModelCompatibility{
				IsOpenAICompatible:        true,
				HasJsonResponseMode:       true,
				HasStreaming:              true,
				HasFunctionCalling:        true,
				HasStreamingFunctionCalls: false,
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
			},
			BaseUrl: BaseUrlByProvider[ModelProviderOpenRouter],
		},
	},
	{
		Description:                 "Mixtral-8x22B via Together.ai",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 4096,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderTogether,
			ModelName:    "mistralai/Mixtral-8x22B-Instruct-v0.1",
			MaxTokens:    65536,
			ApiKeyEnvVar: ApiKeyByProvider[ModelProviderTogether],
			ModelCompatibility: ModelCompatibility{
				IsOpenAICompatible:        true,
				HasJsonResponseMode:       false,
				HasStreaming:              true,
				HasFunctionCalling:        false,
				HasStreamingFunctionCalls: false,
			},
			BaseUrl: BaseUrlByProvider[ModelProviderTogether],
		},
	},
	{
		Description:                 "Mixtral-8x7B via Together.ai",
		DefaultMaxConvoTokens:       5000,
		DefaultReservedOutputTokens: 4096,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderTogether,
			ModelName:    "mistralai/Mixtral-8x7B-Instruct-v0.1",
			MaxTokens:    32768,
			ApiKeyEnvVar: ApiKeyByProvider[ModelProviderTogether],
			ModelCompatibility: ModelCompatibility{
				IsOpenAICompatible:        true,
				HasJsonResponseMode:       true,
				HasStreaming:              true,
				HasFunctionCalling:        true,
				HasStreamingFunctionCalls: false,
			},
			BaseUrl: BaseUrlByProvider[ModelProviderTogether],
		},
	},
	{
		Description:                 "CodeLLama-34b via Together.ai",
		DefaultMaxConvoTokens:       10000,
		DefaultReservedOutputTokens: 4096,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderTogether,
			ModelName:    "togethercomputer/CodeLlama-34b-Instruct",
			MaxTokens:    16384,
			ApiKeyEnvVar: ApiKeyByProvider[ModelProviderTogether],
			ModelCompatibility: ModelCompatibility{
				IsOpenAICompatible:        true,
				HasJsonResponseMode:       true,
				HasStreaming:              true,
				HasFunctionCalling:        true,
				HasStreamingFunctionCalls: false,
			},
			BaseUrl: BaseUrlByProvider[ModelProviderTogether],
		},
	},
	{
		Description:                 "Google Gemini Pro 1.5 preview via OpenRouter",
		DefaultMaxConvoTokens:       100000,
		DefaultReservedOutputTokens: 22937,
		BaseModelConfig: BaseModelConfig{
			Provider:     ModelProviderOpenRouter,
			ModelName:    "google/gemini-pro-1.5",
			MaxTokens:    2800000000,
			ApiKeyEnvVar: ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility: ModelCompatibility{
				IsOpenAICompatible:        true,
				HasJsonResponseMode:       true,
				HasStreaming:              true,
				HasFunctionCalling:        true,
				HasStreamingFunctionCalls: false,
			},
			BaseUrl: BaseUrlByProvider[ModelProviderOpenRouter],
		},
	},
}

var AvailableModelsByName = map[string]*AvailableModel{}

var Gpt4TurboLatestModelPack ModelPack
var OpenRouterClaude3Dot5SonnetGPT4TurboModelPack ModelPack
var OpenRouterClaude3Dot5SonnetModelPack ModelPack
var TogetherMixtral8x22BModelPack ModelPack
var Gpt4oLatestModelPack ModelPack

var BuiltInModelPacks = []*ModelPack{
	&Gpt4oLatestModelPack,
	&Gpt4TurboLatestModelPack,
	&OpenRouterClaude3Dot5SonnetModelPack,
	&OpenRouterClaude3Dot5SonnetGPT4TurboModelPack,
	&TogetherMixtral8x22BModelPack,
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
		// HasStreamingFunctionCalls: true -- no longer required
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
		// HasStreamingFunctionCalls: true -- no longer required
	},
	ModelRoleAutoFix: {
		IsOpenAICompatible: true,
		HasStreaming:       true,
		HasFunctionCalling: true,
		// HasStreamingFunctionCalls: true -- no longer required
	},
}

func init() {
	for _, model := range AvailableModels {
		AvailableModelsByName[model.ModelName] = model
	}

	Gpt4oLatestModelPack = ModelPack{
		Name:        "gpt-4o-latest",
		Description: "Uses OpenAI's latest gpt-4o model, first released on 2024-05-13, for heavy lifting, and latest version of gpt-3.5-turbo for lighter tasks.",
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
			BaseModelConfig: AvailableModelsByName[openai.GPT3Dot5Turbo].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleName].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleName].TopP,
		},
		CommitMsg: ModelRoleConfig{
			Role:            ModelRoleCommitMsg,
			BaseModelConfig: AvailableModelsByName[openai.GPT3Dot5Turbo].BaseModelConfig,
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

	Gpt4TurboLatestModelPack = ModelPack{
		Name:        "gpt-4-turbo-latest",
		Description: "Uses latest version of OpenAI gpt-4-turbo model (first released on 2024-04-09) for heavy lifting, latest version of gpt-3.5-turbo for lighter tasks.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig: ModelRoleConfig{
				Role:            ModelRolePlanner,
				BaseModelConfig: AvailableModelsByName[openai.GPT4Turbo].BaseModelConfig,
				Temperature:     DefaultConfigByRole[ModelRolePlanner].Temperature,
				TopP:            DefaultConfigByRole[ModelRolePlanner].TopP,
			},
			PlannerModelConfig: getPlannerModelConfig(openai.GPT4Turbo),
		},
		PlanSummary: ModelRoleConfig{
			Role:            ModelRolePlanSummary,
			BaseModelConfig: AvailableModelsByName[openai.GPT4Turbo].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRolePlanSummary].Temperature,
			TopP:            DefaultConfigByRole[ModelRolePlanSummary].TopP,
		},
		Builder: ModelRoleConfig{
			Role:            ModelRoleBuilder,
			BaseModelConfig: AvailableModelsByName[openai.GPT4Turbo].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleBuilder].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleBuilder].TopP,
		},
		Namer: ModelRoleConfig{
			Role:            ModelRoleName,
			BaseModelConfig: AvailableModelsByName[openai.GPT3Dot5Turbo].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleName].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleName].TopP,
		},
		CommitMsg: ModelRoleConfig{
			Role:            ModelRoleCommitMsg,
			BaseModelConfig: AvailableModelsByName[openai.GPT3Dot5Turbo].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleCommitMsg].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleCommitMsg].TopP,
		},
		ExecStatus: ModelRoleConfig{
			Role:            ModelRoleExecStatus,
			BaseModelConfig: AvailableModelsByName[openai.GPT4Turbo].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleExecStatus].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleExecStatus].TopP,
		},
	}

	OpenRouterClaude3Dot5SonnetGPT4TurboModelPack = ModelPack{
		Name:        "anthropic-claude-3.5-sonnet-gpt-4o",
		Description: "Uses Anthropic's Claude 3.5 Sonnet model (via OpenRouter) for planning, summarization, and auto-continue, OpenAI gpt-4o for builds, and gpt-3.5-turbo for lighter tasks.",
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
			BaseModelConfig: AvailableModelsByName[openai.GPT3Dot5Turbo].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleName].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleName].TopP,
		},
		CommitMsg: ModelRoleConfig{
			Role:            ModelRoleCommitMsg,
			BaseModelConfig: AvailableModelsByName[openai.GPT3Dot5Turbo].BaseModelConfig,
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

	TogetherMixtral8x22BModelPack = ModelPack{
		Name:        "Mixtral-8x22b/Mixtral-8x7b/gpt-4o",
		Description: "Uses Together.ai's Mixtral-8x22B for planning and summarization, gpt-4o for builds and auto-continue, and Mixtral-8x7B for lighter tasks.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig: ModelRoleConfig{
				Role:            ModelRolePlanner,
				BaseModelConfig: AvailableModelsByName["mistralai/Mixtral-8x22B-Instruct-v0.1"].BaseModelConfig,
				Temperature:     DefaultConfigByRole[ModelRolePlanner].Temperature,
				TopP:            DefaultConfigByRole[ModelRolePlanner].TopP,
			},
			PlannerModelConfig: getPlannerModelConfig("mistralai/Mixtral-8x22B-Instruct-v0.1"),
		},
		PlanSummary: ModelRoleConfig{
			Role:            ModelRolePlanSummary,
			BaseModelConfig: AvailableModelsByName["mistralai/Mixtral-8x22B-Instruct-v0.1"].BaseModelConfig,
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
			BaseModelConfig: AvailableModelsByName["mistralai/Mixtral-8x7B-Instruct-v0.1"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleName].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleName].TopP,
		},
		CommitMsg: ModelRoleConfig{
			Role:            ModelRoleCommitMsg,
			BaseModelConfig: AvailableModelsByName["mistralai/Mixtral-8x7B-Instruct-v0.1"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleCommitMsg].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleCommitMsg].TopP,
		},
		ExecStatus: ModelRoleConfig{
			Role:            ModelRoleExecStatus,
			BaseModelConfig: AvailableModelsByName[openai.GPT4o].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleExecStatus].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleExecStatus].TopP,
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
