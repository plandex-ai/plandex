package shared

const maxFallbackDepth = 10 // max fallback depth for large context fallback - should never be reached in real scenarios, but protects against infinite loops in case of circular references etc.

func (m ModelRoleConfig) GetFinalLargeContextFallback() ModelRoleConfig {
	var currentConfig ModelRoleConfig = m
	var n int = 0

	for {
		if currentConfig.LargeContextFallback == nil {
			return currentConfig
		} else {
			currentConfig = *currentConfig.LargeContextFallback
		}
		n++
		if n > maxFallbackDepth {
			break
		}
	}

	return currentConfig
}

func (m ModelRoleConfig) GetFinalLargeOutputFallback() ModelRoleConfig {
	var currentConfig ModelRoleConfig = m
	var n int = 0

	if currentConfig.LargeOutputFallback == nil {
		return currentConfig.GetFinalLargeContextFallback()
	}

	for {
		if currentConfig.LargeOutputFallback == nil {
			return currentConfig
		} else {
			currentConfig = *currentConfig.LargeOutputFallback
		}
		n++
		if n > maxFallbackDepth {
			break
		}
	}

	return currentConfig
}

// note that if the token number exeeds all the fallback models, it will return the last fallback model

func (m ModelRoleConfig) GetRoleForInputTokens(inputTokens int) ModelRoleConfig {
	inputTokens = int(float64(inputTokens) * (1 + m.BaseModelConfig.TokenEstimatePaddingPct))
	var currentConfig ModelRoleConfig = m
	var n int = 0
	for {
		if currentConfig.BaseModelConfig.MaxTokens >= inputTokens {
			return currentConfig
		}

		if currentConfig.LargeContextFallback == nil {
			return currentConfig
		} else {
			currentConfig = *currentConfig.LargeContextFallback
		}
		n++
		if n > maxFallbackDepth {
			break
		}
	}
	return currentConfig
}

func (m ModelRoleConfig) GetRoleForOutputTokens(outputTokens int) ModelRoleConfig {
	outputTokens = int(float64(outputTokens) * (1 + m.BaseModelConfig.TokenEstimatePaddingPct))
	var currentConfig ModelRoleConfig = m
	var n int = 0
	for {
		if currentConfig.GetReservedOutputTokens() >= outputTokens {
			return currentConfig
		}

		if currentConfig.LargeOutputFallback == nil {
			if currentConfig.LargeContextFallback == nil {
				return currentConfig
			} else {
				currentConfig = *currentConfig.LargeContextFallback
			}
		} else {
			currentConfig = *currentConfig.LargeOutputFallback
		}
		n++
		if n > maxFallbackDepth {
			break
		}
	}
	return currentConfig
}
