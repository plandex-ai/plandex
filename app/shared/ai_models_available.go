package shared

import (
	"strings"

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

'PreferredOutputFormat' is the preferred output format for the model—currently either 'ModelOutputFormatToolCallJson' or 'ModelOutputFormatXml' — OpenAI models like JSON (and benefit from strict JSON schemas), while most other providers are unreliable for JSON generation and do better with XML, even if they claim to support JSON.

'RoleParamsDisabled' is used to disable role-based parameters like temperature, top_p, etc. for the model—OpenAI early releases often don't allow changes to these.

'SystemPromptDisabled' is used to disable the system prompt for the model—OpenAI early releases sometimes don't allow system prompts.

'ReasoningEffortEnabled' is used to enable reasoning effort for the model (like OpenAI's o3-mini).

'ReasoningEffort' is the reasoning effort for the model, when 'ReasoningEffortEnabled' is true.

'PredictedOutputEnabled' is used to enable predicted output for the model (currently only supported by gpt-4o).

'ApiKeyEnvVar' is the environment variable that contains the API key for the model.
*/

var BuiltInModels = []*BaseModelConfigSchema{
	{
		ModelTag:    "openai/o3",
		Publisher:   ModelPublisherOpenAI,
		Description: "OpenAI o3",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens:  15000,
			MaxTokens:              200000,
			MaxOutputTokens:        100000,
			ReservedOutputTokens:   40000, // 25k for reasoning, 15k for output
			ModelCompatibility:     fullCompatibility,
			PreferredOutputFormat:  ModelOutputFormatXml,
			SystemPromptDisabled:   true,
			RoleParamsDisabled:     true,
			ReasoningEffortEnabled: true,
			StopDisabled:           true,
		},

		RequiresVariantOverrides: []string{
			"ReasoningEffort",
		},

		Variants: []BaseModelConfigVariant{
			{
				VariantTag:  "high",
				Description: "high",
				Overrides: BaseModelShared{
					ReasoningEffort: ReasoningEffortHigh,
				},
			},
			{
				VariantTag:  "medium",
				Description: "medium",
				Overrides: BaseModelShared{
					ReasoningEffort: ReasoningEffortMedium,
				},
			},
			{
				VariantTag:  "low",
				Description: "low",
				Overrides: BaseModelShared{
					ReasoningEffort: ReasoningEffortLow,
				},
			},
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "o3",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "azure/o3",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/o3",
			},
		},
	},
	{
		ModelTag:    "openai/o4-mini",
		Publisher:   ModelPublisherOpenAI,
		Description: "OpenAI o4-mini",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens:  10000,
			MaxTokens:              200000,
			MaxOutputTokens:        100000,
			ReservedOutputTokens:   40000, // 25k for reasoning, 15k for output
			ModelCompatibility:     fullCompatibility,
			PreferredOutputFormat:  ModelOutputFormatToolCallJson,
			SystemPromptDisabled:   true,
			RoleParamsDisabled:     true,
			ReasoningEffortEnabled: true,
			ReasoningEffort:        ReasoningEffortHigh,
			StopDisabled:           true,
		},

		RequiresVariantOverrides: []string{
			"ReasoningEffort",
		},

		Variants: []BaseModelConfigVariant{
			{
				VariantTag:  "high",
				Description: "high",
				Overrides: BaseModelShared{
					ReasoningEffort: ReasoningEffortHigh,
				},
			},
			{
				VariantTag:  "medium",
				Description: "medium",
				Overrides: BaseModelShared{
					ReasoningEffort:      ReasoningEffortMedium,
					ReservedOutputTokens: 30000, // 15k for reasoning, 15k for output
				},
			},
			{
				VariantTag:  "low",
				Description: "low",
				Overrides: BaseModelShared{
					ReasoningEffort:      ReasoningEffortLow,
					ReservedOutputTokens: 20000, // 5-10k for reasoning, 5-10k for output
				},
			},
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "o4-mini",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "azure/o4-mini",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/o4-mini",
			},
		},
	},

	{
		ModelTag:    "openai/gpt-4.1",
		Publisher:   ModelPublisherOpenAI,
		Description: "OpenAI gpt-4.1",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens: 75000,
			MaxTokens:             1047576,
			MaxOutputTokens:       32768,
			ReservedOutputTokens:  32768,
			ModelCompatibility:    fullCompatibility,
			PreferredOutputFormat: ModelOutputFormatToolCallJson,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "gpt-4.1",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "azure/gpt-4.1",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/gpt-4.1",
			},
		},
	},

	{
		ModelTag:    "openai/gpt-4.1-mini",
		Publisher:   ModelPublisherOpenAI,
		Description: "OpenAI gpt-4.1-mini",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens: 75000,
			MaxTokens:             1047576,
			MaxOutputTokens:       32768,
			ReservedOutputTokens:  32768,
			ModelCompatibility:    fullCompatibility,
			PreferredOutputFormat: ModelOutputFormatToolCallJson,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "gpt-4.1-mini",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "azure/gpt-4.1-mini",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/gpt-4.1-mini",
			},
		},
	},

	{
		ModelTag:    "openai/gpt-4.1-nano",
		Publisher:   ModelPublisherOpenAI,
		Description: "OpenAI gpt-4.1-nano",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens: 75000,
			MaxTokens:             1047576,
			MaxOutputTokens:       32768,
			ReservedOutputTokens:  32768,
			ModelCompatibility:    fullCompatibility,
			PreferredOutputFormat: ModelOutputFormatToolCallJson,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "gpt-4.1-nano",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "azure/gpt-4.1-nano",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/gpt-4.1-nano",
			},
		},
	},

	{
		ModelTag:    "openai/o3-mini",
		Publisher:   ModelPublisherOpenAI,
		Description: "OpenAI o3-mini",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens:  10000,
			MaxTokens:              200000,
			MaxOutputTokens:        100000,
			ReservedOutputTokens:   40000, // 25k for reasoning, 15k for output
			ModelCompatibility:     fullCompatibility,
			PreferredOutputFormat:  ModelOutputFormatToolCallJson,
			SystemPromptDisabled:   true,
			RoleParamsDisabled:     true,
			ReasoningEffortEnabled: true,
			ReasoningEffort:        ReasoningEffortHigh,
			StopDisabled:           true,
		},

		RequiresVariantOverrides: []string{
			"ReasoningEffort",
		},

		Variants: []BaseModelConfigVariant{
			{
				VariantTag:  "high",
				Description: "high",
				Overrides: BaseModelShared{
					ReasoningEffort: ReasoningEffortHigh,
				},
			},
			{
				VariantTag:  "medium",
				Description: "medium",
				Overrides: BaseModelShared{
					ReasoningEffort:      ReasoningEffortMedium,
					ReservedOutputTokens: 30000, // 15k for reasoning, 15k for output
				},
			},
			{
				VariantTag:  "low",
				Description: "low",
				Overrides: BaseModelShared{
					ReasoningEffort:      ReasoningEffortLow,
					ReservedOutputTokens: 20000, // 5-10k for reasoning, 5-10k for output
				},
			},
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderOpenAI,
				ModelName: "o3-mini",
			},
			{
				Provider:  ModelProviderAzureOpenAI,
				ModelName: "azure/o3-mini",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/o3-mini",
			},
		},
	},

	{
		ModelTag:    "anthropic/claude-3.7-sonnet",
		Publisher:   ModelPublisherAnthropic,
		Description: "Anthropic Claude 3.7 Sonnet",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens:       15000,
			MaxTokens:                   200000,
			MaxOutputTokens:             128000,
			ReservedOutputTokens:        20000,
			SupportsCacheControl:        true,
			PreferredOutputFormat:       ModelOutputFormatXml,
			SingleMessageNoSystemPrompt: true,
			TokenEstimatePaddingPct:     0.10,
		},

		Variants: []BaseModelConfigVariant{
			{
				IsBaseVariant: true,
			},
			{
				VariantTag:  "thinking",
				Description: "thinking",
				Overrides: BaseModelShared{
					ReasoningBudget: AnthropicMaxReasoningBudget,
				},

				Variants: []BaseModelConfigVariant{
					{
						VariantTag:  "visible",
						Description: "visible",
						Overrides: BaseModelShared{
							IncludeReasoning: true,
						},
					},
					{
						VariantTag:  "hidden",
						Description: "hidden",
						Overrides: BaseModelShared{
							IncludeReasoning: false,
						},
					},
				},
			},
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderAnthropic,
				ModelName: "anthropic/claude-3-7-sonnet-latest",
			},
			{
				Provider:  ModelProviderAmazonBedrock,
				ModelName: "bedrock/anthropic.claude-3-7-sonnet-20250219-v1:0",
			},
			{
				Provider:  ModelProviderGoogleVertex,
				ModelName: "vertex_ai/claude-3-7-sonnet@20250219",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "anthropic/claude-3.7-sonnet",
			},
		},
	},

	{
		ModelTag:    "anthropic/claude-3.5-sonnet",
		Publisher:   ModelPublisherAnthropic,
		Description: "Anthropic Claude 3.5 Sonnet",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens:       15000,
			MaxTokens:                   200000,
			MaxOutputTokens:             128000,
			ReservedOutputTokens:        20000,
			SupportsCacheControl:        true,
			PreferredOutputFormat:       ModelOutputFormatXml,
			SingleMessageNoSystemPrompt: true,
			TokenEstimatePaddingPct:     0.10,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderAnthropic,
				ModelName: "anthropic/claude-3-5-sonnet-latest",
			},
			{
				Provider:  ModelProviderGoogleVertex,
				ModelName: "vertex_ai/claude-3-5-sonnet@20240620",
			},
			{
				Provider:  ModelProviderAmazonBedrock,
				ModelName: "bedrock/anthropic.claude-3-5-sonnet-20241022-v2:0",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "anthropic/claude-3.5-sonnet",
			},
		},
	},

	{
		ModelTag:    "anthropic/claude-3.5-haiku",
		Publisher:   ModelPublisherAnthropic,
		Description: "Anthropic Claude 3.5 Haiku",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens:       15000,
			MaxTokens:                   200000,
			MaxOutputTokens:             8192,
			ReservedOutputTokens:        8192,
			SupportsCacheControl:        true,
			PreferredOutputFormat:       ModelOutputFormatXml,
			SingleMessageNoSystemPrompt: true,
			TokenEstimatePaddingPct:     0.10,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderAnthropic,
				ModelName: "anthropic/claude-3-5-haiku-latest",
			},
			{
				Provider:  ModelProviderGoogleVertex,
				ModelName: "vertex_ai/claude-3-5-haiku@20241022",
			},
			{
				Provider:  ModelProviderAmazonBedrock,
				ModelName: "bedrock/anthropic.claude-3-5-haiku-20241022-v1:0",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "anthropic/claude-3.5-haiku",
			},
		},
	},

	{
		ModelTag:    "google/gemini-pro-1.5",
		Publisher:   ModelPublisherGoogle,
		Description: "Google Gemini 1.5 Pro",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens: 75000,
			MaxTokens:             2000000,
			MaxOutputTokens:       8192,
			ReservedOutputTokens:  8192,
			// SupportsCacheControl:       true, // gemini now uses implicit caching
			PreferredOutputFormat: ModelOutputFormatXml,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderGoogleAIStudio,
				ModelName: "gemini/gemini-1.5-pro",
			},
			{
				Provider:  ModelProviderGoogleVertex,
				ModelName: "vertex_ai/gemini-1.5-pro-latest",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "google/gemini-pro-1.5",
			},
		},
	},
	{
		ModelTag:    "google/gemini-2.5-pro-preview",
		Publisher:   ModelPublisherGoogle,
		Description: "Google Gemini 2.5 Pro (Preview)",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens: 75000,
			MaxTokens:             1048576,
			MaxOutputTokens:       65535,
			ReservedOutputTokens:  65535,
			// SupportsCacheControl:       true, // gemini now uses implicit caching
			PreferredOutputFormat: ModelOutputFormatXml,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderGoogleAIStudio,
				ModelName: "gemini/gemini-2.5-pro-preview-05-06",
			},
			{
				Provider:  ModelProviderGoogleVertex,
				ModelName: "vertex_ai/gemini-2.5-pro-preview-05-06",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "google/gemini-2.5-pro-preview",
			},
		},
	},
	{
		ModelTag:    "google/gemini-2.5-pro-exp",
		Publisher:   ModelPublisherGoogle,
		Description: "Google Gemini 2.5 Pro (Experimental)",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens: 75000,
			MaxTokens:             1048576,
			MaxOutputTokens:       65535,
			ReservedOutputTokens:  65535,
			PreferredOutputFormat: ModelOutputFormatXml,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderGoogleAIStudio,
				ModelName: "gemini/gemini-2.5-pro-exp-03-25",
			},
			{
				Provider:  ModelProviderGoogleVertex,
				ModelName: "vertex_ai/gemini-2.5-pro-exp-03-25",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "google/gemini-2.5-pro-exp-03-25",
			},
		},
	},
	{
		ModelTag:    "google/gemini-2.5-flash-preview",
		Publisher:   ModelPublisherGoogle,
		Description: "Google Gemini 2.5 Flash (Preview)",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens: 75000,

			MaxTokens:            1048576,
			MaxOutputTokens:      65535,
			ReservedOutputTokens: 65535,
			// SupportsCacheControl:       true, // gemini now uses implicit caching
			PreferredOutputFormat: ModelOutputFormatXml,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderGoogleAIStudio,
				ModelName: "gemini/gemini-2.5-flash-preview-04-17",
			},
			{
				Provider:  ModelProviderGoogleVertex,
				ModelName: "vertex_ai/gemini-2.5-flash-preview-04-17",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "google/gemini-2.5-flash-preview",
			},
		},
	},

	{
		ModelTag:    "deepseek/v3-0324",
		Publisher:   ModelPublisherDeepSeek,
		Description: "DeepSeek V3 (0324)",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens: 7500,

			MaxTokens:             64000,
			MaxOutputTokens:       8192,
			ReservedOutputTokens:  8192,
			PreferredOutputFormat: ModelOutputFormatXml,
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderDeepSeek,
				ModelName: "deepseek/deepseek-chat",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "deepseek/deepseek-chat-v3-0324",
			},
		},
	},
	{
		ModelTag:    "deepseek/r1",
		Publisher:   ModelPublisherDeepSeek,
		Description: "DeepSeek R1",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens: 7500,

			MaxTokens:             64000,
			MaxOutputTokens:       8192,
			ReservedOutputTokens:  8192,
			PreferredOutputFormat: ModelOutputFormatXml,
		},

		Variants: []BaseModelConfigVariant{
			{
				VariantTag:  "reasoning-visible",
				Description: "(reasoning visible)",
				Overrides: BaseModelShared{
					IncludeReasoning: true,
				},
			},
			{
				VariantTag:  "reasoning-hidden",
				Description: "(reasoning hidden)",
				Overrides: BaseModelShared{
					IncludeReasoning: false,
				},
			},
		},

		Providers: []BaseModelUsesProvider{
			{
				Provider:  ModelProviderDeepSeek,
				ModelName: "deepseek/deepseek-reasoner",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "deepseek/deepseek-r1",
			},
		},
	},

	{
		ModelTag:    "perplexity/r1-1776",
		Publisher:   ModelPublisherPerplexity,
		Description: "Perplexity R1-1776",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens: 7500,

			MaxTokens:             128000,
			MaxOutputTokens:       128000,
			ReservedOutputTokens:  30000,
			PreferredOutputFormat: ModelOutputFormatXml,
		},

		Variants: []BaseModelConfigVariant{
			{
				VariantTag:  "reasoning-visible",
				Description: "(reasoning visible)",
				Overrides: BaseModelShared{
					IncludeReasoning: true,
				},
			},
			{
				VariantTag:  "reasoning-hidden",
				Description: "(reasoning hidden)",
				Overrides: BaseModelShared{
					IncludeReasoning: false,
				},
			},
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderPerplexity, ModelName: "r1-1776-online"},
			{Provider: ModelProviderOpenRouter, ModelName: "perplexity/r1-1776"},
		},
	},
	{
		ModelTag:    "perplexity/sonar-reasoning",
		Description: "Perplexity Sonar Reasoning",
		Publisher:   ModelPublisherPerplexity,

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens: 7500,

			MaxTokens:             127000,
			MaxOutputTokens:       127000,
			ReservedOutputTokens:  30000,
			PreferredOutputFormat: ModelOutputFormatXml,
		},

		Variants: []BaseModelConfigVariant{
			{
				VariantTag:  "visible",
				Description: "(reasoning visible)",
				Overrides: BaseModelShared{
					IncludeReasoning: true,
				},
			},
			{
				VariantTag:  "hidden",
				Description: "(reasoning hidden)",
				Overrides: BaseModelShared{
					IncludeReasoning: false,
				},
			},
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderPerplexity, ModelName: "sonar-reasoning-online"},
			{Provider: ModelProviderOpenRouter, ModelName: "perplexity/sonar-reasoning"},
		},
	},

	{
		ModelTag:    "qwen/qwen-2.5-coder-32b-instruct",
		Publisher:   ModelPublisherQwen,
		Description: "Qwen 2.5 Coder 32B (Instruct)",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens: 10000,

			MaxTokens:             128000,
			MaxOutputTokens:       8192,
			ReservedOutputTokens:  8192,
			PreferredOutputFormat: ModelOutputFormatXml,
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderOpenRouter, ModelName: "qwen/qwen-2.5-coder-32b-instruct"},
		},
	},

	{
		ModelTag:    "qwen/qwen3-32b",
		Publisher:   ModelPublisherQwen,
		Description: "Qwen 3-32B (Experimental)",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens: 15000,
			MaxTokens:             128000,
			MaxOutputTokens:       16384,
			ReservedOutputTokens:  16384,
			PreferredOutputFormat: ModelOutputFormatXml,
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
		ModelTag:    "qwen/qwen3-8b",
		Publisher:   ModelPublisherQwen,
		Description: "Qwen 3-8B (Experimental)",

		BaseModelShared: BaseModelShared{
			DefaultMaxConvoTokens: 7500,
			MaxTokens:             32768,
			MaxOutputTokens:       8192,
			ReservedOutputTokens:  8192,
			PreferredOutputFormat: ModelOutputFormatXml,
		},

		Providers: []BaseModelUsesProvider{
			{Provider: ModelProviderOpenRouter, ModelName: "qwen/qwen3-8b-04-28"},
		},
	},
}

