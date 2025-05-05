package shared

import (
	"github.com/davecgh/go-spew/spew"
)

/*
'MaxTokens' is the absolute input limit for the provider.

'MaxOutputTokens' is the absolute output limit for the provider.

'ReservedOutputTokens' is how much we set aside in context for the model to use in its output. It's more of a realistic output limit, since for some models, the hard maximum 'MaxTokens' is actually equal to the input limit, which would leave no room for input.

The effective input limit is 'MaxTokens' - 'ReservedOutputTokens'.

For example, OpenAI o3-mini has a MaxTokens of 200k and a MaxOutputTokens of 100k. But in practice, we are very unlikely to use all the output tokens, and we want to leave more space for input. So we set ReservedOutputTokens to 40k, allowing ~25k for reasoning tokens, as well as ~15k for real output tokens, which is enough for most use cases. The new effective input limit is therefore 200k - 40k = 160k. However, these are not passed through as hard limits. So if we have a smaller amount of input (under 100k) the model could still use up to the full 100k output tokens if necessary.

For models with a low output limit, we just set ReservedOutputTokens to the MaxOutputTokens.

When checking for sufficient credits on Plandex Cloud, we use MaxOutputTokens-InputTokens, since this is the maximum that could hypothetically be used.

'DefaultMaxConvoTokens' is the default maximum number of conversation tokens that are allowed before we start using gradual summarization to shorten the conversation.

'ModelName' is the name of the model on the provider's side.

'ModelId' is the identifier for the model on the Plandex side—it must be unique per provider. We have this so that models with the same name and provider, but different settings can be differentiated.

'ModelCompatibility' is used to check for feature support (like image support).

'BaseUrl' is the base URL for the provider.

'PreferredModelOutputFormat' is the preferred output format for the model—currently either 'ModelOutputFormatToolCallJson' or 'ModelOutputFormatXml' — OpenAI models like JSON (and benefit from strict JSON schemas), while most other providers are unreliable for JSON generation and do better with XML, even if they claim to support JSON.

'RoleParamsDisabled' is used to disable role-based parameters like temperature, top_p, etc. for the model—OpenAI early releases often don't allow changes to these.

'SystemPromptDisabled' is used to disable the system prompt for the model—OpenAI early releases sometimes don't allow system prompts.

'ReasoningEffortEnabled' is used to enable reasoning effort for the model (like OpenAI's o3-mini).

'ReasoningEffort' is the reasoning effort for the model, when 'ReasoningEffortEnabled' is true.

'PredictedOutputEnabled' is used to enable predicted output for the model (currently only supported by gpt-4o).

'ApiKeyEnvVar' is the environment variable that contains the API key for the model.
*/

