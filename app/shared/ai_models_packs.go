package shared

var DailyDriverModelPack ModelPack
var ReasoningModelPack ModelPack
var StrongModelPack ModelPack

var Sonnet4ModelPack ModelPack
var Opus4PlannerModelPack ModelPack

var OSSModelPack ModelPack
var CheapModelPack ModelPack
var AnthropicModelPack ModelPack
var OpenAIModelPack ModelPack

var GeminiPreviewModelPack ModelPack
var GeminiExperimentalModelPack ModelPack
var R1PlannerModelPack ModelPack
var PerplexityPlannerModelPack ModelPack

var BuiltInModelPacks = []*ModelPack{
	&DailyDriverModelPack,
	&ReasoningModelPack,
	&StrongModelPack,
	&CheapModelPack,
	&OSSModelPack,

	&Sonnet4ModelPack,
	&Opus4PlannerModelPack,

	&AnthropicModelPack,
	&OpenAIModelPack,

	&GeminiPreviewModelPack,
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
			ModelRoleConfig: *claude37Sonnet(ModelRolePlanner, &modelConfig{
				largeContextFallback: geminipro25preview(ModelRolePlanner, &modelConfig{
					largeContextFallback: gemini15pro(ModelRolePlanner, nil),
				}),
			}),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "anthropic/claude-sonnet-4"),
		},
		Coder: claude37Sonnet(ModelRoleCoder, &modelConfig{
			largeContextFallback: openaigpt41(ModelRoleCoder, nil),
		}),
		Architect: claude37Sonnet(ModelRoleArchitect, &modelConfig{
			largeContextFallback: geminipro25preview(ModelRolePlanner, &modelConfig{
				largeContextFallback: gemini15pro(ModelRolePlanner, nil),
			}),
		}),
		PlanSummary: *openaio4miniLow(ModelRolePlanSummary, nil),
		Builder: *openaio4miniMedium(ModelRoleBuilder, &modelConfig{
			strongModel: openaio4miniHigh(ModelRoleBuilder, nil),
		}),
		WholeFileBuilder: openaio4miniMedium(ModelRoleWholeFileBuilder, nil),
		Namer:            *openaigpt41mini(ModelRoleName, nil),
		CommitMsg:        *openaigpt41mini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio4miniLow(ModelRoleExecStatus, nil),
	}

	Sonnet4ModelPack = ModelPack{
		Name:        "sonnet-4-daily",
		Description: "Same as daily driver, but uses newly released Claude Sonnet 4 in place of 3.7",
		Planner: PlannerRoleConfig{
			ModelRoleConfig: *claudeSonnet4(ModelRolePlanner, &modelConfig{
				largeContextFallback: geminipro25preview(ModelRolePlanner, &modelConfig{
					largeContextFallback: gemini15pro(ModelRolePlanner, nil),
				}),
			}),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "anthropic/claude-sonnet-4"),
		},
		Coder: claudeSonnet4(ModelRoleCoder, &modelConfig{
			largeContextFallback: openaigpt41(ModelRoleCoder, nil),
		}),
		Architect: claudeSonnet4(ModelRoleArchitect, &modelConfig{
			largeContextFallback: geminipro25preview(ModelRolePlanner, &modelConfig{
				largeContextFallback: gemini15pro(ModelRolePlanner, nil),
			}),
		}),
		PlanSummary: *openaio4miniLow(ModelRolePlanSummary, nil),
		Builder: *openaio4miniMedium(ModelRoleBuilder, &modelConfig{
			strongModel: openaio4miniHigh(ModelRoleBuilder, nil),
		}),
		WholeFileBuilder: openaio4miniMedium(ModelRoleWholeFileBuilder, nil),
		Namer:            *openaigpt41mini(ModelRoleName, nil),
		CommitMsg:        *openaigpt41mini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio4miniLow(ModelRoleExecStatus, nil),
	}

	Opus4PlannerModelPack = ModelPack{
		Name:        "opus-4-planner",
		Description: "Like daily driver, but uses newly released Claude Opus 4 for planning, and Sonnet 4 for coding. Supports up to 160k input context.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *claudeOpus4(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "anthropic/claude-opus-4"),
		},
		Coder:       claudeSonnet4(ModelRoleCoder, nil),
		Architect:   claudeOpus4(ModelRoleArchitect, nil),
		PlanSummary: *openaio4miniLow(ModelRolePlanSummary, nil),
		Builder: *openaio4miniMedium(ModelRoleBuilder, &modelConfig{
			strongModel: openaio4miniHigh(ModelRoleBuilder, nil),
		}),
		WholeFileBuilder: openaio4miniMedium(ModelRoleWholeFileBuilder, nil),
		Namer:            *openaigpt41mini(ModelRoleName, nil),
		CommitMsg:        *openaigpt41mini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio4miniLow(ModelRoleExecStatus, nil),
	}

	ReasoningModelPack = ModelPack{
		Name:        "reasoning",
		Description: "Like the daily driver, but uses 3.7-sonnet:thinking with reasoning enabled for planning and coding. Supports up to 160k input context.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *claude37SonnetThinkingHidden(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "anthropic/claude-3.7-sonnet:thinking-hidden"),
		},
		Coder:       claude37SonnetThinkingHidden(ModelRoleCoder, nil),
		PlanSummary: *openaio4miniLow(ModelRolePlanSummary, nil),
		Builder: *openaio4miniMedium(ModelRoleBuilder, &modelConfig{
			strongModel: openaio4miniHigh(ModelRoleBuilder, nil),
		}),

		WholeFileBuilder: openaio4miniMedium(ModelRoleWholeFileBuilder, nil),
		Namer:            *openaigpt41mini(ModelRoleName, nil),
		CommitMsg:        *openaigpt41mini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio4miniLow(ModelRoleExecStatus, nil),
	}

	StrongModelPack = ModelPack{
		Name:        "strong",
		Description: "For difficult tasks where slower responses and builds are ok. Uses o3-high for architecture and planning, claude-3.7-sonnet thinking for implementation, prioritizes reliability over speed for builds. Supports up to 160k input context.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *openaio3high(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenAI, "openai/o3-high"),
		},
		Architect:        openaio3high(ModelRoleArchitect, nil),
		Coder:            claude37SonnetThinkingHidden(ModelRoleCoder, nil),
		PlanSummary:      *openaio4miniLow(ModelRolePlanSummary, nil),
		Builder:          *openaio4miniHigh(ModelRoleBuilder, nil),
		WholeFileBuilder: openaio4miniHigh(ModelRoleWholeFileBuilder, nil),
		Namer:            *openaigpt41mini(ModelRoleName, nil),
		CommitMsg:        *openaigpt41mini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio4miniMedium(ModelRoleExecStatus, nil),
	}

	CheapModelPack = ModelPack{
		Name:        "cheap",
		Description: "Cost-effective models that can still get the job done for easier tasks. Supports up to 160k context. Uses OpenAI's o4-mini model for planning, GPT-4.1 for coding, and GPT-4.1 Mini for lighter tasks.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *openaio4miniMedium(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenAI, "openai/o4-mini-medium"),
		},
		Coder:       openaigpt41(ModelRoleCoder, nil),
		PlanSummary: *openaigpt41mini(ModelRolePlanSummary, nil),
		Builder: *openaio4miniLow(ModelRoleBuilder, &modelConfig{
			strongModel: openaio4miniMedium(ModelRoleBuilder, nil),
		}),
		WholeFileBuilder: openaio4miniLow(ModelRoleWholeFileBuilder, nil),
		Namer:            *openaigpt41mini(ModelRoleName, nil),
		CommitMsg:        *openaigpt41mini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio4miniLow(ModelRoleExecStatus, nil),
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

	OpenAIModelPack = ModelPack{
		Name:        "openai",
		Description: "OpenAI blend. Supports up to 1M context. Uses OpenAI's GPT-4.1 model for heavy lifting, GPT-4.1 Mini for lighter tasks.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *openaigpt41(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenAI, "openai/gpt-4.1"),
		},
		PlanSummary: *openaio4miniLow(ModelRolePlanSummary, nil),
		Builder: *openaio4miniMedium(ModelRoleBuilder, &modelConfig{
			strongModel: openaio4miniHigh(ModelRoleBuilder, nil),
		}),
		WholeFileBuilder: openaio4miniMedium(ModelRoleWholeFileBuilder, nil),
		Namer:            *openaigpt41mini(ModelRoleName, nil),
		CommitMsg:        *openaigpt41mini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio4miniLow(ModelRoleExecStatus, nil),
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

	GeminiPreviewModelPack = ModelPack{
		Name:        "gemini-preview",
		Description: "Uses Gemini 2.5 Pro Preview for planning and coding, default models for other roles. Supports up to 1M input context.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *geminipro25preview(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "google/gemini-2.5-pro-preview"),
		},
		Coder:       geminipro25preview(ModelRoleCoder, nil),
		PlanSummary: *openaio4miniLow(ModelRolePlanSummary, nil),
		Builder: *openaio4miniMedium(ModelRoleBuilder, &modelConfig{
			strongModel: openaio4miniHigh(ModelRoleBuilder, nil),
		}),
		WholeFileBuilder: openaio4miniMedium(ModelRoleWholeFileBuilder, nil),
		Namer:            *openaigpt41mini(ModelRoleName, nil),
		CommitMsg:        *openaigpt41mini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio4miniLow(ModelRoleExecStatus, nil),
	}

	GeminiExperimentalModelPack = ModelPack{
		Name:        "gemini-exp",
		Description: "Uses Gemini 2.5 Pro Experimental (free) for planning and coding, default models for other roles. Supports up to 1M input context.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *geminipro25exp(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "google/gemini-2.5-pro-exp-03-25"),
		},
		Coder:       geminipro25exp(ModelRoleCoder, nil),
		PlanSummary: *openaio4miniLow(ModelRolePlanSummary, nil),
		Builder: *openaio4miniMedium(ModelRoleBuilder, &modelConfig{
			strongModel: openaio4miniHigh(ModelRoleBuilder, nil),
		}),
		WholeFileBuilder: openaio4miniMedium(ModelRoleWholeFileBuilder, nil),
		Namer:            *openaigpt41mini(ModelRoleName, nil),
		CommitMsg:        *openaigpt41mini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio4miniLow(ModelRoleExecStatus, nil),
	}

	R1PlannerModelPack = ModelPack{
		Name:        "r1-planner",
		Description: "Uses DeepSeek R1 for planning, Qwen for light tasks, and default models for implementation. Supports up to 56k input context.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *deepseekr1Reasoning(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "deepseek/deepseek-r1-reasoning"),
		},
		Coder:            claude37Sonnet(ModelRoleCoder, nil),
		PlanSummary:      *openaio4miniLow(ModelRolePlanSummary, nil),
		Builder:          *openaio4miniMedium(ModelRoleBuilder, nil),
		WholeFileBuilder: openaio4miniLow(ModelRoleWholeFileBuilder, nil),
		Namer:            *openaigpt41mini(ModelRoleName, nil),
		CommitMsg:        *openaigpt41mini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio4miniMedium(ModelRoleExecStatus, nil),
	}

	PerplexityPlannerModelPack = ModelPack{
		Name:        "perplexity-planner",
		Description: "Uses Perplexity Sonar for planning, Qwen for light tasks, and default models for implementation. Supports up to 97k input context.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *perplexitySonarReasoning(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "perplexity/sonar-reasoning"),
		},
		Coder:            claude37Sonnet(ModelRoleCoder, nil),
		PlanSummary:      *openaio4miniLow(ModelRolePlanSummary, nil),
		Builder:          *openaio4miniMedium(ModelRoleBuilder, nil),
		WholeFileBuilder: openaio4miniLow(ModelRoleWholeFileBuilder, nil),
		Namer:            *openaigpt41mini(ModelRoleName, nil),
		CommitMsg:        *openaigpt41mini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio4miniMedium(ModelRoleExecStatus, nil),
	}

}