var BuiltInBaseModelsById = map[ModelId]*BaseModelConfigSchema{}

var BuiltInModelProvidersByModelId = map[ModelId][]BaseModelUsesProvider{}
var BuiltInBaseModels = []*BaseModelConfigSchema{}

var AvailableModels = []*AvailableModel{}

var AvailableModelsByComposite = map[string]*AvailableModel{}

func init() {
	for _, model := range BuiltInModels {
		AvailableModels = append(AvailableModels, model.ToAvailableModels()...)

		var addVariants func(variants []BaseModelConfigVariant, baseId ModelId)
		addVariants = func(variants []BaseModelConfigVariant, baseId ModelId) {
			for _, variant := range variants {
				var modelId ModelId
				if variant.IsBaseVariant {
					modelId = baseId
				} else {
					modelId = ModelId(strings.Join([]string{string(baseId), string(variant.VariantTag)}, "-"))
				}

				if len(variant.Variants) > 0 {
					addVariants(variant.Variants, modelId)
					continue
				}

				model.ModelId = modelId
				BuiltInModelProvidersByModelId[modelId] = model.Providers
				BuiltInBaseModelsById[modelId] = model
				BuiltInBaseModels = append(BuiltInBaseModels, model)
			}
		}

		if len(model.Variants) > 0 {
			addVariants(model.Variants, ModelId(string(model.ModelTag)))
		} else {
			modelId := ModelId(string(model.ModelTag))
			model.ModelId = modelId
			BuiltInModelProvidersByModelId[modelId] = model.Providers
			BuiltInBaseModelsById[modelId] = model
			BuiltInBaseModels = append(BuiltInBaseModels, model)
		}
	}

	// fmt.Println("AvailableModels")
	// spew.Dump(AvailableModels)

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

		if model.PreferredOutputFormat == "" {
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
