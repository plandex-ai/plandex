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
			StopDisabled:               true,
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
				ModelName: "o3",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/o3",
			},
		},
	},
	{
		ModelTag:              "openai/o4-mini",
		Description:           "OpenAI o4-mini",
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
				ModelName: "o4-mini",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/o4-mini",
			},
		},
	},

	{
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
		ModelTag:              "openai/o3-mini",
		Description:           "OpenAI o3-mini",
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
				ModelName: "o3-mini",
			},
			{
				Provider:  ModelProviderOpenRouter,
				ModelName: "openai/o3-mini",
			},
		},
	},

	{
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
		ModelTag:              "deepseek/r1",
		Description:           "DeepSeek R1",
		DefaultMaxConvoTokens: 7500,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  64000,
			MaxOutputTokens:            8192,
			ReservedOutputTokens:       8192,
			PreferredModelOutputFormat: ModelOutputFormatXml,
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
			{Provider: ModelProviderDeepSeek, ModelName: "deepseek-reasoner"},
			{Provider: ModelProviderOpenRouter, ModelName: "deepseek/deepseek-r1"},
		},
	},

	{
		ModelTag:              "perplexity/r1-1776",
		Description:           "Perplexity R1-1776",
		DefaultMaxConvoTokens: 7500,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  128000,
			MaxOutputTokens:            128000,
			ReservedOutputTokens:       30000,
			PreferredModelOutputFormat: ModelOutputFormatXml,
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
		ModelTag:              "perplexity/sonar-reasoning",
		Description:           "Perplexity Sonar Reasoning",
		DefaultMaxConvoTokens: 7500,

		BaseModelShared: BaseModelShared{
			MaxTokens:                  127000,
			MaxOutputTokens:            127000,
			ReservedOutputTokens:       30000,
			PreferredModelOutputFormat: ModelOutputFormatXml,
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

var BuiltInModelProvidersByModelId = map[ModelId][]BaseModelUsesProvider{}

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

				BuiltInModelProvidersByModelId[modelId] = model.Providers
			}
		}

		if len(model.Variants) > 0 {
			addVariants(model.Variants, ModelId(string(model.ModelTag)))
		} else {
			BuiltInModelProvidersByModelId[ModelId(string(model.ModelTag))] = model.Providers
		}
	}

	spew.Dump(BuiltInModelProvidersByModelId)

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
