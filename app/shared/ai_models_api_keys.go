package shared

func (m ModelRoleConfig) GetRequiredEnvVars() RequiredEnvVars {
	envVars := RequiredEnvVars{
		RequiresEither: map[string]bool{},
		RequiresAll:    map[string]bool{},
	}

	if m.MissingKeyFallback != nil {
		envVars.RequiresEither[m.BaseModelConfig.ApiKeyEnvVar] = true
		envVars.RequiresEither[m.MissingKeyFallback.BaseModelConfig.ApiKeyEnvVar] = true
	} else {
		envVars.RequiresAll[m.BaseModelConfig.ApiKeyEnvVar] = true
	}

	if m.ErrorFallback != nil {
		res := m.ErrorFallback.GetRequiredEnvVars()
		for envVar := range res.RequiresEither {
			envVars.RequiresEither[envVar] = true
		}
		for envVar := range res.RequiresAll {
			envVars.RequiresAll[envVar] = true
		}
	}

	if m.LargeContextFallback != nil {
		res := m.LargeContextFallback.GetRequiredEnvVars()
		for envVar := range res.RequiresEither {
			envVars.RequiresEither[envVar] = true
		}
		for envVar := range res.RequiresAll {
			envVars.RequiresAll[envVar] = true
		}
	}

	if m.LargeOutputFallback != nil {
		res := m.LargeOutputFallback.GetRequiredEnvVars()
		for envVar := range res.RequiresEither {
			envVars.RequiresEither[envVar] = true
		}
		for envVar := range res.RequiresAll {
			envVars.RequiresAll[envVar] = true
		}
	}

	if m.StrongModel != nil {
		res := m.StrongModel.GetRequiredEnvVars()
		for envVar := range res.RequiresEither {
			envVars.RequiresEither[envVar] = true
		}
		for envVar := range res.RequiresAll {
			envVars.RequiresAll[envVar] = true
		}
	}

	return envVars
}

func (m ModelRoleConfig) BaseModelConfigForEnvVar(envVar string) *BaseModelConfig {
	if envVar == m.BaseModelConfig.ApiKeyEnvVar {
		return &m.BaseModelConfig
	}

	if m.MissingKeyFallback != nil {
		return m.MissingKeyFallback.BaseModelConfigForEnvVar(envVar)
	}

	return nil
}
