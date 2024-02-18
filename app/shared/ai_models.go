package shared

import (
	"github.com/sashabaranov/go-openai"
)

var AvailableModels = []BaseModelConfig{
	{
		Provider:  ModelProviderOpenAI,
		ModelName: openai.GPT4TurboPreview,
		MaxTokens: 128000,
	},
	{
		Provider:  ModelProviderOpenAI,
		ModelName: openai.GPT4Turbo0125,
		MaxTokens: 128000,
	},
	{
		Provider:  ModelProviderOpenAI,
		ModelName: openai.GPT4Turbo1106,
		MaxTokens: 128000,
	},
	{
		Provider:  ModelProviderOpenAI,
		ModelName: openai.GPT4,
		MaxTokens: 8000,
	},
	{
		Provider:  ModelProviderOpenAI,
		ModelName: openai.GPT3Dot5Turbo,
		MaxTokens: 16385,
	},
	{
		Provider:  ModelProviderOpenAI,
		ModelName: openai.GPT3Dot5Turbo0125,
		MaxTokens: 16385,
	},
	{
		Provider:  ModelProviderOpenAI,
		ModelName: openai.GPT3Dot5Turbo1106,
		MaxTokens: 16385,
	},
}

var PlannerModelConfigByName = map[string]PlannerModelConfig{
	openai.GPT4TurboPreview: {
		MaxConvoTokens:      10000,
		ReserveOutputTokens: 4096,
	},
	openai.GPT4Turbo0125: {
		MaxConvoTokens:      10000,
		ReserveOutputTokens: 4096,
	},
	openai.GPT4Turbo1106: {
		MaxConvoTokens:      10000,
		ReserveOutputTokens: 4096,
	},
	openai.GPT4: {
		MaxConvoTokens:      2500,
		ReserveOutputTokens: 1000,
	},
	openai.GPT3Dot5Turbo: {
		MaxConvoTokens:      5000,
		ReserveOutputTokens: 2000,
	},
	openai.GPT3Dot5Turbo0125: {
		MaxConvoTokens:      5000,
		ReserveOutputTokens: 2000,
	},
	openai.GPT3Dot5Turbo1106: {
		MaxConvoTokens:      5000,
		ReserveOutputTokens: 2000,
	},
}

var TaskModelConfigByName = map[string]TaskModelConfig{
	openai.GPT4TurboPreview: {
		OpenAIResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
	},
	openai.GPT4Turbo0125: {
		OpenAIResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
	},
	openai.GPT4Turbo1106: {
		OpenAIResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
	},
	openai.GPT4: {
		OpenAIResponseFormat: nil,
	},
	openai.GPT3Dot5Turbo: {
		OpenAIResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
	},
	openai.GPT3Dot5Turbo0125: {
		OpenAIResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
	},
	openai.GPT3Dot5Turbo1106: {
		OpenAIResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
	},
}

var AvailableModelsByName = map[string]BaseModelConfig{}
var DefaultModelSet ModelSet

func init() {
	for _, model := range AvailableModels {
		AvailableModelsByName[model.ModelName] = model
	}

	DefaultModelSet = ModelSet{
		Planner: PlannerRoleConfig{
			ModelRoleConfig: ModelRoleConfig{
				Role:            ModelRolePlannerRole,
				BaseModelConfig: AvailableModelsByName[openai.GPT4TurboPreview],
				Temperature:     0.6,
				TopP:            0.7,
			},
			PlannerModelConfig: PlannerModelConfigByName[openai.GPT4TurboPreview],
		},
		PlanSummary: ModelRoleConfig{
			Role:            ModelRolePlanSummaryRole,
			BaseModelConfig: AvailableModelsByName[openai.GPT4TurboPreview],
			Temperature:     0.3,
			TopP:            0.5,
		},
		Builder: TaskRoleConfig{
			ModelRoleConfig: ModelRoleConfig{
				Role:            ModelRoleBuilderRole,
				BaseModelConfig: AvailableModelsByName[openai.GPT4TurboPreview],
				Temperature:     0.3,
				TopP:            0.1,
			},
			TaskModelConfig: TaskModelConfigByName[openai.GPT4TurboPreview],
		},
		Namer: TaskRoleConfig{
			ModelRoleConfig: ModelRoleConfig{
				Role:            ModelRoleNameRole,
				BaseModelConfig: AvailableModelsByName[openai.GPT3Dot5Turbo],
				Temperature:     1.0,
				TopP:            0.5,
			},
			TaskModelConfig: TaskModelConfigByName[openai.GPT3Dot5Turbo],
		},
		CommitMsg: TaskRoleConfig{
			ModelRoleConfig: ModelRoleConfig{
				Role:            ModelRoleCommitMsgRole,
				BaseModelConfig: AvailableModelsByName[openai.GPT3Dot5Turbo],
				Temperature:     1.0,
				TopP:            0.5,
			},
			TaskModelConfig: TaskModelConfigByName[openai.GPT3Dot5Turbo],
		},
		ExecStatus: TaskRoleConfig{
			ModelRoleConfig: ModelRoleConfig{
				Role:            ModelRoleExecStatusRole,
				BaseModelConfig: AvailableModelsByName[openai.GPT4TurboPreview],
				Temperature:     0.1,
				TopP:            0.1,
			},
			TaskModelConfig: TaskModelConfigByName[openai.GPT4TurboPreview],
		},
	}
}