var BuiltInModels = []*BaseModelConfigSchema{
	{
		ModelTag:              "openai/o3",
		Description:           "OpenAI o3",
		DefaultMaxConvoTokens: 15000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  200000,
			MaxOutputTokens:            100000,
			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
			ModelCompatibility:         fullCompatibility,
			PreferredModelOutputFormat: ModelOutputFormatXml,
			SystemPromptDisabled:       true,
			RoleParamsDisabled:         true,
			ReasoningEffortEnabled:     true,
			ReasoningEffort:            ReasoningEffortHigh,
			StopDisabled:               true,
		},

		Variants: []BaseModelConfigVariant{
			{
				VariantTag:  "high",
				Description: "high",
			},
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "o3",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "o3",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/o3",
			},
		},
	},
	{
		PriceId:               "openai/o3",
		ModelTag:              "openai/o3-medium",
		Description:           "OpenAI o3-medium",
		DefaultMaxConvoTokens: 15000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  200000,
			MaxOutputTokens:            100000,
			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
			ModelCompatibility:         fullCompatibility,
			PreferredModelOutputFormat: ModelOutputFormatXml,
			SystemPromptDisabled:       true,
			RoleParamsDisabled:         true,
			ReasoningEffortEnabled:     true,
			ReasoningEffort:            ReasoningEffortMedium,
			StopDisabled:               true,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "o3",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "o3",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/o3",
			},
		},
	},
	{
		PriceId:               "openai/o3",
		ModelTag:              "openai/o3-low",
		Description:           "OpenAI o3-low",
		DefaultMaxConvoTokens: 15000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  200000,
			MaxOutputTokens:            100000,
			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
			ModelCompatibility:         fullCompatibility,
			PreferredModelOutputFormat: ModelOutputFormatXml,
			SystemPromptDisabled:       true,
			RoleParamsDisabled:         true,
			ReasoningEffortEnabled:     true,
			ReasoningEffort:            ReasoningEffortLow,
			StopDisabled:               true,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "o3",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "o3",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/o3",
			},
		},
	},

	{
		PriceId:               "openai/o4-mini",
		ModelTag:              "openai/o4-mini-high",
		Description:           "OpenAI o4-mini-high",
		DefaultMaxConvoTokens: 10000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  200000,
			MaxOutputTokens:            100000,
			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
			ModelCompatibility:         fullCompatibility,
			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
			SystemPromptDisabled:       true,
			RoleParamsDisabled:         true,
			ReasoningEffortEnabled:     true,
			ReasoningEffort:            ReasoningEffortHigh,
			StopDisabled:               true,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "o4-mini",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "o4-mini",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/o4-mini",
			},
		},
	},

	{
		PriceId:               "openai/o4-mini",
		ModelTag:              "openai/o4-mini-medium",
		Description:           "OpenAI o4-mini-medium",
		DefaultMaxConvoTokens: 10000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  200000,
			MaxOutputTokens:            100000,
			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
			ModelCompatibility:         fullCompatibility,
			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
			SystemPromptDisabled:       true,
			RoleParamsDisabled:         true,
			ReasoningEffortEnabled:     true,
			ReasoningEffort:            ReasoningEffortMedium,
			StopDisabled:               true,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "o4-mini",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "o4-mini",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/o4-mini",
			},
		},
	},

	{
		PriceId:               "openai/o4-mini",
		ModelTag:              "openai/o4-mini-low",
		Description:           "OpenAI o4-mini-low",
		DefaultMaxConvoTokens: 10000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  200000,
			MaxOutputTokens:            100000,
			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
			ModelCompatibility:         fullCompatibility,
			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
			SystemPromptDisabled:       true,
			RoleParamsDisabled:         true,
			ReasoningEffortEnabled:     true,
			ReasoningEffort:            ReasoningEffortLow,
			StopDisabled:               true,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "o4-mini",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "o4-mini",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/o4-mini",
			},
		},
	},

	{
		PriceId:               "openai/gpt-4.1",
		ModelTag:              "openai/gpt-4.1",
		Description:           "OpenAI gpt-4.1",
		DefaultMaxConvoTokens: 75000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  1047576,
			MaxOutputTokens:            32768,
			ReservedOutputTokens:       32768,
			ModelCompatibility:         fullCompatibility,
			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "gpt-4.1",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "gpt-4.1",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/gpt-4.1",
			},
		},
	},

	{
		PriceId:               "openai/gpt-4.1-mini",
		ModelTag:              "openai/gpt-4.1-mini",
		Description:           "OpenAI gpt-4.1-mini",
		DefaultMaxConvoTokens: 75000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  1047576,
			MaxOutputTokens:            32768,
			ReservedOutputTokens:       32768,
			ModelCompatibility:         fullCompatibility,
			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "gpt-4.1-mini",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "gpt-4.1-mini",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/gpt-4.1-mini",
			},
		},
	},

	{
		PriceId:               "openai/gpt-4.1-nano",
		ModelTag:              "openai/gpt-4.1-nano",
		Description:           "OpenAI gpt-4.1-nano",
		DefaultMaxConvoTokens: 75000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  1047576,
			MaxOutputTokens:            32768,
			ReservedOutputTokens:       32768,
			ModelCompatibility:         fullCompatibility,
			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "gpt-4.1-nano",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "gpt-4.1-nano",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/gpt-4.1-nano",
			},
		},
	},

	{
		PriceId:               "openai/o3-mini",
		ModelTag:              "openai/o3-mini-high",
		Description:           "OpenAI o3-mini-high",
		DefaultMaxConvoTokens: 10000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  200000,
			MaxOutputTokens:            100000,
			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
			ModelCompatibility:         fullCompatibility,
			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
			SystemPromptDisabled:       true,
			RoleParamsDisabled:         true,
			ReasoningEffortEnabled:     true,
			ReasoningEffort:            ReasoningEffortHigh,
			StopDisabled:               true,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "o3-mini",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "o3-mini",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/o3-mini",
			},
		},
	},

	{
		PriceId:               "openai/o3-mini",
		ModelTag:              "openai/o3-mini-medium",
		Description:           "OpenAI o3-mini-medium",
		DefaultMaxConvoTokens: 10000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  200000,
			MaxOutputTokens:            100000,
			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
			ModelCompatibility:         fullCompatibility,
			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
			SystemPromptDisabled:       true,
			RoleParamsDisabled:         true,
			ReasoningEffortEnabled:     true,
			ReasoningEffort:            ReasoningEffortMedium,
			StopDisabled:               true,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "o3-mini",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "o3-mini",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/o3-mini",
			},
		},
	},

	{
		PriceId:               "openai/o3-mini",
		ModelTag:              "openai/o3-mini-low",
		Description:           "OpenAI o3-mini-low",
		DefaultMaxConvoTokens: 10000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  200000,
			MaxOutputTokens:            100000,
			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
			ModelCompatibility:         fullCompatibility,
			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
			SystemPromptDisabled:       true,
			RoleParamsDisabled:         true,
			ReasoningEffortEnabled:     true,
			ReasoningEffort:            ReasoningEffortLow,
			StopDisabled:               true,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "o3-mini",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "o3-mini",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/o3-mini",
			},
		},
	},

	{
		PriceId:               "anthropic/claude-3.7-sonnet",
		ModelTag:              "anthropic/claude-3.7-sonnet",
		Description:           "Anthropic Claude 3.7 Sonnet",
		DefaultMaxConvoTokens: 15000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                   200000,
			MaxOutputTokens:             128000,
			ReservedOutputTokens:        20000,
			SupportsCacheControl:        true,
			PreferredModelOutputFormat:  ModelOutputFormatXml,
			SingleMessageNoSystemPrompt: true,
			TokenEstimatePaddingPct:     0.10,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderAnthropic,
				ModelName: "claude-3.7-sonnet",
			},
			{
				Provider:  ModelProviderGoogleVertex,
				ModelName: "claude-3-7-sonnet@20250219",
			},
			{
				Provider:  ModelProviderAmazonBedrock,
				ModelName: "anthropic.claude-3-7-sonnet-20250219-v1:0",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "anthropic/claude-3.7-sonnet",
			},
		},
	},

	{
		PriceId:               "anthropic/claude-3.7-sonnet",
		ModelTag:              "anthropic/claude-3.7-sonnet:thinking-hidden",
		Description:           "Anthropic Claude 3.7 Sonnet (thinking—reasoning hidden) ",
		DefaultMaxConvoTokens: 15000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                   200000,
			MaxOutputTokens:             128000,
			ReservedOutputTokens:        20000,
			SupportsCacheControl:        true,
			PreferredModelOutputFormat:  ModelOutputFormatXml,
			SingleMessageNoSystemPrompt: true,
			TokenEstimatePaddingPct:     0.10,
			IncludeReasoning:            false,
			ReasoningBudget:             AnthropicMaxReasoningBudget,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderAnthropic,
				ModelName: "claude-3.7-sonnet",
			},
			{
				Provider:  ModelProviderGoogleVertex,
				ModelName: "claude-3-7-sonnet@20250219",
			},
			{
				Provider:  ModelProviderAmazonBedrock,
				ModelName: "anthropic.claude-3-7-sonnet-20250219-v1:0",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "anthropic/claude-3.7-sonnet",
			},
		},
	},

	{
		PriceId:               "anthropic/claude-3.7-sonnet",
		ModelTag:              "anthropic/claude-3.7-sonnet:thinking-visible",
		Description:           "Anthropic Claude 3.7 Sonnet (thinking—reasoning visible) ",
		DefaultMaxConvoTokens: 15000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                   200000,
			MaxOutputTokens:             128000,
			ReservedOutputTokens:        20000,
			SupportsCacheControl:        true,
			PreferredModelOutputFormat:  ModelOutputFormatXml,
			SingleMessageNoSystemPrompt: true,
			TokenEstimatePaddingPct:     0.10,
			IncludeReasoning:            true,
			ReasoningBudget:             AnthropicMaxReasoningBudget,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderAnthropic,
				ModelName: "claude-3.7-sonnet",
			},
			{
				Provider:  ModelProviderGoogleVertex,
				ModelName: "claude-3-7-sonnet@20250219",
			},
			{
				Provider:  ModelProviderAmazonBedrock,
				ModelName: "anthropic.claude-3-7-sonnet-20250219-v1:0",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "anthropic/claude-3.7-sonnet",
			},
		},
	},

	{
		PriceId:               "anthropic/claude-3.5-sonnet",
		ModelTag:              "anthropic/claude-3.5-sonnet",
		Description:           "Anthropic Claude 3.5 Sonnet",
		DefaultMaxConvoTokens: 15000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                   200000,
			MaxOutputTokens:             128000,
			ReservedOutputTokens:        20000,
			SupportsCacheControl:        true,
			PreferredModelOutputFormat:  ModelOutputFormatXml,
			SingleMessageNoSystemPrompt: true,
			TokenEstimatePaddingPct:     0.10,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderAnthropic,
				ModelName: "claude-3.5-sonnet",
			},
			{
				Provider:  ModelProviderGoogleVertex,
				ModelName: "claude-3-5-sonnet@20240620",
			},
			{
				Provider:  ModelProviderAmazonBedrock,
				ModelName: "anthropic.claude-3-5-sonnet-20241022-v2:0",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "anthropic/claude-3.5-sonnet",
			},
		},
	},

	{
		PriceId:               "anthropic/claude-3.5-haiku",
		ModelTag:              "anthropic/claude-3.5-haiku",
		Description:           "Anthropic Claude 3.5 Haiku",
		DefaultMaxConvoTokens: 15000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                   200000,
			MaxOutputTokens:             8192,
			ReservedOutputTokens:        8192,
			SupportsCacheControl:        true,
			PreferredModelOutputFormat:  ModelOutputFormatXml,
			SingleMessageNoSystemPrompt: true,
			TokenEstimatePaddingPct:     0.10,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderAnthropic,
				ModelName: "claude-3.5-haiku",
			},
			{
				Provider:  ModelProviderGoogleVertex,
				ModelName: "claude-3-5-haiku@20241022",
			},
			{
				Provider:  ModelProviderAmazonBedrock,
				ModelName: "anthropic.claude-3-5-haiku-20241022-v1:0",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "anthropic/claude-3.5-haiku",
			},
		},
	},

	{
		PriceId:               "google/gemini-pro-1.5",
		ModelTag:              "google/gemini-pro-1.5",
		Description:           "Google Gemini 1.5 Pro",
		DefaultMaxConvoTokens: 75000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  2000000,
			MaxOutputTokens:            8192,
			ReservedOutputTokens:       8192,
			SupportsCacheControl:       true,
			PreferredModelOutputFormat: ModelOutputFormatXml,
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderGoogleAIStudio, ModelName: "models/gemini-1.5-pro"},
			{Provider: ModelProviderGoogleVertex, ModelName: "gemini-1.5-pro-latest"},
			{Provider: ModelProviderOpenRouter, ModelName: "google/gemini-pro-1.5"},
		},
	},
	{
		PriceId:               "google/gemini-2.5-pro-preview",
		ModelTag:              "google/gemini-2.5-pro-preview",
		Description:           "Google Gemini 2.5 Pro (Preview)",
		DefaultMaxConvoTokens: 75000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  1048576,
			MaxOutputTokens:            65535,
			ReservedOutputTokens:       65535,
			SupportsCacheControl:       true,
			PreferredModelOutputFormat: ModelOutputFormatXml,
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderGoogleAIStudio, ModelName: "models/gemini-2.5-pro-preview-03-25"},
			{Provider: ModelProviderGoogleVertex, ModelName: "gemini-2.5-pro-preview-03-25"},
			{Provider: ModelProviderOpenRouter, ModelName: "google/gemini-2.5-pro-preview-03-25"},
		},
	},
	{
		PriceId:               "google/gemini-2.5-pro-exp",
		ModelTag:              "google/gemini-2.5-pro-exp",
		Description:           "Google Gemini 2.5 Pro (Experimental)",
		DefaultMaxConvoTokens: 75000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  1048576,
			MaxOutputTokens:            65535,
			ReservedOutputTokens:       65535,
			PreferredModelOutputFormat: ModelOutputFormatXml,
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderGoogleAIStudio, ModelName: "models/gemini-2.5-pro-exp-03-25"},
			{Provider: ModelProviderGoogleVertex, ModelName: "gemini-2.5-pro-exp-03-25"},
			{Provider: ModelProviderOpenRouter, ModelName: "google/gemini-2.5-pro-exp-03-25"},
		},
	},
	{
		PriceId:               "google/gemini-2.5-flash-preview",
		ModelTag:              "google/gemini-2.5-flash-preview",
		Description:           "Google Gemini 2.5 Flash (Preview)",
		DefaultMaxConvoTokens: 75000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  1048576,
			MaxOutputTokens:            65535,
			ReservedOutputTokens:       65535,
			SupportsCacheControl:       true,
			PreferredModelOutputFormat: ModelOutputFormatXml,
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderGoogleAIStudio, ModelName: "models/gemini-2.5-flash-preview-04-17"},
			{Provider: ModelProviderGoogleVertex, ModelName: "gemini-2.5-flash-preview-04-17"},
			{Provider: ModelProviderOpenRouter, ModelName: "google/gemini-2.5-flash-preview"},
		},
	},

	{
		PriceId:               "deepseek/v3-0324",
		ModelTag:              "deepseek/v3-0324",
		Description:           "DeepSeek V3 (0324)",
		DefaultMaxConvoTokens: 7500,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  64000,
			MaxOutputTokens:            8192,
			ReservedOutputTokens:       8192,
			PreferredModelOutputFormat: ModelOutputFormatXml,
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderDeepSeek, ModelName: "deepseek-chat"},
			{Provider: ModelProviderOpenRouter, ModelName: "deepseek/deepseek-chat-v3-0324"},
		},
	},
	{
		PriceId:               "deepseek/r1",
		ModelTag:              "deepseek/r1",
		Description:           "DeepSeek R1 (includes reasoning)",
		DefaultMaxConvoTokens: 7500,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  64000,
			MaxOutputTokens:            8192,
			ReservedOutputTokens:       8192,
			PreferredModelOutputFormat: ModelOutputFormatXml,
			IncludeReasoning:           true,
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderDeepSeek, ModelName: "deepseek-reasoner"},
			{Provider: ModelProviderOpenRouter, ModelName: "deepseek/deepseek-r1"},
		},
	},
	{
		PriceId:               "deepseek/r1",
		ModelTag:              "deepseek/r1:reasoning-hidden",
		Description:           "DeepSeek R1 (reasoning hidden)",
		DefaultMaxConvoTokens: 7500,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  64000,
			MaxOutputTokens:            8192,
			ReservedOutputTokens:       8192,
			PreferredModelOutputFormat: ModelOutputFormatXml,
			IncludeReasoning:           false,
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderDeepSeek, ModelName: "deepseek-reasoner"},
			{Provider: ModelProviderOpenRouter, ModelName: "deepseek/deepseek-r1"},
		},
	},

	{
		PriceId:               "perplexity/r1-1776",
		ModelTag:              "perplexity/r1-1776",
		Description:           "Perplexity R1-1776 (includes reasoning)",
		DefaultMaxConvoTokens: 7500,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  128000,
			MaxOutputTokens:            128000,
			ReservedOutputTokens:       30000,
			PreferredModelOutputFormat: ModelOutputFormatXml,
			IncludeReasoning:           true,
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderPerplexity, ModelName: "r1-1776-online"},
			{Provider: ModelProviderOpenRouter, ModelName: "perplexity/r1-1776"},
		},
	},
	{
		PriceId:               "perplexity/sonar-reasoning",
		ModelTag:              "perplexity/sonar-reasoning",
		Description:           "Perplexity Sonar Reasoning (includes reasoning)",
		DefaultMaxConvoTokens: 7500,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  127000,
			MaxOutputTokens:            127000,
			ReservedOutputTokens:       30000,
			PreferredModelOutputFormat: ModelOutputFormatXml,
			IncludeReasoning:           true,
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderPerplexity, ModelName: "sonar-reasoning-online"},
			{Provider: ModelProviderOpenRouter, ModelName: "perplexity/sonar-reasoning"},
		},
	},

	{
		PriceId:               "qwen/qwen-2.5-coder-32b-instruct",
		ModelTag:              "qwen/qwen-2.5-coder-32b-instruct",
		Description:           "Qwen 2.5 Coder 32B (Instruct)",
		DefaultMaxConvoTokens: 10000,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  128000,
			MaxOutputTokens:            8192,
			ReservedOutputTokens:       8192,
			PreferredModelOutputFormat: ModelOutputFormatXml,
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderOpenRouter, ModelName: "qwen/qwen-2.5-coder-32b-instruct"},
		},
	},

	{
		PriceId:               "qwen/qwen3-32b",
		ModelTag:              "qwen/qwen3-32b",
		Description:           "Qwen 3-32B (Experimental)",
		DefaultMaxConvoTokens: 15000, // feel free to raise once you test

		BaseModelShared: BaseModelShared{
			MaxTokens:                  128000,
			MaxOutputTokens:            16384,
			ReservedOutputTokens:       16384,
			PreferredModelOutputFormat: ModelOutputFormatXml,
			// leave IncludeReasoning unset – Qwen3 has its own “thinking” mode but OR
			// exposes it via a special prompt key, not a flag
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderOpenRouter, ModelName: "qwen/qwen3-32b-a3b"}, // OR cloud
			// optional local provider once you wire up Ollama or HF Inference
			// {Provider: ModelProviderOllama,    ModelName: "qwen3-32b"},
		},
	},
	{
		PriceId:               "qwen/qwen3-8b",
		ModelTag:              "qwen/qwen3-8b",
		Description:           "Qwen 3-8B (Experimental)",
		DefaultMaxConvoTokens: 7500,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  32768,
			MaxOutputTokens:            8192,
			ReservedOutputTokens:       8192,
			PreferredModelOutputFormat: ModelOutputFormatXml,
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderOpenRouter, ModelName: "qwen/qwen3-8b-04-28"},
		},
	},
}

