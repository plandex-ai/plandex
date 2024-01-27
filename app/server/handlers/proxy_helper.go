package handlers

import (
	"io"
	"log"
	"net/http"
)

func proxyRequest(w http.ResponseWriter, originalRequest *http.Request, url string) {
	client := &http.Client{}

	// Create a new request based on the original request
	req, err := http.NewRequest(originalRequest.Method, url, originalRequest.Body)
	if err != nil {
		log.Printf("Error creating request for proxy: %v\n", err)
		http.Error(w, "Error creating request for proxy", http.StatusInternalServerError)
		return
	}

	// Copy the headers from the original request to the new request
	for name, headers := range originalRequest.Header {
		for _, h := range headers {
			req.Header.Add(name, h)
		}
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error forwarding request: %v\n", err)
		http.Error(w, "Error forwarding request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy the response headers and status code
	for name, headers := range resp.Header {
		for _, h := range headers {
			w.Header().Add(name, h)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Copy the response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Error copying response body: %v\n", err)
		http.Error(w, "Error copying response body", http.StatusInternalServerError)
	}
}
