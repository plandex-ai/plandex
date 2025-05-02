package shared

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
	FallbackTypeError   FallbackType = "error"
	FallbackTypeContext FallbackType = "context"
)

type FallbackResult struct {
	ModelRoleConfig  *ModelRoleConfig
	HasErrorFallback bool
	IsFallback       bool
	FallbackType     FallbackType
}

const MAX_RETRIES_BEFORE_FALLBACK = 1

func (m *ModelRoleConfig) GetFallbackForModelError(numTotalRetry int, modelErr *ModelError) FallbackResult {
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
		}
	}

	return FallbackResult{
		ModelRoleConfig: m,
		IsFallback:      false,
	}
}
