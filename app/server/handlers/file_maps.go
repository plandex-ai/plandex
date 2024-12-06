package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plandex-server/syntax/file_map"
	"runtime"
	"sync"

	"github.com/plandex/plandex/shared"
)

func GetFileMapHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for GetFileMapHandler")

	var req shared.GetFileMapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding request: %v", err), http.StatusBadRequest)
		return
	}

	maps := make(shared.FileMapBodies)

	// Use half of available CPUs
	cpus := runtime.NumCPU()
	log.Printf("GetFileMapHandler: Available CPUs: %d", cpus)
	maxWorkers := cpus / 2
	if maxWorkers < 1 {
		maxWorkers = 1 // Ensure at least one worker
	}
	log.Printf("GetFileMapHandler: Max workers: %d", maxWorkers)
	sem := make(chan struct{}, maxWorkers)
	wg := sync.WaitGroup{}
	var mu sync.Mutex

	for path, input := range req.MapInputs {
		wg.Add(1)
		sem <- struct{}{}
		go func(path string, input string) {
			defer wg.Done()
			defer func() { <-sem }()
			fileMap, err := file_map.MapFile(r.Context(), path, []byte(input))
			if err != nil {
				// Skip files that can't be parsed, just log the error
				log.Printf("Error mapping file %s: %v", path, err)
				return
			}
			mu.Lock()
			maps[path] = fileMap.String()
			mu.Unlock()
		}(path, input)
	}
	wg.Wait()

	resp := shared.GetFileMapResponse{
		MapBodies: maps,
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error marshalling response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}
