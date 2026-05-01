package model

import (
	"context"
	"net/http"
	"testing"

	shared "plandex-shared"
)

func TestWithStreamingRetriesHandlesMissingBaseModelConfig(t *testing.T) {
	attempts := 0
	overloadedErr := &HTTPError{
		StatusCode: http.StatusTooManyRequests,
		Body:       `{"type":"error","error":{"type":"overloaded_error","message":"The server cluster is currently under high load"}}`,
	}

	_, err := withStreamingRetries(context.Background(), func(numRetry int, didProviderFallback bool, modelErr *shared.ModelError) (*string, shared.FallbackResult, error) {
		attempts++
		return nil, shared.FallbackResult{
			ModelRoleConfig: &shared.ModelRoleConfig{},
			IsFallback:      false,
			BaseModelConfig: nil,
		}, overloadedErr
	}, func(resp *string, err error) {})

	if err == nil {
		t.Fatal("expected overloaded error after retries are exhausted")
	}
	if attempts != MAX_RETRIES_WITHOUT_FALLBACK+1 {
		t.Fatalf("expected %d attempts, got %d", MAX_RETRIES_WITHOUT_FALLBACK+1, attempts)
	}
}