type modelConfig struct {
	largeContextFallback *ModelRoleConfig
	largeOutputFallback  *ModelRoleConfig
	errorFallback        *ModelRoleConfig
	strongModel          *ModelRoleConfig
	missingKeyFallback   *ModelRoleConfig
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
		ErrorFallback:        fallbacks.errorFallback,
		StrongModel:          fallbacks.strongModel,
		MissingKeyFallback:   fallbacks.missingKeyFallback,
	}
}

func claude37Sonnet(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "anthropic/claude-3.7-sonnet", fallbacks)
}

func claudeSonnet4(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "anthropic/claude-sonnet-4", fallbacks)
}

func claudeOpus4(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "anthropic/claude-opus-4", fallbacks)
}

func claudeSonnet4With37Fallback(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	claude37Fallback := getModelConfig(role, ModelProviderOpenRouter, "anthropic/claude-3.7-sonnet", nil)
	if fallbacks == nil {
		fallbacks = &modelConfig{}
	}
	fallbacks.errorFallback = claude37Fallback

	return getModelConfig(role, ModelProviderOpenRouter, "anthropic/claude-sonnet-4", fallbacks)
}

func claude37SonnetThinking(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "anthropic/claude-3.7-sonnet:thinking", fallbacks)
}

func claude37SonnetThinkingHidden(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "anthropic/claude-3.7-sonnet:thinking-hidden", fallbacks)
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

