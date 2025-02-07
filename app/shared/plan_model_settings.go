package shared

var SettingDescriptions = map[string]string{
	"max-convo-tokens":       "max conversation 🪙 before summarization",
	"max-tokens":             "overall 🪙 limit",
	"reserved-output-tokens": "🪙 reserved for model output",
}

var ModelOverridePropsDasherized = []string{"max-convo-tokens", "max-tokens", "reserved-output-tokens"}

func (ps PlanSettings) GetPlannerMaxTokens() int {
	if ps.ModelOverrides.MaxTokens == nil {
		if ps.ModelPack == nil {
			defaultPlanner := DefaultModelPack.Planner
			return defaultPlanner.GetFinalLargeContextFallback().BaseModelConfig.MaxTokens
		} else {
			planner := ps.ModelPack.Planner
			return planner.GetFinalLargeContextFallback().BaseModelConfig.MaxTokens
		}
	} else {
		return *ps.ModelOverrides.MaxTokens
	}
}

func (ps PlanSettings) GetPlannerMaxReservedOutputTokens() int {
	if ps.ModelOverrides.MaxTokens == nil {
		if ps.ModelPack == nil {
			defaultPlanner := DefaultModelPack.Planner
			return defaultPlanner.GetFinalLargeContextFallback().GetReservedOutputTokens()
		} else {
			planner := ps.ModelPack.Planner
			return planner.GetFinalLargeContextFallback().GetReservedOutputTokens()
		}
	} else {
		return *ps.ModelOverrides.MaxTokens
	}
}

func (ps PlanSettings) GetArchitectMaxTokens() int {
	if ps.ModelOverrides.MaxTokens == nil {
		if ps.ModelPack == nil {
			defaultLoader := DefaultModelPack.GetArchitect()
			return defaultLoader.GetFinalLargeContextFallback().BaseModelConfig.MaxTokens
		} else {
			loader := ps.ModelPack.GetArchitect()
			return loader.GetFinalLargeContextFallback().BaseModelConfig.MaxTokens
		}
	} else {
		return *ps.ModelOverrides.MaxTokens
	}
}

func (ps PlanSettings) GetArchitectMaxReservedOutputTokens() int {
	if ps.ModelOverrides.MaxTokens == nil {
		if ps.ModelPack == nil {
			defaultLoader := DefaultModelPack.GetArchitect()
			return defaultLoader.GetFinalLargeContextFallback().GetReservedOutputTokens()
		} else {
			loader := ps.ModelPack.GetArchitect()
			return loader.GetFinalLargeContextFallback().GetReservedOutputTokens()
		}
	} else {
		return *ps.ModelOverrides.MaxTokens
	}
}

func (ps PlanSettings) GetWholeFileBuilderMaxTokens() int {
	if ps.ModelOverrides.MaxTokens == nil {
		if ps.ModelPack == nil {
			defaultBuilder := DefaultModelPack.WholeFileBuilder
			return defaultBuilder.GetFinalLargeContextFallback().BaseModelConfig.MaxTokens
		} else {
			builder := ps.ModelPack.WholeFileBuilder
			return builder.GetFinalLargeContextFallback().BaseModelConfig.MaxTokens
		}
	} else {
		return *ps.ModelOverrides.MaxTokens
	}
}

func (ps PlanSettings) GetWholeFileBuilderMaxReservedOutputTokens() int {
	if ps.ModelOverrides.MaxTokens == nil {
		if ps.ModelPack == nil {
			defaultBuilder := DefaultModelPack.WholeFileBuilder
			return defaultBuilder.GetFinalLargeOutputFallback().GetReservedOutputTokens()
		} else {
			builder := ps.ModelPack.WholeFileBuilder
			return builder.GetFinalLargeOutputFallback().GetReservedOutputTokens()
		}
	} else {
		return *ps.ModelOverrides.MaxTokens
	}
}

func (ps PlanSettings) GetPlannerMaxConvoTokens() int {
	if ps.ModelOverrides.MaxConvoTokens == nil {
		if ps.ModelPack == nil {
			defaultPlanner := DefaultModelPack.Planner
			return defaultPlanner.GetFinalLargeContextFallback().MaxConvoTokens
		} else {
			planner := ps.ModelPack.Planner
			return planner.GetFinalLargeContextFallback().MaxConvoTokens
		}
	} else {
		return *ps.ModelOverrides.MaxConvoTokens
	}
}

func (ps PlanSettings) GetPlannerEffectiveMaxTokens() int {
	maxPlannerTokens := ps.GetPlannerMaxTokens()
	maxReservedOutputTokens := ps.GetPlannerMaxReservedOutputTokens()

	return maxPlannerTokens - maxReservedOutputTokens
}

func (ps PlanSettings) GetArchitectEffectiveMaxTokens() int {
	maxArchitectTokens := ps.GetArchitectMaxTokens()
	maxReservedOutputTokens := ps.GetArchitectMaxReservedOutputTokens()

	return maxArchitectTokens - maxReservedOutputTokens
}

func (ps PlanSettings) GetWholeFileBuilderEffectiveMaxTokens() int {
	maxWholeFileBuilderTokens := ps.GetWholeFileBuilderMaxTokens()
	maxReservedOutputTokens := ps.GetWholeFileBuilderMaxReservedOutputTokens()

	return maxWholeFileBuilderTokens - maxReservedOutputTokens
}

func (ps PlanSettings) GetRequiredEnvVars() map[string]bool {
	envVars := map[string]bool{}

	ms := ps.ModelPack
	if ms == nil {
		ms = DefaultModelPack
	}

	envVars[ms.Planner.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.Builder.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.WholeFileBuilder.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.PlanSummary.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.Namer.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.CommitMsg.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.ExecStatus.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.Architect.BaseModelConfig.ApiKeyEnvVar] = true
	envVars[ms.Coder.BaseModelConfig.ApiKeyEnvVar] = true

	// for backward compatibility with <= 0.8.4 server versions
	if len(envVars) == 0 {
		envVars["OPENAI_API_KEY"] = true
	}

	return envVars
}
