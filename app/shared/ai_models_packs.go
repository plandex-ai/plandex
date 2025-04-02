package shared

var DailyDriverModelPack ModelPack
var ReasoningModelPack ModelPack
var StrongModelPack ModelPack

// var CrazyModelPack ModelPack
var OSSModelPack ModelPack
var CheapModelPack ModelPack
var AnthropicModelPack ModelPack
var OpenAIModelPack ModelPack

// var GeminiModelPack ModelPack
var GeminiExperimentalModelPack ModelPack
var R1PlannerModelPack ModelPack
var PerplexityPlannerModelPack ModelPack

var BuiltInModelPacks = []*ModelPack{
	&DailyDriverModelPack,
	&ReasoningModelPack,
	&StrongModelPack,
	&CheapModelPack,
	&OSSModelPack,
	// &GeminiPlannerModelPack,
	// &CrazyModelPack,
	&AnthropicModelPack,
	&OpenAIModelPack,
	// &GeminiModelPack,
	&GeminiExperimentalModelPack,
	&R1PlannerModelPack,
	&PerplexityPlannerModelPack,
}

var DefaultModelPack *ModelPack = &DailyDriverModelPack

func init() {
	DailyDriverModelPack = ModelPack{
		Name:        "daily-driver",
		Description: "A mix of models from Anthropic, OpenAI, and Google that balances speed, quality, and cost. Supports up to 2M context. Plandex prompts are especially tested and optimized for this pack.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *claude37Sonnet(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "anthropic/claude-3.7-sonnet"),
			PlannerLargeContextFallback: &PlannerRoleConfig{
				ModelRoleConfig: *gemini15pro(ModelRolePlanner, nil),
				PlannerModelConfig: PlannerModelConfig{
					// Use the same max convo tokens as the default model so we don't mess with summarization too much
					MaxConvoTokens: GetAvailableModel(ModelProviderOpenRouter, "google/gemini-pro-1.5").DefaultMaxConvoTokens,
				},
			},
		},
		Coder: claude37Sonnet(ModelRoleCoder, nil),
		Architect: claude37Sonnet(ModelRoleArchitect, &modelConfig{
			largeContextFallback: gemini15pro(ModelRoleArchitect, nil),
		}),
		PlanSummary: *openaio3miniLow(ModelRolePlanSummary, nil),
		Builder: *openaio3miniMedium(ModelRoleBuilder, &modelConfig{
			strongModel: openaio3miniHigh(ModelRoleBuilder, nil),
		}),
		WholeFileBuilder: openaio3miniMedium(ModelRoleWholeFileBuilder, nil),
		Namer:            *openai4omini(ModelRoleName, nil),
		CommitMsg:        *openai4omini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio3miniLow(ModelRoleExecStatus, nil),
	}

	ReasoningModelPack = ModelPack{
		Name:        "reasoning",
		Description: "Like the daily driver, but use 3.7-sonnet:thinking with reasoning enabled for planning and coding.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *claude37SonnetThinking(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "anthropic/claude-3.7-sonnet:thinking"),
		},
		Coder:       claude37SonnetThinking(ModelRoleCoder, nil),
		PlanSummary: *openaio3miniLow(ModelRolePlanSummary, nil),
		Builder: *openaio3miniMedium(ModelRoleBuilder, &modelConfig{
			strongModel: openaio3miniHigh(ModelRoleBuilder, nil),
		}),

		WholeFileBuilder: openaio3miniMedium(ModelRoleWholeFileBuilder, nil),
		Namer:            *openai4omini(ModelRoleName, nil),
		CommitMsg:        *openai4omini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio3miniLow(ModelRoleExecStatus, nil),
	}

	StrongModelPack = ModelPack{
		Name:        "strong",
		Description: "For difficult tasks where slower responses and builds are ok. Uses o1 for architecture and planning, claude-3.7-sonnet for implementation, prioritizes reliability over speed for builds. Supports up to 160k input context.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *openaio1(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenAI, "openai/o1"),
		},
		Architect:        openaio1(ModelRoleArchitect, nil),
		Coder:            claude37SonnetThinking(ModelRoleCoder, nil),
		PlanSummary:      *openaio3miniLow(ModelRolePlanSummary, nil),
		Builder:          *openaio3miniHigh(ModelRoleBuilder, nil),
		WholeFileBuilder: openaio3miniHigh(ModelRoleWholeFileBuilder, nil),
		Namer:            *openai4omini(ModelRoleName, nil),
		CommitMsg:        *openai4omini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio3miniMedium(ModelRoleExecStatus, nil),
	}

	CheapModelPack = ModelPack{
		Name:        "cheap",
		Description: "Cost-effective models that can still get the job done for easier tasks. Supports up to 160k context. Uses OpenAI's o3-mini model for heavy lifting, GPT-4o Mini for lighter tasks.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *openaio3miniMedium(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenAI, "openai/o3-mini-medium"),
		},
		PlanSummary: *openai4omini(ModelRolePlanSummary, nil),
		Builder: *openaio3miniLow(ModelRoleBuilder, &modelConfig{
			strongModel: openaio3miniMedium(ModelRoleBuilder, nil),
		}),
		WholeFileBuilder: openaio3miniLow(ModelRoleWholeFileBuilder, nil),
		Namer:            *openai4omini(ModelRoleName, nil),
		CommitMsg:        *openai4omini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio3miniLow(ModelRoleExecStatus, nil),
	}

	OSSModelPack = ModelPack{
		Name:        "oss",
		Description: "An experimental mix of the best open source models for coding. Supports up to 56k context, 8k per file. Works best with smaller projects and files. Includes reasoning.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *deepseekr1Reasoning(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "deepseek/deepseek-r1-reasoning"),
		},
		Coder:            deepseekv3(ModelRoleCoder, nil),
		PlanSummary:      *deepseekr1NoReasoning(ModelRolePlanSummary, nil),
		Builder:          *deepseekr1NoReasoning(ModelRoleBuilder, nil),
		WholeFileBuilder: deepseekr1NoReasoning(ModelRoleWholeFileBuilder, nil),
		Namer:            *qwen25coder32b(ModelRoleName, nil),
		CommitMsg:        *qwen25coder32b(ModelRoleCommitMsg, nil),
		ExecStatus:       *deepseekr1NoReasoning(ModelRoleExecStatus, nil),
	}

	// Disabled for now - o1-pro is not supported yet
	// CrazyModelPack = ModelPack{
	// 	Name:        "crazy",
	// 	Description: "Uses o1-pro for planning, 3.7 sonnet for implementation, o3-mini-high for builds and auto-continue checks, and gpt-4o for lighter tasks. Slow and expensive. Highly capable. Be careful, you can spend hundreds of dollars on a large task.",
	// 	Planner: PlannerRoleConfig{
	// 		ModelRoleConfig: *openaio1pro(ModelRolePlanner, nil),
	// 	},
	// 	Architect:        openaio1pro(ModelRoleArchitect, nil),
	// 	Coder:            claude37Sonnet(ModelRoleCoder, nil),
	// 	PlanSummary:      *openaio3miniLow(ModelRolePlanSummary, nil),
	// 	Builder:          *openaio3miniHigh(ModelRoleBuilder, nil),
	// 	WholeFileBuilder: openaio3miniHigh(ModelRoleWholeFileBuilder, nil),
	// 	Namer:            *openai4o(ModelRoleName, nil),
	// 	CommitMsg:        *openai4o(ModelRoleCommitMsg, nil),
	// 	ExecStatus:       *openaio3miniHigh(ModelRoleExecStatus, nil),
	// }

	OpenAIModelPack = ModelPack{
		Name:        "openai",
		Description: "OpenAI blend. Supports up to 160k context. Uses OpenAI's o3-mini model for heavy lifting, GPT-4o Mini for lighter tasks.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *openaio3miniHigh(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenAI, "openai/o3-mini-high"),
		},
		PlanSummary: *openaio3miniLow(ModelRolePlanSummary, nil),
		Builder: *openaio3miniLow(ModelRoleBuilder, &modelConfig{
			strongModel: openaio3miniHigh(ModelRoleBuilder, nil),
		}),
		WholeFileBuilder: openaio3miniLow(ModelRoleWholeFileBuilder, nil),
		Namer:            *openai4omini(ModelRoleName, nil),
		CommitMsg:        *openai4omini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio3miniLow(ModelRoleExecStatus, nil),
	}

	AnthropicModelPack = ModelPack{
		Name:        "anthropic",
		Description: "Anthropic blend. Supports up to 180k context. Uses Claude 3.5 Sonnet for heavy lifting, Claude 3 Haiku for lighter tasks.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *claude37Sonnet(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "anthropic/claude-3.7-sonnet"),
		},
		PlanSummary:      *claude35haiku(ModelRolePlanSummary, nil),
		Builder:          *claude37Sonnet(ModelRoleBuilder, nil),
		WholeFileBuilder: claude37Sonnet(ModelRoleWholeFileBuilder, nil),
		Namer:            *claude35haiku(ModelRoleName, nil),
		CommitMsg:        *claude35haiku(ModelRoleCommitMsg, nil),
		ExecStatus:       *claude37Sonnet(ModelRoleExecStatus, nil),
	}

	// GeminiModelPack = ModelPack{
	// 	Name:        "gemini-experimental",
	// 	Description: "Uses Gemini 2.5 Pro experimental (free) for heavy lifting, Gemini Flash 2.0 for light tasks. Supports up to 1M input context.",
	// 	Planner: PlannerRoleConfig{
	// 		ModelRoleConfig:    *geminipro25exp(ModelRolePlanner, nil),
	// 		PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "google/gemini-2.5-pro-exp-03-25:free"),
	// 	},
	// 	Coder:            geminipro25exp(ModelRoleCoder, nil),
	// 	PlanSummary:      *geminiflash20(ModelRolePlanSummary, nil),
	// 	Builder:          *geminipro25exp(ModelRoleBuilder, nil),
	// 	WholeFileBuilder: geminipro25exp(ModelRoleWholeFileBuilder, nil),
	// 	Namer:            *geminiflash20(ModelRoleName, nil),
	// 	CommitMsg:        *geminiflash20(ModelRoleCommitMsg, nil),
	// 	ExecStatus:       *geminipro25exp(ModelRoleExecStatus, nil),
	// }

	GeminiExperimentalModelPack = ModelPack{
		Name:        "gemini-exp",
		Description: "Uses Gemini 2.5 Pro Experimental for planning and coding, default models for other roles. Supports up to 1M input context.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *geminipro25exp(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "google/gemini-2.5-pro-exp-03-25:free"),
		},
		Coder:       geminipro25exp(ModelRoleCoder, nil),
		PlanSummary: *openaio3miniLow(ModelRolePlanSummary, nil),
		Builder: *openaio3miniMedium(ModelRoleBuilder, &modelConfig{
			strongModel: openaio3miniHigh(ModelRoleBuilder, nil),
		}),
		WholeFileBuilder: openaio3miniMedium(ModelRoleWholeFileBuilder, nil),
		Namer:            *openai4omini(ModelRoleName, nil),
		CommitMsg:        *openai4omini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio3miniLow(ModelRoleExecStatus, nil),
	}

	R1PlannerModelPack = ModelPack{
		Name:        "r1-planner",
		Description: "Uses DeepSeek R1 for planning, Qwen for light tasks, and default models for implementation. Supports up to 56k input context.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *deepseekr1Reasoning(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "deepseek/deepseek-r1-reasoning"),
		},
		Coder:            claude37Sonnet(ModelRoleCoder, nil),
		PlanSummary:      *openaio3miniLow(ModelRolePlanSummary, nil),
		Builder:          *openaio3miniMedium(ModelRoleBuilder, nil),
		WholeFileBuilder: openaio3miniLow(ModelRoleWholeFileBuilder, nil),
		Namer:            *openai4omini(ModelRoleName, nil),
		CommitMsg:        *openai4omini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio3miniMedium(ModelRoleExecStatus, nil),
	}

	PerplexityPlannerModelPack = ModelPack{
		Name:        "perplexity-planner",
		Description: "Uses Perplexity Sonar for planning, Qwen for light tasks, and default models for implementation. Supports up to 97k input context.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *perplexitySonarReasoning(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "perplexity/sonar-reasoning"),
		},
		Coder:            claude37Sonnet(ModelRoleCoder, nil),
		PlanSummary:      *openaio3miniLow(ModelRolePlanSummary, nil),
		Builder:          *openaio3miniMedium(ModelRoleBuilder, nil),
		WholeFileBuilder: openaio3miniLow(ModelRoleWholeFileBuilder, nil),
		Namer:            *openai4omini(ModelRoleName, nil),
		CommitMsg:        *openai4omini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio3miniMedium(ModelRoleExecStatus, nil),
	}

}

