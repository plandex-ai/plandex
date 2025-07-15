package shared

type ModelProviderOption struct {
	Publishers map[ModelPublisher]bool
	Config     *ModelProviderConfigSchema
	Priority   int
}

type ModelProviderOptions map[string]ModelProviderOption

func (m ModelRoleConfig) GetModelProviderOptions(settings *PlanSettings) ModelProviderOptions {
	opts := ModelProviderOptions{}

	builtInUsesProviders := BuiltInModelProvidersByModelId[m.ModelId]

	var customUsesProviders []BaseModelUsesProvider
	if settings != nil {
		customUsesProviders = settings.UsesCustomProviderByModelId[m.ModelId]
	}

	usesProviders := append(builtInUsesProviders, customUsesProviders...)
	if len(usesProviders) == 0 {
		return opts
	}

	for i, usesProvider := range usesProviders {
		composite := usesProvider.ToComposite()

		foundProvider := false
		config, ok := BuiltInModelProviderConfigs[usesProvider.Provider]
		if ok {
			// built-in provider
			foundProvider = true
		} else if settings != nil && settings.CustomProviders != nil {
			// no built-in provider, check custom providers
			for _, customProvider := range settings.CustomProviders {
				if usesProvider.CustomProvider != nil && customProvider.Name == *usesProvider.CustomProvider {
					config = customProvider.ToModelProviderConfigSchema()
					foundProvider = true
					break
				}
			}
		}

		if !foundProvider {
			continue
		}

		var publisher ModelPublisher

		baseModel, ok := BuiltInBaseModelsById[m.ModelId]
		if ok {
			publisher = baseModel.Publisher
		} else if settings != nil && settings.CustomModelsById != nil {
			customModel, ok := settings.CustomModelsById[m.ModelId]
			if ok {
				publisher = customModel.Publisher
			}
		}

		if publisher == "" {
			continue
		}

		opts[composite] = ModelProviderOption{
			Publishers: map[ModelPublisher]bool{
				publisher: true,
			},
			Config:   &config,
			Priority: i,
		}
	}

	if m.ErrorFallback != nil {
		opts = opts.Condense(m.ErrorFallback.GetModelProviderOptions(settings))
	}

	if m.LargeContextFallback != nil {
		opts = opts.Condense(m.LargeContextFallback.GetModelProviderOptions(settings))
	}

	if m.LargeOutputFallback != nil {
		opts = opts.Condense(m.LargeOutputFallback.GetModelProviderOptions(settings))
	}

	if m.StrongModel != nil {
		opts = opts.Condense(m.StrongModel.GetModelProviderOptions(settings))
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
