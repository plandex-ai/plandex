package shared

import (
	"github.com/sashabaranov/go-openai"
)

const OpenAIEnvVar = "OPENAI_API_KEY"
const OpenAIV1BaseUrl = "https://api.openai.com/v1"

var fullCompatibility = ModelCompatibility{
	HasImageSupport: true,
}

var AvailableModels = []*AvailableModel{
	{
		Description:                 "OpenAI's latest gpt-4o model",
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
		},
	},
	{
		Description:                 "OpenAI's latest gpt-4o-mini model",
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
		},
	},
	{
		Description:                 "OpenAI's latest o1-mini model",
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
		Description:                 "OpenAI's latest o1 model",
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
	{
		Description:                 "Anthropic Claude 3.5 Sonnet via OpenRouter",
		DefaultMaxConvoTokens:       15000,
		DefaultReservedOutputTokens: 8192,
		BaseModelConfig: BaseModelConfig{
			Provider:                   ModelProviderOpenRouter,
			ModelName:                  "anthropic/claude-3.5-sonnet",
			MaxTokens:                  200000,
			ApiKeyEnvVar:               ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility:         fullCompatibility,
			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat: ModelOutputFormatXml,
		},
	},
	{
		Description:                 "Anthropic Claude 3.5 Haiku via OpenRouter",
		DefaultMaxConvoTokens:       15000,
		DefaultReservedOutputTokens: 8192,
		BaseModelConfig: BaseModelConfig{
			Provider:                   ModelProviderOpenRouter,
			ModelName:                  "anthropic/claude-3.5-haiku",
			MaxTokens:                  200000,
			ApiKeyEnvVar:               ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility:         fullCompatibility,
			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat: ModelOutputFormatXml,
		},
	},
	{
		Description:                 "Google Gemini Pro 1.5 via OpenRouter",
		DefaultMaxConvoTokens:       100000,
		DefaultReservedOutputTokens: 8192,
		BaseModelConfig: BaseModelConfig{
			Provider:                   ModelProviderOpenRouter,
			ModelName:                  "google/gemini-pro-1.5",
			MaxTokens:                  2000000,
			ApiKeyEnvVar:               ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility:         fullCompatibility,
			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat: ModelOutputFormatXml,
		},
	},
	{
		Description:                 "Google Gemini Flash 1.5 via OpenRouter",
		DefaultMaxConvoTokens:       75000,
		DefaultReservedOutputTokens: 8192,
		BaseModelConfig: BaseModelConfig{
			Provider:                   ModelProviderOpenRouter,
			ModelName:                  "google/gemini-flash-1.5",
			MaxTokens:                  1000000,
			ApiKeyEnvVar:               ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility:         fullCompatibility,
			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat: ModelOutputFormatXml,
		},
	},
	{
		Description:                 "Google Gemini Flash 2.0 Experimental via OpenRouter",
		DefaultMaxConvoTokens:       75000,
		DefaultReservedOutputTokens: 8192,
		BaseModelConfig: BaseModelConfig{
			Provider:                   ModelProviderOpenRouter,
			ModelName:                  "google/gemini-2.0-flash-exp:free",
			MaxTokens:                  1000000,
			ApiKeyEnvVar:               ApiKeyByProvider[ModelProviderOpenRouter],
			ModelCompatibility:         fullCompatibility,
			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat: ModelOutputFormatXml,
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
			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat: ModelOutputFormatXml,
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
			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
			PreferredModelOutputFormat: ModelOutputFormatXml,
		},
	},
}

var AvailableModelsByName = map[string]*AvailableModel{}

var OpenRouterClaude3Dot5SonnetGPT4oModelPack ModelPack
var OpenRouterClaude3Dot5SonnetModelPack ModelPack
var Gpt4oLatestModelPack ModelPack
var GoogleGeminiModelPack ModelPack

var BuiltInModelPacks = []*ModelPack{
	&Gpt4oLatestModelPack,
	&OpenRouterClaude3Dot5SonnetModelPack,
	&OpenRouterClaude3Dot5SonnetGPT4oModelPack,
	&GoogleGeminiModelPack,
}

var DefaultModelPack *ModelPack = &OpenRouterClaude3Dot5SonnetGPT4oModelPack

func getPlannerModelConfig(name string) PlannerModelConfig {
	return PlannerModelConfig{
		MaxConvoTokens: AvailableModelsByName[name].DefaultMaxConvoTokens,
	}
}

