package shared

import (
	"fmt"
)

const OpenAIV1BaseUrl = "https://api.openai.com/v1"
const OpenRouterBaseUrl = "https://openrouter.ai/api/v1"
const LiteLLMBaseUrl = "http://localhost:4000/v1" // runs in the same container alongside the plandex server

const OpenAIEnvVar = "OPENAI_API_KEY"
const OpenRouterApiKeyEnvVar = "OPENROUTER_API_KEY"
const AnthropicApiKeyEnvVar = "ANTHROPIC_API_KEY"
const GoogleAIStudioApiKeyEnvVar = "GEMINI_API_KEY"
const AzureOpenAIEnvVar = "AZURE_OPENAI_API_KEY"
const DeepSeekApiKeyEnvVar = "DEEPSEEK_API_KEY"
const PerplexityApiKeyEnvVar = "PERPLEXITY_API_KEY"

type ModelPublisher string

const (
	ModelPublisherOpenAI     ModelPublisher = "OpenAI"
	ModelPublisherAnthropic  ModelPublisher = "Anthropic"
	ModelPublisherGoogle     ModelPublisher = "Google"
	ModelPublisherDeepSeek   ModelPublisher = "DeepSeek"
	ModelPublisherPerplexity ModelPublisher = "Perplexity"
	ModelPublisherQwen       ModelPublisher = "Qwen"
	ModelPublisherMistral    ModelPublisher = "Mistral"
)

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

	ModelProviderAmazonBedrock ModelProvider = "aws-bedrock"

	ModelProviderOllama ModelProvider = "ollama"

	ModelProviderCustom ModelProvider = "custom"
)

var ModelProviderToLiteLLMId = map[ModelProvider]string{
	ModelProviderGoogleAIStudio: "gemini",
	ModelProviderGoogleVertex:   "vertex_ai",
	ModelProviderAzureOpenAI:    "azure",
	ModelProviderAmazonBedrock:  "bedrock",
}

var AllModelProviders = []ModelProvider{
	ModelProviderOpenAI,
	ModelProviderOpenRouter,
	ModelProviderAnthropic,
	ModelProviderGoogleAIStudio,
	ModelProviderGoogleVertex,
	ModelProviderAzureOpenAI,
	ModelProviderDeepSeek,
	ModelProviderPerplexity,
	ModelProviderAmazonBedrock,
	ModelProviderOllama,
	ModelProviderCustom,
}

type ModelProviderExtraAuthVars struct {
	Var               string `json:"var"`
	MaybeJSONFilePath bool   `json:"maybeJSONFilePath,omitempty"`
	Required          bool   `json:"required,omitempty"`
	Default           string `json:"default,omitempty"`
}

type ModelProviderConfigSchema struct {
	Provider       ModelProvider `json:"provider"`
	CustomProvider *string       `json:"customProvider,omitempty"`
	BaseUrl        string        `json:"baseUrl"`

	// for AWS Bedrock models
	HasAWSAuth bool `json:"hasAWSAuth,omitempty"`

	// for local models that don't require auth (ollama, etc.)
	SkipAuth  bool `json:"skipAuth,omitempty"`
	LocalOnly bool `json:"localOnly,omitempty"`

	ApiKeyEnvVar  string                       `json:"apiKeyEnvVar,omitempty"`
	ExtraAuthVars []ModelProviderExtraAuthVars `json:"extraAuthVars,omitempty"`
}

func (m *ModelProviderConfigSchema) ToComposite() string {
	if m.CustomProvider != nil {
		return fmt.Sprintf("%s|%s", m.Provider, *m.CustomProvider)
	}
	return string(m.Provider)
}

const DefaultAzureApiVersion = "2025-04-01-preview"
const AnthropicMaxReasoningBudget = 32000
const GoogleMaxReasoningBudget = 32000

