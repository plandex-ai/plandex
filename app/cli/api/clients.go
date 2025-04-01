package api

import (
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"plandex-cli/auth"
	"plandex-cli/types"
	"time"
)

const dialTimeout = 10 * time.Second
const fastReqTimeout = 30 * time.Second
const slowReqTimeout = 5 * time.Minute

type Api struct{}

var CloudApiHost string
var Client types.ApiClient = (*Api)(nil)

func init() {
	if os.Getenv("PLANDEX_ENV") == "development" {
		CloudApiHost = os.Getenv("PLANDEX_API_HOST")
		if CloudApiHost == "" {
			CloudApiHost = "http://localhost:8099"
		}
	} else {
		CloudApiHost = "https://api-v2.plandex.ai"
	}
}

func GetApiHost() string {
	if auth.Current == nil {
		return CloudApiHost
	} else if auth.Current.IsCloud {
		return CloudApiHost
	} else {
		return auth.Current.Host
	}
}

type authenticatedTransport struct {
	underlyingTransport http.RoundTripper
}

func (t *authenticatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	err := auth.SetAuthHeader(req)
	if err != nil {
		return nil, err
	}
	return t.underlyingTransport.RoundTrip(req)
}

type unauthenticatedTransport struct {
	underlyingTransport http.RoundTripper
}

func (t *unauthenticatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.underlyingTransport.RoundTrip(req)
}

type retryTransport struct {
	Base          http.RoundTripper
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	Jitter        time.Duration
	RetryStatuses map[int]bool
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Base == nil {
		t.Base = http.DefaultTransport
	}
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= t.MaxRetries; attempt++ {
		resp, err = t.Base.RoundTrip(req)

		// If there's a low-level error (e.g. network), retry unless it's a timeout, as these are often transient.
		if err != nil {
			if netErr, ok := err.(net.Error); ok {
				if netErr.Timeout() {
					return resp, err
				}
			}
			// continue to next attempt
		} else {
			// If status code not in our RetryStatuses, return immediately.
			if !t.RetryStatuses[resp.StatusCode] {
				return resp, nil
			}

			// Close the body before retrying.
			_ = resp.Body.Close()
		}

		// If we reached the max, break out of loop (will return last resp).
		if attempt == t.MaxRetries {
			break
		}

		// Exponential backoff + jitter
		backoff := float64(t.BaseDelay) * math.Pow(2, float64(attempt))
		if backoff > float64(t.MaxDelay) {
			backoff = float64(t.MaxDelay)
		}
		sleepDuration := time.Duration(backoff) + time.Duration(rand.Int63n(int64(t.Jitter)))
		time.Sleep(sleepDuration)
	}
	return resp, err
}

var netDialer = &net.Dialer{
	Timeout: dialTimeout,
}

var baseTransport = &http.Transport{
	Dial: netDialer.Dial,
}

var sharedRetryTransport = &retryTransport{
	Base:          baseTransport,
	MaxRetries:    3,
	BaseDelay:     500 * time.Millisecond,
	MaxDelay:      5 * time.Second,
	Jitter:        300 * time.Millisecond,
	RetryStatuses: map[int]bool{502: true, 503: true, 504: true},
}

var unauthenticatedClient = &http.Client{
	Transport: &unauthenticatedTransport{
		underlyingTransport: sharedRetryTransport,
	},
	Timeout: fastReqTimeout,
}

var authenticatedFastClient = &http.Client{
	Transport: &authenticatedTransport{
		underlyingTransport: sharedRetryTransport,
	},
	Timeout: fastReqTimeout,
}

var authenticatedSlowClient = &http.Client{
	Transport: &authenticatedTransport{
		underlyingTransport: sharedRetryTransport,
	},
	Timeout: slowReqTimeout,
}

var authenticatedStreamingClient = &http.Client{
	Transport: &authenticatedTransport{
		underlyingTransport: sharedRetryTransport,
	},
}