func gemini25propreview(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "google/gemini-2.5-pro-preview", fallbacks)
}

func openaigpt41(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	openrouterFallback := getModelConfig(role, ModelProviderOpenRouter, "openai/gpt-4.1", nil)
	if fallbacks == nil {
		fallbacks = &modelConfig{}
	}
	fallbacks.missingKeyFallback = openrouterFallback
	return getModelConfig(role, ModelProviderOpenAI, "openai/gpt-4.1", fallbacks)
}

func openaigpt41mini(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	openrouterFallback := getModelConfig(role, ModelProviderOpenRouter, "openai/gpt-4.1-mini", nil)
	if fallbacks == nil {
		fallbacks = &modelConfig{}
	}
	fallbacks.missingKeyFallback = openrouterFallback
	return getModelConfig(role, ModelProviderOpenAI, "openai/gpt-4.1-mini", fallbacks)
}

func openaio3highWitho4miniFallback(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	openrouterFallback := getModelConfig(role, ModelProviderOpenRouter, "openai/o3-high", nil)
	if fallbacks == nil {
		fallbacks = &modelConfig{}
	}
	fallbacks.missingKeyFallback = openrouterFallback
	fallbacks.errorFallback = openaio4miniHigh(role, nil)
	return getModelConfig(role, ModelProviderOpenAI, "openai/o3-high", fallbacks)
}

