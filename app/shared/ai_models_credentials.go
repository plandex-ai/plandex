package shared

type ModelProviderOption struct {
	Publishers map[ModelPublisher]bool
	Config     *ModelProviderConfigSchema
	Priority   int
}

type ModelProviderOptions map[string]ModelProviderOption

func (m ModelRoleConfig) GetModelProviderOptions() ModelProviderOptions {
	opts := ModelProviderOptions{}

	usesProviders, ok := BuiltInModelProvidersByModelId[m.ModelId]
	if !ok {
		return opts
	}

	for i, usesProvider := range usesProviders {
		composite := usesProvider.ToComposite()
		config, ok := BuiltInModelProviderConfigs[usesProvider.Provider]
		if !ok {
			continue
		}

		baseModel, ok := BuiltInBaseModelsById[m.ModelId]
		if !ok {
			continue
		}

		opts[composite] = ModelProviderOption{
			Publishers: map[ModelPublisher]bool{
				baseModel.Publisher: true,
			},
			Config:   &config,
			Priority: i,
		}
	}

	if m.ErrorFallback != nil {
		opts = opts.Condense(m.ErrorFallback.GetModelProviderOptions())
	}

	if m.LargeContextFallback != nil {
		opts = opts.Condense(m.LargeContextFallback.GetModelProviderOptions())
	}

	if m.LargeOutputFallback != nil {
		opts = opts.Condense(m.LargeOutputFallback.GetModelProviderOptions())
	}

	if m.StrongModel != nil {
		opts = opts.Condense(m.StrongModel.GetModelProviderOptions())
	}

	return opts
}

func (m ModelProviderOptions) Condense(opts ...ModelProviderOptions) ModelProviderOptions {
	for _, opt := range opts {
		for composite, option := range opt {
			existingOption, exists := m[composite]
			if !exists {
				// first time seeing this composite, add directly
				m[composite] = ModelProviderOption{
					Publishers: copyPublishersMap(option.Publishers),
					Config:     option.Config,
					Priority:   option.Priority,
				}
				continue
			}

			if option.Priority < existingOption.Priority {
				existingOption.Priority = option.Priority
			}

			// composite already exists, merge Publishers
			for pub := range option.Publishers {
				existingOption.Publishers[pub] = true
			}

			// no need to overwrite Config, as it should be identical
			m[composite] = existingOption
		}
	}
	return m
}

func copyPublishersMap(src map[ModelPublisher]bool) map[ModelPublisher]bool {
	cpy := make(map[ModelPublisher]bool, len(src))
	for k, v := range src {
		cpy[k] = v
	}
	return cpy
}
