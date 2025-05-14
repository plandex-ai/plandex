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
			fallback := defaultPlanner.GetFinalLargeContextFallback()
			return fallback.GetSharedBaseConfig().MaxTokens
		} else {
			planner := ps.ModelPack.Planner
			fallback := planner.GetFinalLargeContextFallback()
			return fallback.GetSharedBaseConfig().MaxTokens
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
			fallback := defaultLoader.GetFinalLargeContextFallback()
			return fallback.GetSharedBaseConfig().MaxTokens
		} else {
			loader := ps.ModelPack.GetArchitect()
			fallback := loader.GetFinalLargeContextFallback()
			return fallback.GetSharedBaseConfig().MaxTokens
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

func (ps PlanSettings) GetCoderMaxTokens() int {
	if ps.ModelOverrides.MaxTokens == nil {
		if ps.ModelPack == nil {
			defaultCoder := DefaultModelPack.Coder
			fallback := defaultCoder.GetFinalLargeContextFallback()
			return fallback.GetSharedBaseConfig().MaxTokens
		} else {
			coder := ps.ModelPack.Coder
			fallback := coder.GetFinalLargeContextFallback()
			return fallback.GetSharedBaseConfig().MaxTokens
		}
	} else {
		return *ps.ModelOverrides.MaxTokens
	}
}

func (ps PlanSettings) GetCoderMaxReservedOutputTokens() int {
	if ps.ModelOverrides.MaxTokens == nil {
		if ps.ModelPack == nil {
			defaultCoder := DefaultModelPack.Coder
			return defaultCoder.GetFinalLargeContextFallback().GetReservedOutputTokens()
		} else {
			coder := ps.ModelPack.Coder
			return coder.GetFinalLargeContextFallback().GetReservedOutputTokens()
		}
	} else {
		return *ps.ModelOverrides.MaxTokens
	}
}

func (ps PlanSettings) GetWholeFileBuilderMaxTokens() int {
	if ps.ModelOverrides.MaxTokens == nil {
		if ps.ModelPack == nil {
			defaultBuilder := DefaultModelPack.WholeFileBuilder
			fallback := defaultBuilder.GetFinalLargeContextFallback()
			return fallback.GetSharedBaseConfig().MaxTokens
		} else {
			builder := ps.ModelPack.WholeFileBuilder
			fallback := builder.GetFinalLargeContextFallback()
			return fallback.GetSharedBaseConfig().MaxTokens
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
			return defaultPlanner.MaxConvoTokens
		} else {
			planner := ps.ModelPack.Planner
			return planner.MaxConvoTokens
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

func (ps PlanSettings) GetCoderEffectiveMaxTokens() int {
	maxCoderTokens := ps.GetCoderMaxTokens()
	maxReservedOutputTokens := ps.GetCoderMaxReservedOutputTokens()

	return maxCoderTokens - maxReservedOutputTokens
}

func (ps PlanSettings) GetWholeFileBuilderEffectiveMaxTokens() int {
	maxWholeFileBuilderTokens := ps.GetWholeFileBuilderMaxTokens()
	maxReservedOutputTokens := ps.GetWholeFileBuilderMaxReservedOutputTokens()

	return maxWholeFileBuilderTokens - maxReservedOutputTokens
}

func (ps PlanSettings) GetModelProviderOptions() ModelProviderOptions {
	opts := ModelProviderOptions{}

	ms := ps.ModelPack
	if ms == nil {
		ms = DefaultModelPack
	}

	opts = opts.Condense(
		ms.Planner.GetModelProviderOptions(),
		ms.Builder.GetModelProviderOptions(),
		ms.PlanSummary.GetModelProviderOptions(),
		ms.Namer.GetModelProviderOptions(),
		ms.CommitMsg.GetModelProviderOptions(),
		ms.ExecStatus.GetModelProviderOptions(),
		// optional roles
		getOptionalModelProviderOptions(ms.WholeFileBuilder),
		getOptionalModelProviderOptions(ms.Architect),
		getOptionalModelProviderOptions(ms.Coder),
	)

	return opts
}

func getOptionalModelProviderOptions(cfg *ModelRoleConfig) ModelProviderOptions {
	if cfg == nil {
		return ModelProviderOptions{}
	}
	return cfg.GetModelProviderOptions()
}