func openaio3high(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	openrouterFallback := getModelConfig(role, ModelProviderOpenRouter, "openai/o3-high", nil)
	if fallbacks == nil {
		fallbacks = &modelConfig{}
	}
	fallbacks.missingKeyFallback = openrouterFallback
	return getModelConfig(role, ModelProviderOpenAI, "openai/o3-high", fallbacks)
}

func openaio3medium(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	openrouterFallback := getModelConfig(role, ModelProviderOpenRouter, "openai/o3-medium", nil)
	if fallbacks == nil {
		fallbacks = &modelConfig{}
	}
	fallbacks.missingKeyFallback = openrouterFallback
	return getModelConfig(role, ModelProviderOpenAI, "openai/o3-medium", fallbacks)
}

func openaio3low(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	openrouterFallback := getModelConfig(role, ModelProviderOpenRouter, "openai/o3-low", nil)
	if fallbacks == nil {
		fallbacks = &modelConfig{}
	}
	fallbacks.missingKeyFallback = openrouterFallback
	return getModelConfig(role, ModelProviderOpenAI, "openai/o3-low", fallbacks)
}

func openaio4miniHigh(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	openrouterFallback := getModelConfig(role, ModelProviderOpenRouter, "openai/o4-mini-high", nil)
	if fallbacks == nil {
		fallbacks = &modelConfig{}
	}
	fallbacks.missingKeyFallback = openrouterFallback
	return getModelConfig(role, ModelProviderOpenAI, "openai/o4-mini-high", fallbacks)
}

func openaio4miniMedium(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	openrouterFallback := getModelConfig(role, ModelProviderOpenRouter, "openai/o4-mini-medium", nil)
	if fallbacks == nil {
		fallbacks = &modelConfig{}
	}
	fallbacks.missingKeyFallback = openrouterFallback
	res := getModelConfig(role, ModelProviderOpenAI, "openai/o4-mini-medium", fallbacks)
	return res
}

func openaio4miniLow(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	openrouterFallback := getModelConfig(role, ModelProviderOpenRouter, "openai/o4-mini-low", nil)
	if fallbacks == nil {
		fallbacks = &modelConfig{}
	}
	fallbacks.missingKeyFallback = openrouterFallback
	return getModelConfig(role, ModelProviderOpenAI, "openai/o4-mini-low", fallbacks)
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

func geminiflash25preview(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "google/gemini-2.5-flash-preview", fallbacks)
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
	return getModelConfig(role, ModelProviderOpenRouter, "google/gemini-2.5-pro-exp-03-25", fallbacks)
}

func geminipro25preview(role ModelRole, fallbacks *modelConfig) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "google/gemini-2.5-pro-preview", fallbacks)
}
