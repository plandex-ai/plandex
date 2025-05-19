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

func (ps PlanSettings) GetCoderMaxTokens() int {
	if ps.ModelOverrides.MaxTokens == nil {
		if ps.ModelPack == nil {
			defaultCoder := DefaultModelPack.GetCoder()
			return defaultCoder.GetFinalLargeContextFallback().BaseModelConfig.MaxTokens
		} else {
			coder := ps.ModelPack.GetCoder()
			return coder.GetFinalLargeContextFallback().BaseModelConfig.MaxTokens
		}
	} else {
		return *ps.ModelOverrides.MaxTokens
	}
}

func (ps PlanSettings) GetCoderMaxReservedOutputTokens() int {
	if ps.ModelOverrides.MaxTokens == nil {
		if ps.ModelPack == nil {
			defaultCoder := DefaultModelPack.GetCoder()
			return defaultCoder.GetFinalLargeContextFallback().GetReservedOutputTokens()
		} else {
			coder := ps.ModelPack.GetCoder()
			return coder.GetFinalLargeContextFallback().GetReservedOutputTokens()
		}
	} else {
		return *ps.ModelOverrides.MaxTokens
	}
}

func (ps PlanSettings) GetWholeFileBuilderMaxTokens() int {
	if ps.ModelOverrides.MaxTokens == nil {
		if ps.ModelPack == nil {
			defaultBuilder := DefaultModelPack.GetWholeFileBuilder()
			return defaultBuilder.GetFinalLargeContextFallback().BaseModelConfig.MaxTokens
		} else {
			builder := ps.ModelPack.GetWholeFileBuilder()
			return builder.GetFinalLargeContextFallback().BaseModelConfig.MaxTokens
		}
	} else {
		return *ps.ModelOverrides.MaxTokens
	}
}

func (ps PlanSettings) GetWholeFileBuilderMaxReservedOutputTokens() int {
	if ps.ModelOverrides.MaxTokens == nil {
		if ps.ModelPack == nil {
			defaultBuilder := DefaultModelPack.GetWholeFileBuilder()
			return defaultBuilder.GetFinalLargeOutputFallback().GetReservedOutputTokens()
		} else {
			builder := ps.ModelPack.GetWholeFileBuilder()
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

type RequiredEnvVars struct {
	RequiresEither map[string]bool
	RequiresAll    map[string]bool
}

func (ps PlanSettings) GetRequiredEnvVars() RequiredEnvVars {
	envVars := RequiredEnvVars{
		RequiresEither: map[string]bool{},
		RequiresAll:    map[string]bool{},
	}

	ms := ps.ModelPack
	if ms == nil {
		ms = DefaultModelPack
	}

	res := ms.Planner.GetRequiredEnvVars()
	for envVar := range res.RequiresEither {
		envVars.RequiresEither[envVar] = true
	}
	for envVar := range res.RequiresAll {
		envVars.RequiresAll[envVar] = true
	}

	res = ms.Builder.GetRequiredEnvVars()
	for envVar := range res.RequiresEither {
		envVars.RequiresEither[envVar] = true
	}
	for envVar := range res.RequiresAll {
		envVars.RequiresAll[envVar] = true
	}

	if ms.WholeFileBuilder != nil {
		res = ms.WholeFileBuilder.GetRequiredEnvVars()
		for envVar := range res.RequiresEither {
			envVars.RequiresEither[envVar] = true
		}
		for envVar := range res.RequiresAll {
			envVars.RequiresAll[envVar] = true
		}
	}

	res = ms.PlanSummary.GetRequiredEnvVars()
	for envVar := range res.RequiresEither {
		envVars.RequiresEither[envVar] = true
	}
	for envVar := range res.RequiresAll {
		envVars.RequiresAll[envVar] = true
	}

	res = ms.Namer.GetRequiredEnvVars()
	for envVar := range res.RequiresEither {
		envVars.RequiresEither[envVar] = true
	}
	for envVar := range res.RequiresAll {
		envVars.RequiresAll[envVar] = true
	}

	res = ms.CommitMsg.GetRequiredEnvVars()
	for envVar := range res.RequiresEither {
		envVars.RequiresEither[envVar] = true
	}
	for envVar := range res.RequiresAll {
		envVars.RequiresAll[envVar] = true
	}

	res = ms.ExecStatus.GetRequiredEnvVars()
	for envVar := range res.RequiresEither {
		envVars.RequiresEither[envVar] = true
	}
	for envVar := range res.RequiresAll {
		envVars.RequiresAll[envVar] = true
	}

	if ms.Architect != nil {
		res = ms.Architect.GetRequiredEnvVars()
		for envVar := range res.RequiresEither {
			envVars.RequiresEither[envVar] = true
		}
		for envVar := range res.RequiresAll {
			envVars.RequiresAll[envVar] = true
		}
	}

	if ms.Coder != nil {
		res = ms.Coder.GetRequiredEnvVars()
		for envVar := range res.RequiresEither {
			envVars.RequiresEither[envVar] = true
		}
		for envVar := range res.RequiresAll {
			envVars.RequiresAll[envVar] = true
		}
	}

	return envVars
}
