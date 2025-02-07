package shared

var PlandexDefaultModelPack ModelPack
var ClaudeModelPack ModelPack
var O3MiniModelPack ModelPack

//TODO: built-in deepseek/qwen model pack(s)

// So far Gemini isn't able to follow instructions reliably enough for the 'coder' role to use it for a full model pack

var BuiltInModelPacks = []*ModelPack{
	&PlandexDefaultModelPack,
	&ClaudeModelPack,
	&O3MiniModelPack,
}

var DefaultModelPack *ModelPack = &PlandexDefaultModelPack

func init() {
	PlandexDefaultModelPack = ModelPack{
		Name:        "plandex-default-pack",
		Description: "A mix of models from Anthropic, OpenAI, and Google. Supports up to 2M context.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *claude35sonnet(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "anthropic/claude-3.5-sonnet"),
			PlannerLargeContextFallback: &PlannerRoleConfig{
				ModelRoleConfig: *gemini15pro(ModelRolePlanner, nil),
				PlannerModelConfig: PlannerModelConfig{
					// Use the same max convo tokens as the default model so we don't mess with summarization too much
					MaxConvoTokens: GetAvailableModel(ModelProviderOpenRouter, "anthropic/claude-3.5-sonnet").DefaultMaxConvoTokens,
				},
			},
		},
		Coder: claude35sonnet(ModelRoleCoder, nil),
		Architect: claude35sonnet(ModelRoleArchitect, &modelConfigFallbacks{
			largeContextFallback: gemini15pro(ModelRoleArchitect, nil),
		}),
		PlanSummary:      *openaio3mini(ModelRolePlanSummary, ReasoningEffortLow, nil),
		Builder:          *openaio3mini(ModelRoleBuilder, ReasoningEffortMedium, nil),
		WholeFileBuilder: openaio3mini(ModelRoleWholeFileBuilder, ReasoningEffortLow, nil),
		Namer:            *openai4omini(ModelRoleName, nil),
		CommitMsg:        *openai4omini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio3mini(ModelRoleExecStatus, ReasoningEffortMedium, nil),
	}

	O3MiniModelPack = ModelPack{
		Name:        "o3-mini-pack",
		Description: "OpenAI blend. Supports up to 200k context. Uses OpenAI's o3-mini model for heavy lifting, GPT-4o Mini for lighter tasks.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *openaio3mini(ModelRolePlanner, ReasoningEffortHigh, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenAI, "o3-mini"),
		},
		PlanSummary:      *openaio3mini(ModelRolePlanSummary, ReasoningEffortLow, nil),
		Builder:          *openaio3mini(ModelRoleBuilder, ReasoningEffortMedium, nil),
		WholeFileBuilder: openaio3mini(ModelRoleWholeFileBuilder, ReasoningEffortMedium, nil),
		Namer:            *openai4omini(ModelRoleName, nil),
		CommitMsg:        *openai4omini(ModelRoleCommitMsg, nil),
		ExecStatus:       *openaio3mini(ModelRoleExecStatus, ReasoningEffortMedium, nil),
	}

	ClaudeModelPack = ModelPack{
		Name:        "claude-3-5-sonnet-pack",
		Description: "Anthropic blend. Supports up to 200k context. Uses Claude 3.5 Sonnet for heavy lifting, Claude 3 Haiku for lighter tasks.",
		Planner: PlannerRoleConfig{
			ModelRoleConfig:    *claude35sonnet(ModelRolePlanner, nil),
			PlannerModelConfig: getPlannerModelConfig(ModelProviderOpenRouter, "anthropic/claude-3.5-sonnet"),
		},
		PlanSummary: *claude35haiku(ModelRolePlanSummary, nil),
		Builder:     *claude35sonnet(ModelRoleBuilder, nil),
		Namer:       *claude35haiku(ModelRoleName, nil),
		CommitMsg:   *claude35haiku(ModelRoleCommitMsg, nil),
		ExecStatus:  *claude35sonnet(ModelRoleExecStatus, nil),
	}
}

type modelConfigFallbacks struct {
	largeContextFallback *ModelRoleConfig
	largeOutputFallback  *ModelRoleConfig
	// errorFallback        *ModelRoleConfig
}

func getModelConfig(role ModelRole, provider ModelProvider, modelName string, fallbacks *modelConfigFallbacks) *ModelRoleConfig {
	if fallbacks == nil {
		fallbacks = &modelConfigFallbacks{}
	}

	return &ModelRoleConfig{
		Role:            role,
		BaseModelConfig: GetAvailableModel(provider, modelName).BaseModelConfig,
		Temperature:     DefaultConfigByRole[role].Temperature,
		TopP:            DefaultConfigByRole[role].TopP,

		LargeContextFallback: fallbacks.largeContextFallback,
		LargeOutputFallback:  fallbacks.largeOutputFallback,
		// ErrorFallback:        fallbacks.errorFallback,
	}
}

// allow OpenAI calls to first try OpenAI directly, then fallback to OpenRouter (which can route to Azure if OpenAI is down)
func getOpenAIModelConfig(role ModelRole, modelName string, reasoningEffort ReasoningEffort, fallbacks *modelConfigFallbacks) *ModelRoleConfig {
	var largeContextFallback *ModelRoleConfig
	var largeOutputFallback *ModelRoleConfig
	if fallbacks != nil {
		largeContextFallback = fallbacks.largeContextFallback
		largeOutputFallback = fallbacks.largeOutputFallback
	}

	config := getModelConfig(role, ModelProviderOpenAI, modelName, &modelConfigFallbacks{
		largeContextFallback: largeContextFallback,
		largeOutputFallback:  largeOutputFallback,
		// errorFallback:        getModelConfig(role, ModelProviderOpenRouter, "openai/gpt-4o", nil),
	})

	if reasoningEffort != "" {
		config.ReasoningEffort = reasoningEffort
	}

	return config
}

func claude35sonnet(role ModelRole, fallbacks *modelConfigFallbacks) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "anthropic/claude-3.5-sonnet", fallbacks)
}

func claude35haiku(role ModelRole, fallbacks *modelConfigFallbacks) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "anthropic/claude-3.5-haiku", fallbacks)
}

func gemini15pro(role ModelRole, fallbacks *modelConfigFallbacks) *ModelRoleConfig {
	return getModelConfig(role, ModelProviderOpenRouter, "google/gemini-pro-1.5", fallbacks)
}

func openai4o(role ModelRole, fallbacks *modelConfigFallbacks) *ModelRoleConfig {
	return getOpenAIModelConfig(role, "gpt-4o", "", fallbacks)
}

func openai4omini(role ModelRole, fallbacks *modelConfigFallbacks) *ModelRoleConfig {
	return getOpenAIModelConfig(role, "gpt-4o-mini", "", fallbacks)
}

func openaio1mini(role ModelRole, fallbacks *modelConfigFallbacks) *ModelRoleConfig {
	return getOpenAIModelConfig(role, "o1-mini", "", fallbacks)
}

func openaio3mini(role ModelRole, reasoningEffort ReasoningEffort, fallbacks *modelConfigFallbacks) *ModelRoleConfig {
	return getOpenAIModelConfig(role, "o3-mini", reasoningEffort, fallbacks)
}
