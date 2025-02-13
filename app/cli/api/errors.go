package api

import (
	"encoding/json"
	"log"
	"net/http"
	"plandex-cli/auth"
	"plandex-cli/term"
	"strings"

	shared "plandex-shared"
)

func HandleApiError(r *http.Response, errBody []byte) *shared.ApiError {
	// Check if the response is JSON
	if r.Header.Get("Content-Type") != "application/json" {
		return &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: r.StatusCode,
			Msg:    strings.TrimSpace(string(errBody)),
		}
	}

	var apiError shared.ApiError
	if err := json.Unmarshal(errBody, &apiError); err != nil {
		log.Printf("Error unmarshalling JSON: %v\n", err)
		return &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: r.StatusCode,
			Msg:    strings.TrimSpace(string(errBody)),
		}
	}

	// return error for authentication/retry
	if apiError.Type == shared.ApiErrorTypeInvalidToken {
		return &apiError
	}

	term.HandleApiError(&apiError)

	return &apiError
}

func refreshTokenIfNeeded(apiErr *shared.ApiError) (bool, *shared.ApiError) {
	if apiErr.Type == shared.ApiErrorTypeInvalidToken {
		err := auth.RefreshInvalidToken()
		if err != nil {
			return false, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: "error refreshing invalid token"}
		}
		return true, nil
	}
	return false, apiErr
}
