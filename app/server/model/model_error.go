package model

import (
	"fmt"
	"log"
	"net/http"
	shared "plandex-shared"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type HTTPError struct {
	StatusCode int
	Body       string
	Header     http.Header
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("status code: %d, body: %s", e.StatusCode, e.Body)
}

// JSON-style  `"retry_after_ms":1234`
var reJSON = regexp.MustCompile(`"retry_after_ms"\s*:\s*(\d+)`)

// Header- or text-style  "Retry-After: 12" / "retry_after: 12s"
var reRetryAfter = regexp.MustCompile(
	`retry[_\-\s]?after[_\-\s]?(?:[:\s]+)?(\d+)(ms|seconds?|secs?|s)?`,
)

// Free-form Azure style  "Try again in 59 seconds."
// Also matches "Retry in 10 seconds."
var reTryAgain = regexp.MustCompile(
	`(?:re)?try[_\-\s]+(?:again[_\-\s]+)?in[_\-\s]+(\d+)(ms|seconds?|secs?|s)?`,
)

func ClassifyErrMsg(msg string) *shared.ModelError {
	log.Printf("Classifying error message: %s", msg)

	msg = strings.ToLower(msg)

	if strings.Contains(msg, "maximum context length") ||
		strings.Contains(msg, "context length exceeded") ||
		strings.Contains(msg, "exceed context limit") ||
		strings.Contains(msg, "decrease input length") ||
		strings.Contains(msg, "too many tokens") ||
		strings.Contains(msg, "payload too large") ||
		strings.Contains(msg, "payload is too large") ||
		strings.Contains(msg, "input is too large") ||
		strings.Contains(msg, "input too large") ||
		strings.Contains(msg, "input is too long") ||
		strings.Contains(msg, "input too long") {
		log.Printf("Context too long error: %s", msg)
		return &shared.ModelError{
			Kind:              shared.ErrContextTooLong,
			Retriable:         false,
			RetryAfterSeconds: 0,
		}
	}

	if strings.Contains(msg, "model_overloaded") ||
		strings.Contains(msg, "model overloaded") ||
		strings.Contains(msg, "server is overloaded") ||
		strings.Contains(msg, "model is currently overloaded") ||
		strings.Contains(msg, "overloaded_error") ||
		strings.Contains(msg, "resource has been exhausted") {
		log.Printf("Overloaded error: %s", msg)
		return &shared.ModelError{
			Kind:              shared.ErrOverloaded,
			Retriable:         true,
			RetryAfterSeconds: 0,
		}
	}

	if strings.Contains(msg, "cache control") {
		log.Printf("Cache control error: %s", msg)
		return &shared.ModelError{
			Kind:              shared.ErrCacheSupport,
			Retriable:         true,
			RetryAfterSeconds: 0,
		}
	}

	log.Println("No error classification based on message")

	return nil
}

func ClassifyModelError(code int, message string, headers http.Header) shared.ModelError {
	msg := strings.ToLower(message)

	// first try to classify the error based on the message only
	msgRes := ClassifyErrMsg(msg)
	if msgRes != nil {
		log.Printf("Classified error message: %+v", msgRes)
		return *msgRes
	}

	var res shared.ModelError

	switch code {
	case 429, 529:
		res = shared.ModelError{
			Kind:              shared.ErrRateLimited,
			Retriable:         true,
			RetryAfterSeconds: 0,
		}
	case 413:
		res = shared.ModelError{
			Kind:              shared.ErrContextTooLong,
			Retriable:         false,
			RetryAfterSeconds: 0,
		}

	// rare codes but they never succeed on retry if they do show up
	case 501, 505:
		res = shared.ModelError{
			Kind:              shared.ErrOther,
			Retriable:         false,
			RetryAfterSeconds: 0,
		}
	default:
		res = shared.ModelError{
			Kind:              shared.ErrOther,
			Retriable:         code >= 500 || strings.Contains(msg, "provider returned error"), // 'provider returned error' is from OpenRouter, and unless it's a non-retriable status code, it should still be retried since OpenRouter may switch to a different provider
			RetryAfterSeconds: 0,
		}
	}

	log.Printf("Model error: %+v", res)

	// best‑effort parse of "Retry‑After" style hints in the message
	if res.Retriable {
		retryAfter := extractRetryAfter(headers, msg)

		// if the retry after is greater than the max delay, then the error is not retriable
		if retryAfter > MAX_RETRY_DELAY_SECONDS {
			log.Printf("Retry after %d seconds is greater than the max delay of %d seconds - not retriable", retryAfter, MAX_RETRY_DELAY_SECONDS)
			res.Retriable = false
		} else {
			res.RetryAfterSeconds = retryAfter
		}

	}

	return res
}

func extractRetryAfter(h http.Header, body string) (sec int) {
	now := time.Now()

	// Retry-After header: seconds or HTTP-date
	if v := h.Get("Retry-After"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
		if t, err := time.Parse(http.TimeFormat, v); err == nil {
			d := int(t.Sub(now).Seconds())
			if d > 0 {
				return d
			}
		}
	}

	// X-RateLimit-Reset epoch
	if v := h.Get("X-RateLimit-Reset"); v != "" {
		if reset, _ := strconv.ParseInt(v, 10, 64); reset > now.Unix() {
			return int(reset - now.Unix())
		}
	}

	lower := strings.ToLower(strings.TrimSpace(body))

	// "retry_after_ms": 1234
	if m := reJSON.FindStringSubmatch(lower); len(m) == 2 {
		n, _ := strconv.Atoi(m[1])
		return n / 1000
	}
	// "retry after 12"
	if m := reRetryAfter.FindStringSubmatch(lower); len(m) >= 2 {
		unit := ""
		if len(m) == 3 {
			unit = m[2]
		}
		return normalizeUnit(m[1], unit)
	}

	// "try again in 8"
	if m := reTryAgain.FindStringSubmatch(lower); len(m) >= 2 {
		unit := ""
		if len(m) == 3 {
			unit = m[2]
		}
		return normalizeUnit(m[1], unit)
	}
	return 0
}

func normalizeUnit(numStr, unit string) int {
	n, _ := strconv.Atoi(numStr) // safe because the regex matched \d+

	switch unit {
	case "ms": // milliseconds
		return n / 1000
	case "sec", "secs", "second", "seconds", "s":
		return n // already in seconds
	default: // unit omitted ⇒ assume seconds
		return n
	}
}

func classifyBasicError(err error) shared.ModelError {
	// if it's an http error, classify it based on the status code and body
	if httpErr, ok := err.(*HTTPError); ok {
		me := ClassifyModelError(
			httpErr.StatusCode,
			httpErr.Body,
			httpErr.Header,
		)
		return me
	}

	// try to classify the error based on the message only
	msgRes := ClassifyErrMsg(err.Error())
	if msgRes != nil {
		return *msgRes
	}

	// Fall back to old heuristic – still keeps the signature identical
	if isNonRetriableBasicErr(err) {
		return shared.ModelError{Kind: shared.ErrOther, Retriable: false}
	}
	return shared.ModelError{Kind: shared.ErrOther, Retriable: true}
}

func isNonRetriableBasicErr(err error) bool {
	errStr := err.Error()

	// we don't want to retry on the errors below
	if strings.Contains(errStr, "context deadline exceeded") || strings.Contains(errStr, "context canceled") {
		log.Println("Context deadline exceeded or canceled - no retry")
		return true
	}

	if strings.Contains(errStr, "status code: 400") &&
		strings.Contains(errStr, "reduce the length of the messages") {
		log.Println("Token limit exceeded - no retry")
		return true
	}

	if strings.Contains(errStr, "status code: 401") {
		log.Println("Invalid auth or api key - no retry")
		return true
	}

	if strings.Contains(errStr, "status code: 429") && strings.Contains(errStr, "exceeded your current quota") {
		log.Println("Current quota exceeded - no retry")
		return true
	}

	return false
}
