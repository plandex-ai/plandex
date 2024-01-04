package api

import (
	"net"
	"net/http"
	"os"
	"plandex/auth"
	"plandex/types"
	"time"
)

const dialTimeout = 10 * time.Second
const fastReqTimeout = 30 * time.Second
const slowReqTimeout = 5 * time.Minute

type Api struct{}

var apiHost string

var Client types.ApiClient = (*Api)(nil)

func init() {
	var port = os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}
	apiHost = "http://localhost:" + port
}

type authenticatedTransport struct {
	underlyingTransport http.RoundTripper
}

// RoundTrip executes a single HTTP transaction and adds a custom header
func (t *authenticatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	auth.SetAuthHeader(req)
	return t.underlyingTransport.RoundTrip(req)
}

var netDialer = &net.Dialer{
	Timeout: dialTimeout,
}

var unauthenticatedClient = &http.Client{
	Transport: &http.Transport{
		Dial: netDialer.Dial,
	},
	Timeout: fastReqTimeout,
}

var authenticatedFastClient = &http.Client{
	Transport: &authenticatedTransport{
		underlyingTransport: &http.Transport{
			Dial: netDialer.Dial,
		},
	},
	Timeout: fastReqTimeout,
}

var authenticatedSlowClient = &http.Client{
	Transport: &authenticatedTransport{
		underlyingTransport: &http.Transport{
			Dial: netDialer.Dial,
		},
	},
	Timeout: slowReqTimeout,
}

var authenticatedStreamingClient = &http.Client{
	Transport: &authenticatedTransport{
		underlyingTransport: &http.Transport{
			Dial: netDialer.Dial,
		},
	},
	// No global timeout set for the streaming client
}