var DefaultConfigByRole = map[ModelRole]ModelRoleConfig{
	ModelRolePlanner: {
		Temperature: 0.3,
		TopP:        0.3,
	},
	ModelRoleCoder: {
		Temperature: 0.3,
		TopP:        0.3,
	},
	ModelRoleContextLoader: {
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
	ModelRoleWholeFileBuilder: {
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
}

var RequiredCompatibilityByRole = map[ModelRole]ModelCompatibility{
	ModelRolePlanner:     {},
	ModelRolePlanSummary: {},
	ModelRoleBuilder:     {},
	ModelRoleName:        {},
	ModelRoleCommitMsg:   {},
	ModelRoleExecStatus:  {},
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
			BaseModelConfig: AvailableModelsByName["gpt-4o-mini"].BaseModelConfig,
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
	}

	OpenRouterClaude3Dot5SonnetGPT4oModelPack = ModelPack{
		Name:        "plandex-default-pack",
		Description: "Uses Anthropic's Claude 3.5 Sonnet model (via OpenRouter) for planning, coding, and diff-based edits, OpenAI gpt-4o for whole-file edits, and gpt-4o-mini for lighter tasks.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig: ModelRoleConfig{
				Role:            ModelRolePlanner,
				BaseModelConfig: AvailableModelsByName["anthropic/claude-3.5-sonnet"].BaseModelConfig,
				Temperature:     DefaultConfigByRole[ModelRolePlanner].Temperature,
				TopP:            DefaultConfigByRole[ModelRolePlanner].TopP,
			},
			PlannerModelConfig: getPlannerModelConfig("anthropic/claude-3.5-sonnet"),
		},
		// Planner: PlannerRoleConfig{
		// 	ModelRoleConfig: ModelRoleConfig{
		// 		Role:            ModelRolePlanner,
		// 		BaseModelConfig: AvailableModelsByName["google/gemini-pro-1.5"].BaseModelConfig,
		// 		Temperature:     DefaultConfigByRole[ModelRolePlanner].Temperature,
		// 		TopP:            DefaultConfigByRole[ModelRolePlanner].TopP,
		// 	},
		// 	PlannerModelConfig: getPlannerModelConfig("google/gemini-pro-1.5"),
		// },
		Coder: &ModelRoleConfig{
			Role:            ModelRoleCoder,
			BaseModelConfig: AvailableModelsByName["anthropic/claude-3.5-sonnet"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleCoder].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleCoder].TopP,
		},
		ContextLoader: &ModelRoleConfig{
			Role:            ModelRoleContextLoader,
			BaseModelConfig: AvailableModelsByName["anthropic/claude-3.5-sonnet"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleContextLoader].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleContextLoader].TopP,
		},
		PlanSummary: ModelRoleConfig{
			Role:            ModelRolePlanSummary,
			BaseModelConfig: AvailableModelsByName["gpt-4o-mini"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRolePlanSummary].Temperature,
			TopP:            DefaultConfigByRole[ModelRolePlanSummary].TopP,
		},
		Builder: ModelRoleConfig{
			Role:            ModelRoleBuilder,
			BaseModelConfig: AvailableModelsByName["anthropic/claude-3.5-sonnet"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleBuilder].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleBuilder].TopP,
		},
		WholeFileBuilder: &ModelRoleConfig{
			Role:            ModelRoleWholeFileBuilder,
			BaseModelConfig: AvailableModelsByName[openai.GPT4o].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleWholeFileBuilder].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleWholeFileBuilder].TopP,
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
			BaseModelConfig: AvailableModelsByName["anthropic/claude-3.5-haiku"].BaseModelConfig,
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
			BaseModelConfig: AvailableModelsByName["anthropic/claude-3.5-haiku"].BaseModelConfig,
			Temperature:     DefaultConfigByRole[ModelRoleName].Temperature,
			TopP:            DefaultConfigByRole[ModelRoleName].TopP,
		},
		CommitMsg: ModelRoleConfig{
			Role:            ModelRoleCommitMsg,
			BaseModelConfig: AvailableModelsByName["anthropic/claude-3.5-haiku"].BaseModelConfig,
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

}

func FilterCompatibleModels(models []*AvailableModel, role ModelRole) []*AvailableModel {
	// required := RequiredCompatibilityByRole[role]
	var compatibleModels []*AvailableModel

	for _, model := range models {
		// no compatibility checks are needed in v2, but keeping this here in case compatibility checks are needed in the future

		compatibleModels = append(compatibleModels, model)
	}

	return compatibleModels
}
