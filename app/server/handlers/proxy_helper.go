package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"plandex-server/db"
	"plandex-server/host"
	"time"

	"github.com/plandex/plandex/shared"
)

func proxyActivePlanMethod(w http.ResponseWriter, r *http.Request, planId, branch, method string) {
	modelStream, err := db.GetActiveModelStream(planId, branch)

	if err != nil {
		log.Printf("Error getting active model stream: %v\n", err)
		http.Error(w, "Error getting active model stream", http.StatusInternalServerError)
		return
	}

	if modelStream == nil {
		log.Printf("No active model stream for plan %s\n", planId)
		http.Error(w, "No active model stream for plan", http.StatusNotFound)
		return
	}

	if modelStream.InternalIp == host.Ip {
		// No active plan for this plan or else we wouldn't be calling proxyActivePlanMethod -- set the model stream to finished because something went wrong
		err := db.SetModelStreamFinished(modelStream.Id)
		if err != nil {
			log.Printf("Error setting model stream %s to finished: %v\n", modelStream.Id, err)
		}

		err = db.SetPlanStatus(planId, branch, shared.PlanStatusError, "No active stream for plan")
		if err != nil {
			log.Printf("Error setting plan %s status to error: %v\n", planId, err)
		}

		log.Printf("No active plan for plan %s\n", planId)
		http.Error(w, "No active plan for plan", http.StatusNotFound)
		return
	} else {
		log.Printf("Forwarding request to %s\n", modelStream.InternalIp)
		proxyUrl := fmt.Sprintf("http://%s:%s/plans/%s/%s/%s", modelStream.InternalIp, os.Getenv("PORT"), planId, branch, method)
		proxyUrl += "?proxy=true"

		log.Printf("Proxy url: %s\n", proxyUrl)
		proxyRequest(w, r, proxyUrl)
		return
	}
}

func proxyRequest(w http.ResponseWriter, originalRequest *http.Request, url string) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

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

	// Copy the body from the original request to the new request if it's a POST or PUT
	if originalRequest.Method == http.MethodPost || originalRequest.Method == http.MethodPut {
		req.Body = originalRequest.Body
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