var BuiltInModelProviderConfigs = map[ModelProvider]ModelProviderConfigSchema{
	ModelProviderOpenAI: {
		Provider:     ModelProviderOpenAI,
		BaseUrl:      OpenAIV1BaseUrl,
		ApiKeyEnvVar: OpenAIEnvVar,
		ExtraAuthVars: []ModelProviderExtraAuthVars{
			{
				Var:      "OPENAI_ORG_ID",
				Required: false,
			},
		},
	},
	ModelProviderOpenRouter: {
		Provider:     ModelProviderOpenRouter,
		BaseUrl:      OpenRouterBaseUrl,
		ApiKeyEnvVar: OpenRouterApiKeyEnvVar,
	},
	ModelProviderAnthropic: {
		Provider:     ModelProviderAnthropic,
		BaseUrl:      LiteLLMBaseUrl,
		ApiKeyEnvVar: AnthropicApiKeyEnvVar,
	},
	ModelProviderGoogleAIStudio: {
		Provider:     ModelProviderGoogleAIStudio,
		BaseUrl:      LiteLLMBaseUrl,
		ApiKeyEnvVar: GoogleAIStudioApiKeyEnvVar,
	},
	ModelProviderGoogleVertex: {
		Provider: ModelProviderGoogleVertex,
		BaseUrl:  LiteLLMBaseUrl,
		ExtraAuthVars: []ModelProviderExtraAuthVars{
			{
				// this is a file path, but client-side it will be read and then passed along as an auth var just as if it were an env var
				Var:               "GOOGLE_APPLICATION_CREDENTIALS",
				MaybeJSONFilePath: true,
				Required:          true,
			},
			{
				Var:      "VERTEXAI_PROJECT",
				Required: true,
			},
			{
				Var:      "VERTEXAI_LOCATION",
				Required: true,
			},
		},
	},
	ModelProviderAzureOpenAI: {
		Provider:     ModelProviderAzureOpenAI,
		BaseUrl:      LiteLLMBaseUrl,
		ApiKeyEnvVar: AzureOpenAIEnvVar,
		ExtraAuthVars: []ModelProviderExtraAuthVars{
			{
				Var:      "AZURE_API_BASE",
				Required: true,
			},
			{
				Var:      "AZURE_API_VERSION",
				Required: false,
				Default:  DefaultAzureApiVersion,
			},
			{
				Var:               "AZURE_DEPLOYMENTS_MAP",
				Required:          false,
				MaybeJSONFilePath: true,
			},
		},
	},
	ModelProviderDeepSeek: {
		Provider:     ModelProviderDeepSeek,
		BaseUrl:      LiteLLMBaseUrl,
		ApiKeyEnvVar: DeepSeekApiKeyEnvVar,
	},
	ModelProviderPerplexity: {
		Provider:     ModelProviderPerplexity,
		BaseUrl:      LiteLLMBaseUrl,
		ApiKeyEnvVar: PerplexityApiKeyEnvVar,
	},
	ModelProviderAmazonBedrock: {
		Provider:   ModelProviderAmazonBedrock,
		BaseUrl:    LiteLLMBaseUrl,
		HasAWSAuth: true,

		// these aren't required as env vars—but if found in the credentials file, they are passed along as auth vars just as if they were env vars
		ExtraAuthVars: []ModelProviderExtraAuthVars{
			{Var: "AWS_ACCESS_KEY_ID", Required: true},
			{Var: "AWS_SECRET_ACCESS_KEY", Required: true},
			{Var: "AWS_REGION", Required: true},
			{Var: "AWS_SESSION_TOKEN", Required: false},
			{Var: "AWS_INFERENCE_PROFILE_ARN", Required: false},
		},
	},
	ModelProviderOllama: {
		Provider:  ModelProviderOllama,
		BaseUrl:   LiteLLMBaseUrl,
		SkipAuth:  true,
		LocalOnly: true,
	},
}

var BuiltInModelProviderConfigsByComposite = map[string]ModelProviderConfigSchema{}

func init() {
	for _, providerConfig := range BuiltInModelProviderConfigs {
		BuiltInModelProviderConfigsByComposite[providerConfig.ToComposite()] = providerConfig
	}
}

func GetProvidersForAuthVars(authVars map[string]string, settings *PlanSettings) []ModelProviderConfigSchema {
	var foundProviders []ModelProviderConfigSchema

	allProviders := []ModelProviderConfigSchema{}

	for _, providerConfig := range BuiltInModelProviderConfigs {
		allProviders = append(allProviders, providerConfig)
	}

	if settings != nil {
		for _, customProvider := range settings.CustomProviders {
			allProviders = append(allProviders, customProvider.ToModelProviderConfigSchema())
		}
	}

	for _, providerConfig := range allProviders {

		if providerConfig.SkipAuth {
			foundProviders = append(foundProviders, providerConfig)
			continue
		}

		var checkVars []string
		if providerConfig.ApiKeyEnvVar != "" {
			checkVars = append(checkVars, providerConfig.ApiKeyEnvVar)
		}
		for _, extraAuthVar := range providerConfig.ExtraAuthVars {
			if extraAuthVar.Required {
				checkVars = append(checkVars, extraAuthVar.Var)
			}
		}

		missingAny := false
		for _, checkVar := range checkVars {
			if _, ok := authVars[checkVar]; !ok {
				missingAny = true
				break
			}
		}
		if missingAny {
			continue
		}

		foundProviders = append(foundProviders, providerConfig)
	}

	return foundProviders
}

func GetProvidersForAuthVarsWithModelId(authVars map[string]string, settings *PlanSettings, modelId ModelId) []ModelProviderConfigSchema {
	var localProvider ModelProvider
	if settings != nil {
		modelPack := settings.GetModelPack()
		if modelPack != nil {
			localProvider = modelPack.LocalProvider
		}
	}

	builtInUsesProviders := BuiltInModelProvidersByModelId[modelId]

	var customUsesProviders []BaseModelUsesProvider
	if settings != nil {
		customUsesProviders = settings.UsesCustomProviderByModelId[modelId]
	}

	usesProviders := append(builtInUsesProviders, customUsesProviders...)
	if len(usesProviders) == 0 {
		return []ModelProviderConfigSchema{}
	}

	providers := GetProvidersForAuthVars(authVars, settings)
	providersByComposite := map[string]ModelProviderConfigSchema{}
	for _, provider := range providers {
		providersByComposite[provider.ToComposite()] = provider
	}

	res := []ModelProviderConfigSchema{}
	for _, usesProvider := range usesProviders {
		composite := usesProvider.ToComposite()
		provider, ok := providersByComposite[composite]
		if !ok {
			continue
		}

		if localProvider != "" {
			if !provider.LocalOnly {
				continue
			}
			if provider.Provider != localProvider {
				continue
			}
		}

		res = append(res, provider)
	}

	return res

}

func (m ModelRoleConfig) GetProvidersForAuthVars(authVars map[string]string, settings *PlanSettings) []ModelProviderConfigSchema {
	return GetProvidersForAuthVarsWithModelId(authVars, settings, m.ModelId)
}

func (m ModelRoleConfig) GetFirstProviderForAuthVars(authVars map[string]string, settings *PlanSettings) *ModelProviderConfigSchema {
	providers := m.GetProvidersForAuthVars(authVars, settings)
	if len(providers) == 0 {
		return nil
	}
	return &providers[0]
}