var AvailableModels = []*AvailableModel{}

// var AvailableModels = []*AvailableModel{
// 	// Direct OpenAI models
// 	{
// 		Description:           "OpenAI o3-high",
// 		DefaultMaxConvoTokens: 15000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenAI,
// 			ModelName:                  "o3",
// 			ModelId:                    "openai/o3-high",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
// 			ApiKeyEnvVar:               OpenAIEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    OpenAIV1BaseUrl,
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 			SystemPromptDisabled:       true,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortHigh,
// 			StopDisabled:               true,
// 		},
// 	},

// 	{
// 		Description:           "OpenAI o3-medium",
// 		DefaultMaxConvoTokens: 15000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenAI,
// 			ModelName:                  "o3",
// 			ModelId:                    "openai/o3-medium",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
// 			ApiKeyEnvVar:               OpenAIEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    OpenAIV1BaseUrl,
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 			SystemPromptDisabled:       true,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortMedium,
// 			StopDisabled:               true,
// 		},
// 	},

// 	{
// 		Description:           "OpenAI o3-low",
// 		DefaultMaxConvoTokens: 15000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenAI,
// 			ModelName:                  "o3",
// 			ModelId:                    "openai/o3-low",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
// 			ApiKeyEnvVar:               OpenAIEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    OpenAIV1BaseUrl,
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 			SystemPromptDisabled:       true,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortLow,
// 			StopDisabled:               true,
// 		},
// 	},