type modelConfig struct {
	largeContextFallback *ModelRoleConfig
	largeOutputFallback  *ModelRoleConfig
	// errorFallback        *ModelRoleConfig
	strongModel *ModelRoleConfig
}

func getModelConfig(role ModelRole, provider ModelProvider, modelId ModelId, fallbacks *modelConfig) *ModelRoleConfig {
	if fallbacks == nil {
		fallbacks = &modelConfig{}
	}

	return &ModelRoleConfig{
		Role:            role,
		BaseModelConfig: GetAvailableModel(provider, modelId).BaseModelConfig,
		Temperature:     DefaultConfigByRole[role].Temperature,
		TopP:            DefaultConfigByRole[role].TopP,

		LargeContextFallback: fallbacks.largeContextFallback,
		LargeOutputFallback:  fallbacks.largeOutputFallback,
		// ErrorFallback:        fallbacks.errorFallback,
		StrongModel: fallbacks.strongModel,
	}
}

func claude37Sonnet(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "anthropic/claude-3.7-sonnet", fallbacks)
}

func claude37SonnetThinking(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "anthropic/claude-3.7-sonnet:thinking", fallbacks)
}

func claude35Sonnet(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "anthropic/claude-3.5-sonnet", fallbacks)
}

func claude35haiku(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "anthropic/claude-3.5-haiku", fallbacks)
}

