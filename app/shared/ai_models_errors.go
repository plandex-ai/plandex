package shared

import (
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/jinzhu/copier"
)

type ModelErrKind string

const (
	ErrOverloaded     ModelErrKind = "ErrOverloaded"
	ErrContextTooLong ModelErrKind = "ErrContextTooLong"
	ErrRateLimited    ModelErrKind = "ErrRateLimited"
	ErrOther          ModelErrKind = "ErrOther"
	ErrCacheSupport   ModelErrKind = "ErrCacheSupport"
)

type ModelError struct {
	Kind              ModelErrKind
	Retriable         bool
	RetryAfterSeconds int
}

// if fallback is defined, retry with main model, then remaining tries use error fallback
type FallbackType string

const (
	FallbackTypeError    FallbackType = "error"
	FallbackTypeContext  FallbackType = "context"
	FallbackTypeProvider FallbackType = "provider"
)

type FallbackResult struct {
	ModelRoleConfig  *ModelRoleConfig
	HasErrorFallback bool
	IsFallback       bool
	FallbackType     FallbackType
}

const MAX_RETRIES_BEFORE_FALLBACK = 1

func (m *ModelRoleConfig) GetFallbackForModelError(
	numTotalRetry int,
	didProviderFallback bool,
	modelErr *ModelError,
	authVars map[string]string,
	localProvider ModelProvider,
) FallbackResult {
	if m == nil || modelErr == nil {
		return FallbackResult{
			ModelRoleConfig: m,
			IsFallback:      false,
		}
	}
	if modelErr.Kind == ErrContextTooLong {
		if m.LargeContextFallback != nil {
			return FallbackResult{
				ModelRoleConfig: m.LargeContextFallback,
				FallbackType:    FallbackTypeContext,
				IsFallback:      true,
			}
		}
	} else if !modelErr.Retriable || numTotalRetry > MAX_RETRIES_BEFORE_FALLBACK {
		if m.ErrorFallback != nil {
			return FallbackResult{
				ModelRoleConfig: m.ErrorFallback,
				FallbackType:    FallbackTypeError,
				IsFallback:      true,
			}
		} else if !didProviderFallback {
			log.Println("no error fallback, trying provider fallback")

			providerFallback := m.GetProviderFallback(authVars, localProvider)

			log.Println(spew.Sdump(map[string]interface{}{
				"providerFallback": providerFallback,
			}))

			if providerFallback != nil {
				return FallbackResult{
					ModelRoleConfig: providerFallback,
					FallbackType:    FallbackTypeProvider,
					IsFallback:      true,
				}
			}
		}
	}

	return FallbackResult{
		ModelRoleConfig: m,
		IsFallback:      false,
	}
}

// we just try a single provider fallback if all defined fallbacks are exhausted
// if we've got openrouter credentials in the stack, we always use OpenRouter as the fallback since it has its own routing/fallback routing to maximize resilience
// otherwise we just use the second provider in the stack
func (m ModelRoleConfig) GetProviderFallback(authVars map[string]string, localProvider ModelProvider) *ModelRoleConfig {
	providers := m.GetProvidersForAuthVars(authVars, localProvider)

	if len(providers) < 2 {
		return nil
	}

	res := ModelRoleConfig{}
	copier.Copy(&res, m)

	var provider ModelProvider
	for _, p := range providers {
		if p.Provider == ModelProviderOpenRouter {
			provider = p.Provider
			break
		}
	}

	if provider == "" {
		provider = providers[1].Provider
	}

	availableModel := GetAvailableModel(provider, m.ModelId)

	if availableModel == nil {
		log.Printf("no available model found for provider %s and model id %s", provider, m.ModelId)
		return nil
	}

	c := availableModel.BaseModelConfig
	res.BaseModelConfig = &c

	return &res
}