// 	{
// 		Description:           "OpenAI o4-mini-high",
// 		DefaultMaxConvoTokens: 10000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenAI,
// 			ModelName:                  "o4-mini",
// 			ModelId:                    "openai/o4-mini-high",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       30000,
// 			ApiKeyEnvVar:               OpenAIEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    OpenAIV1BaseUrl,
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortHigh,
// 			StopDisabled:               true,
// 		},
// 	},
// 	{
// 		Description:           "OpenAI o4-mini-medium",
// 		DefaultMaxConvoTokens: 10000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenAI,
// 			ModelName:                  "o4-mini",
// 			ModelId:                    "openai/o4-mini-medium",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
// 			ApiKeyEnvVar:               OpenAIEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    OpenAIV1BaseUrl,
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortMedium,
// 			StopDisabled:               true,
// 		},
// 	},
// 	{
// 		Description:           "OpenAI o4-mini-low",
// 		DefaultMaxConvoTokens: 10000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenAI,
// 			ModelName:                  "o4-mini",
// 			ModelId:                    "openai/o4-mini-low",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
// 			ApiKeyEnvVar:               OpenAIEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    OpenAIV1BaseUrl,
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortLow,
// 			StopDisabled:               true,
// 		},
// 	},

// 	{
// 		Description:           "OpenAI gpt-4.1",
// 		DefaultMaxConvoTokens: 75000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenAI,
// 			ModelName:                  "gpt-4.1",
// 			ModelId:                    "openai/gpt-4.1",
// 			MaxTokens:                  1047576,
// 			MaxOutputTokens:            32768,
// 			ReservedOutputTokens:       32768,
// 			ApiKeyEnvVar:               OpenAIEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    OpenAIV1BaseUrl,
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 		},
// 	},