func gemini15pro(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "google/gemini-pro-1.5", fallbacks)
}

func openai4o(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenAI, "openai/gpt-4o", fallbacks)
}

func openai4omini(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenAI, "openai/gpt-4o-mini", fallbacks)
}

func openaio3miniHigh(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenAI, "openai/o3-mini-high", fallbacks)
}

func openaio3miniMedium(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenAI, "openai/o3-mini-medium", fallbacks)
}

func openaio3miniLow(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenAI, "openai/o3-mini-low", fallbacks)
}

func openaio1(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenAI, "openai/o1", fallbacks)
}

func openaio1pro(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenAI, "openai/o1-pro", fallbacks)
}

func qwen25coder32b(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "qwen/qwen-2.5-coder-32b-instruct", fallbacks)
}

func deepseekr1Reasoning(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "deepseek/deepseek-r1-reasoning", fallbacks)
}

func deepseekr1NoReasoning(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "deepseek/deepseek-r1-no-reasoning", fallbacks)
}

func deepseekv3(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "deepseek/deepseek-chat-v3-0324", fallbacks)
}

func geminiflash20(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "google/gemini-2.0-flash-001", fallbacks)
}

func perplexityR11776(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "perplexity/r1-1776", fallbacks)
}

func perplexitySonarReasoning(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "perplexity/sonar-reasoning", fallbacks)
}

// func r1distillqwen32b(role ModelRole, fallbacks *modelConfigFallbacks) *ModelRoleConfig {
// 	return getModelConfig(role, ModelProviderOpenRouter, "deepseek/deepseek-r1-distill-qwen-32b", fallbacks)
// }

func geminipro25exp(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "google/gemini-2.5-pro-exp-03-25:free", fallbacks)
}