// 	{
// 		Description:           "OpenAI gpt-4.1-mini",
// 		DefaultMaxConvoTokens: 75000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenAI,
// 			ModelName:                  "gpt-4.1-mini",
// 			ModelId:                    "openai/gpt-4.1-mini",
// 			MaxTokens:                  1047576,
// 			MaxOutputTokens:            32768,
// 			ReservedOutputTokens:       32768,
// 			ApiKeyEnvVar:               OpenAIEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    OpenAIV1BaseUrl,
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 		},
// 	},

// 	{
// 		Description:           "OpenAI gpt-4.1-nano",
// 		DefaultMaxConvoTokens: 75000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenAI,
// 			ModelName:                  "gpt-4.1-nano",
// 			ModelId:                    "openai/gpt-4.1-nano",
// 			MaxTokens:                  1047576,
// 			MaxOutputTokens:            32768,
// 			ReservedOutputTokens:       32768,
// 			ApiKeyEnvVar:               OpenAIEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    OpenAIV1BaseUrl,
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 		},
// 	},

// 	{
// 		Description:           "OpenAI o3-mini-high",
// 		DefaultMaxConvoTokens: 10000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenAI,
// 			ModelName:                  "o3-mini",
// 			ModelId:                    "openai/o3-mini-high",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       30000,
// 			ApiKeyEnvVar:               OpenAIEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    OpenAIV1BaseUrl,
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortHigh,
// 		},
// 	},
// 	{
// 		Description:           "OpenAI o3-mini-medium",
// 		DefaultMaxConvoTokens: 10000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenAI,
// 			ModelName:                  "o3-mini",
// 			ModelId:                    "openai/o3-mini-medium",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
// 			ApiKeyEnvVar:               OpenAIEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    OpenAIV1BaseUrl,
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortMedium,
// 		},
// 	},
// 	{
// 		Description:           "OpenAI o3-mini-low",
// 		DefaultMaxConvoTokens: 10000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenAI,
// 			ModelName:                  "o3-mini",
// 			ModelId:                    "openai/o3-mini-low",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
// 			ApiKeyEnvVar:               OpenAIEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    OpenAIV1BaseUrl,
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortLow,
// 		},
// 	},

// 	// OpenRouter models
// 	{
// 		Description:           "Anthropic Claude 3.7 Sonnet via OpenRouter",
// 		DefaultMaxConvoTokens: 15000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                    ModelProviderOpenRouter,
// 			ModelName:                   "anthropic/claude-3.7-sonnet",
// 			ModelId:                     "anthropic/claude-3.7-sonnet",
// 			MaxTokens:                   200000,
// 			MaxOutputTokens:             128000,
// 			ReservedOutputTokens:        20000,
// 			SupportsCacheControl:        true,
// 			ApiKeyEnvVar:                ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility:          fullCompatibility,
// 			BaseUrl:                     BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat:  ModelOutputFormatXml,
// 			SingleMessageNoSystemPrompt: true,
// 			TokenEstimatePaddingPct:     0.10,
// 		},
// 	},
// 	{
// 		Description:           "Anthropic Claude 3.7 Sonnet (thinking—includes reasoning) via OpenRouter",
// 		DefaultMaxConvoTokens: 15000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                    ModelProviderOpenRouter,
// 			ModelName:                   "anthropic/claude-3.7-sonnet:thinking",
// 			ModelId:                     "anthropic/claude-3.7-sonnet:thinking",
// 			MaxTokens:                   200000,
// 			MaxOutputTokens:             128000,
// 			ReservedOutputTokens:        40000,
// 			SupportsCacheControl:        true,
// 			ApiKeyEnvVar:                ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility:          fullCompatibility,
// 			BaseUrl:                     BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat:  ModelOutputFormatXml,
// 			IncludeReasoning:            true,
// 			SingleMessageNoSystemPrompt: true,
// 			TokenEstimatePaddingPct:     0.10,
// 		},
// 	},
// 	{
// 		Description:           "Anthropic Claude 3.7 Sonnet (thinking—reasoning hidden) via OpenRouter",
// 		DefaultMaxConvoTokens: 15000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                    ModelProviderOpenRouter,
// 			ModelName:                   "anthropic/claude-3.7-sonnet:thinking",
// 			ModelId:                     "anthropic/claude-3.7-sonnet:thinking-hidden",
// 			MaxTokens:                   200000,
// 			MaxOutputTokens:             128000,
// 			ReservedOutputTokens:        40000,
// 			SupportsCacheControl:        true,
// 			ApiKeyEnvVar:                ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility:          fullCompatibility,
// 			BaseUrl:                     BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat:  ModelOutputFormatXml,
// 			IncludeReasoning:            false,
// 			SingleMessageNoSystemPrompt: true,
// 			TokenEstimatePaddingPct:     0.10,
// 		},
// 	},
// 	{
// 		Description:           "Anthropic Claude 3.7 Sonnet",
// 		DefaultMaxConvoTokens: 15000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderAnthropic,
// 			ModelName:                  "claude-3-7-sonnet-latest",
// 			ModelId:                    "claude-3-7-sonnet-latest",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            128000,
// 			ReservedOutputTokens:       20000,
// 			SupportsCacheControl:       true,
// 			ApiKeyEnvVar:               ApiKeyByProvider[ModelProviderAnthropic],
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderAnthropic],
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 		},
// 	},
// 	{
// 		Description:           "Anthropic Claude 3.5 Sonnet via OpenRouter",
// 		DefaultMaxConvoTokens: 15000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                    ModelProviderOpenRouter,
// 			ModelName:                   "anthropic/claude-3.5-sonnet",
// 			ModelId:                     "anthropic/claude-3.5-sonnet",
// 			MaxTokens:                   200000,
// 			MaxOutputTokens:             128000,
// 			ReservedOutputTokens:        20000,
// 			SupportsCacheControl:        true,
// 			ApiKeyEnvVar:                ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility:          fullCompatibility,
// 			BaseUrl:                     BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat:  ModelOutputFormatXml,
// 			SingleMessageNoSystemPrompt: true,
// 			TokenEstimatePaddingPct:     0.10,
// 		},
// 	},
// 	{
// 		Description:           "Anthropic Claude 3.5 Haiku via OpenRouter",
// 		DefaultMaxConvoTokens: 15000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                    ModelProviderOpenRouter,
// 			ModelName:                   "anthropic/claude-3.5-haiku",
// 			ModelId:                     "anthropic/claude-3.5-haiku",
// 			MaxTokens:                   200000,
// 			MaxOutputTokens:             8192,
// 			ReservedOutputTokens:        8192,
// 			SupportsCacheControl:        true,
// 			ApiKeyEnvVar:                ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility:          fullCompatibility,
// 			BaseUrl:                     BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat:  ModelOutputFormatXml,
// 			SingleMessageNoSystemPrompt: true,
// 			TokenEstimatePaddingPct:     0.10,
// 		},
// 	},
// 	{
// 		Description:           "Google Gemini Pro 1.5 via OpenRouter",
// 		DefaultMaxConvoTokens: 100000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "google/gemini-pro-1.5",
// 			ModelId:                    "google/gemini-pro-1.5",
// 			MaxTokens:                  2000000,
// 			MaxOutputTokens:            8192,
// 			ReservedOutputTokens:       8192,
// 			ApiKeyEnvVar:               ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 			SupportsCacheControl:       true,
// 		},
// 	},

// 	{
// 		Description:           "Google Gemini Pro 2.5 Preview via OpenRouter",
// 		DefaultMaxConvoTokens: 75000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "google/gemini-2.5-pro-preview-03-25",
// 			ModelId:                    "google/gemini-2.5-pro-preview-03-25",
// 			MaxTokens:                  1048576,
// 			MaxOutputTokens:            65535,
// 			ReservedOutputTokens:       65535,
// 			ApiKeyEnvVar:               ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 			SupportsCacheControl:       true,
// 		},
// 	},

// 	{
// 		Description:           "Google Gemini Pro 2.5 Experimental via OpenRouter",
// 		DefaultMaxConvoTokens: 75000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "google/gemini-2.5-pro-exp-03-25",
// 			ModelId:                    "google/gemini-2.5-pro-exp-03-25",
// 			MaxTokens:                  1000000,
// 			MaxOutputTokens:            65535,
// 			ReservedOutputTokens:       65535,
// 			ApiKeyEnvVar:               ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 		},
// 	},

// 	{
// 		Description:           "Google Gemini Pro 1.5 Pro via AI Studio",
// 		DefaultMaxConvoTokens: 75000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderGoogleAIStudio,
// 			ModelName:                  "models/gemini-1.5-pro",
// 			ModelId:                    "models/gemini-1.5-pro",
// 			MaxTokens:                  2000000,
// 			MaxOutputTokens:            8192,
// 			ReservedOutputTokens:       8192,
// 			ApiKeyEnvVar:               GoogleAIStudioApiKeyEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderGoogleAIStudio],
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 		},
// 	},

// 	{
// 		Description:           "Google Gemini Flash 2.5 Preview via OpenRouter",
// 		DefaultMaxConvoTokens: 75000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "google/gemini-2.5-flash-preview",
// 			ModelId:                    "google/gemini-2.5-flash-preview",
// 			MaxTokens:                  1048576,
// 			MaxOutputTokens:            65535,
// 			ReservedOutputTokens:       65535,
// 			ApiKeyEnvVar:               ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 			SupportsCacheControl:       true,
// 		},
// 	},

// 	{
// 		Description:           "DeepSeek V3 0324 via OpenRouter",
// 		DefaultMaxConvoTokens: 7500,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:             ModelProviderOpenRouter,
// 			ModelName:            "deepseek/deepseek-chat-v3-0324",
// 			ModelId:              "deepseek/deepseek-chat-v3-0324",
// 			MaxTokens:            64000,
// 			MaxOutputTokens:      8192,
// 			ReservedOutputTokens: 8192,
// 			ApiKeyEnvVar:         ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility: ModelCompatibility{
// 				HasImageSupport: false,
// 			},
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 		},
// 	},

// 	{
// 		Description:           "DeepSeek R1 via OpenRouter (includes reasoning)",
// 		DefaultMaxConvoTokens: 7500,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:             ModelProviderOpenRouter,
// 			ModelName:            "deepseek/deepseek-r1",
// 			ModelId:              "deepseek/deepseek-r1-reasoning",
// 			MaxTokens:            64000,
// 			MaxOutputTokens:      8192,
// 			ReservedOutputTokens: 8192,
// 			ApiKeyEnvVar:         ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility: ModelCompatibility{
// 				HasImageSupport: false,
// 			},
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 			IncludeReasoning:           true,
// 		},
// 	},

// 	{
// 		Description:           "DeepSeek R1 via OpenRouter (reasoning hidden)",
// 		DefaultMaxConvoTokens: 7500,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:             ModelProviderOpenRouter,
// 			ModelName:            "deepseek/deepseek-r1",
// 			ModelId:              "deepseek/deepseek-r1-no-reasoning",
// 			MaxTokens:            64000,
// 			MaxOutputTokens:      8192,
// 			ReservedOutputTokens: 8192,
// 			ApiKeyEnvVar:         ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility: ModelCompatibility{
// 				HasImageSupport: false,
// 			},
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 		},
// 	},

// 	{
// 		Description:           "Perplexity R1 1776 via OpenRouter (includes reasoning)",
// 		DefaultMaxConvoTokens: 7500,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:             ModelProviderOpenRouter,
// 			ModelName:            "perplexity/r1-1776",
// 			ModelId:              "perplexity/r1-1776",
// 			MaxTokens:            128000,
// 			MaxOutputTokens:      128000,
// 			ReservedOutputTokens: 30000,
// 			ApiKeyEnvVar:         ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility: ModelCompatibility{
// 				HasImageSupport: false,
// 			},
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 			IncludeReasoning:           true,
// 		},
// 	},

// 	{
// 		Description:           "Perplexity Sonar Reasoning via OpenRouter (includes reasoning)",
// 		DefaultMaxConvoTokens: 7500,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:             ModelProviderOpenRouter,
// 			ModelName:            "perplexity/sonar-reasoning",
// 			ModelId:              "perplexity/sonar-reasoning",
// 			MaxTokens:            127000,
// 			MaxOutputTokens:      127000,
// 			ReservedOutputTokens: 30000,
// 			ApiKeyEnvVar:         ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility: ModelCompatibility{
// 				HasImageSupport: false,
// 			},
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 			IncludeReasoning:           true,
// 		},
// 	},

// 	{
// 		Description:           "Qwen 2.5 Coder 32B via OpenRouter",
// 		DefaultMaxConvoTokens: 10000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "qwen/qwen-2.5-coder-32b-instruct",
// 			ModelId:                    "qwen/qwen-2.5-coder-32b-instruct",
// 			MaxTokens:                  128000,
// 			MaxOutputTokens:            8192,
// 			ReservedOutputTokens:       8192,
// 			ApiKeyEnvVar:               ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 		},
// 	},

// 	// OpenAI models via OpenRouter

// 	{
// 		Description:           "OpenAI o3-high via OpenRouter",
// 		DefaultMaxConvoTokens: 15000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "openai/o3",
// 			ModelId:                    "openai/o3-high",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
// 			ApiKeyEnvVar:               OpenRouterApiKeyEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 			SystemPromptDisabled:       true,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortHigh,
// 			StopDisabled:               true,
// 		},
// 	},

// 	{
// 		Description:           "OpenAI o3-medium via OpenRouter",
// 		DefaultMaxConvoTokens: 15000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "openai/o3",
// 			ModelId:                    "openai/o3-medium",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
// 			ApiKeyEnvVar:               OpenRouterApiKeyEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 			SystemPromptDisabled:       true,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortMedium,
// 			StopDisabled:               true,
// 		},
// 	},

// 	{
// 		Description:           "OpenAI o3-low via OpenRouter",
// 		DefaultMaxConvoTokens: 15000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "openai/o3",
// 			ModelId:                    "openai/o3-low",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
// 			ApiKeyEnvVar:               OpenRouterApiKeyEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatXml,
// 			SystemPromptDisabled:       true,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortLow,
// 			StopDisabled:               true,
// 		},
// 	},

// 	{
// 		Description:           "OpenAI o4-mini-high via OpenRouter",
// 		DefaultMaxConvoTokens: 10000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "openai/o4-mini",
// 			ModelId:                    "openai/o4-mini-high",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       30000,
// 			ApiKeyEnvVar:               OpenRouterApiKeyEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortHigh,
// 			StopDisabled:               true,
// 		},
// 	},
// 	{
// 		Description:           "OpenAI o4-mini-medium via OpenRouter",
// 		DefaultMaxConvoTokens: 10000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "openai/o4-mini",
// 			ModelId:                    "openai/o4-mini-medium",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
// 			ApiKeyEnvVar:               OpenRouterApiKeyEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortMedium,
// 			StopDisabled:               true,
// 		},
// 	},
// 	{
// 		Description:           "OpenAI o4-mini-low via OpenRouter",
// 		DefaultMaxConvoTokens: 10000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "openai/o4-mini",
// 			ModelId:                    "openai/o4-mini-low",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       40000, // 25k for reasoning, 15k for output
// 			ApiKeyEnvVar:               OpenRouterApiKeyEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortLow,
// 			StopDisabled:               true,
// 		},
// 	},

// 	{
// 		Description:           "OpenAI gpt-4.1 via OpenRouter",
// 		DefaultMaxConvoTokens: 75000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "openai/gpt-4.1",
// 			ModelId:                    "openai/gpt-4.1",
// 			MaxTokens:                  1047576,
// 			MaxOutputTokens:            32768,
// 			ReservedOutputTokens:       32768,
// 			ApiKeyEnvVar:               OpenRouterApiKeyEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 		},
// 	},

// 	{
// 		Description:           "OpenAI gpt-4.1-mini via OpenRouter",
// 		DefaultMaxConvoTokens: 75000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "openai/gpt-4.1-mini",
// 			ModelId:                    "openai/gpt-4.1-mini",
// 			MaxTokens:                  1047576,
// 			MaxOutputTokens:            32768,
// 			ReservedOutputTokens:       32768,
// 			ApiKeyEnvVar:               OpenRouterApiKeyEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 		},
// 	},

// 	{
// 		Description:           "OpenAI gpt-4.1-nano via OpenRouter",
// 		DefaultMaxConvoTokens: 75000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "openai/gpt-4.1-nano",
// 			ModelId:                    "openai/gpt-4.1-nano",
// 			MaxTokens:                  1047576,
// 			MaxOutputTokens:            32768,
// 			ReservedOutputTokens:       32768,
// 			ApiKeyEnvVar:               OpenRouterApiKeyEnvVar,
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 		},
// 	},

// 	{
// 		Description:           "OpenAI o3-mini-high via OpenRouter",
// 		DefaultMaxConvoTokens: 10000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "openai/o3-mini",
// 			ModelId:                    "openai/o3-mini-high",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       40000,
// 			ApiKeyEnvVar:               ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 			SystemPromptDisabled:       true,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortHigh,
// 		},
// 	},
// 	{
// 		Description:           "OpenAI o3-mini-medium via OpenRouter",
// 		DefaultMaxConvoTokens: 10000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "openai/o3-mini",
// 			ModelId:                    "openai/o3-mini-medium",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       40000,
// 			ApiKeyEnvVar:               ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 			SystemPromptDisabled:       true,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortMedium,
// 		},
// 	},
// 	{
// 		Description:           "OpenAI o3-mini-low via OpenRouter",
// 		DefaultMaxConvoTokens: 10000,
// 		BaseModelConfig: BaseModelConfig{
// 			Provider:                   ModelProviderOpenRouter,
// 			ModelName:                  "openai/o3-mini",
// 			ModelId:                    "openai/o3-mini-low",
// 			MaxTokens:                  200000,
// 			MaxOutputTokens:            100000,
// 			ReservedOutputTokens:       40000,
// 			ApiKeyEnvVar:               ApiKeyByProvider[ModelProviderOpenRouter],
// 			ModelCompatibility:         fullCompatibility,
// 			BaseUrl:                    BaseUrlByProvider[ModelProviderOpenRouter],
// 			PreferredModelOutputFormat: ModelOutputFormatToolCallJson,
// 			SystemPromptDisabled:       true,
// 			RoleParamsDisabled:         true,
// 			ReasoningEffortEnabled:     true,
// 			ReasoningEffort:            ReasoningEffortLow,
// 		},
// 	},
// }

var AvailableModelsByComposite = map[string]*AvailableModel{}

func init() {

	for _, model := range AvailableModels {
		if model.Description == "" {
			spew.Dump(model)
			panic("description is not set")
		}

		if model.Provider == "" {
			spew.Dump(model)
			panic("model provider is not set")
		}
		if model.ModelId == "" {
			spew.Dump(model)
			panic("model id is not set")
		}

		if model.DefaultMaxConvoTokens == 0 {
			spew.Dump(model)
			panic("default max convo tokens is not set")
		}

		if model.MaxTokens == 0 {
			spew.Dump(model)
			panic("max tokens is not set")
		}

		if model.MaxOutputTokens == 0 {
			spew.Dump(model)
			panic("max output tokens is not set")
		}

		if model.ReservedOutputTokens == 0 {
			spew.Dump(model)
			panic("reserved output tokens is not set")
		}

		if model.ApiKeyEnvVar == "" && len(model.ExtraAuthVars) == 0 && !model.SkipAuth && !model.HasAWSAuth {
			spew.Dump(model)
			panic("api key or auth settings are not set")
		}

		if model.BaseUrl == "" {
			spew.Dump(model)
			panic("base url is not set")
		}

		if model.PreferredModelOutputFormat == "" {
			spew.Dump(model)
			panic("preferred model output format is not set")
		}

		compositeKey := string(model.Provider) + "/" + string(model.ModelId)
		AvailableModelsByComposite[compositeKey] = model
	}
}

func GetAvailableModel(provider ModelProvider, modelId ModelId) *AvailableModel {
	compositeKey := string(provider) + "/" + string(modelId)
	return AvailableModelsByComposite[compositeKey]
}
