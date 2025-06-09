package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"runtime"
	"runtime/debug"
	"sync"

	shared "plandex-shared"

	"github.com/gorilla/mux"
)

func GetFileMapHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for GetFileMapHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		log.Println("GetFileMapHandler: auth failed")
		return
	}

	var req shared.GetFileMapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding request: %v", err), http.StatusBadRequest)
		return
	}

	log.Println("GetFileMapHandler: checking limits")

	if len(req.MapInputs) > shared.MaxContextMapPaths {
		http.Error(w, fmt.Sprintf("Too many files to map: %d (max %d)", len(req.MapInputs), shared.MaxContextMapPaths), http.StatusBadRequest)
		return
	}

	totalSize := 0
	for path, input := range req.MapInputs {
		// the client should be truncating inputs to the max size, but we'll check here too
		if len(input) > shared.MaxContextMapSingleInputSize {
			http.Error(w, fmt.Sprintf("File %s is too large: %d (max %d)", path, len(input), shared.MaxContextMapSingleInputSize), http.StatusBadRequest)
			return
		}
		totalSize += len(input)
	}

	// On the client, once the total size limit is exceeded, we send empty file maps for remaining files
	if totalSize > shared.MaxContextMapTotalInputSize+10000 {
		http.Error(w, fmt.Sprintf("Max map size exceeded: %d (max %d)", totalSize, shared.MaxContextMapTotalInputSize), http.StatusBadRequest)
		return
	}

	// Check batch size limits
	if len(req.MapInputs) > shared.ContextMapMaxBatchSize {
		http.Error(w, fmt.Sprintf("Batch contains too many files: %d (max %d)", len(req.MapInputs), shared.ContextMapMaxBatchSize), http.StatusBadRequest)
		return
	}

	if int64(totalSize) > shared.ContextMapMaxBatchBytes {
		http.Error(w, fmt.Sprintf("Batch size too large: %d bytes (max %d bytes)", totalSize, shared.ContextMapMaxBatchBytes), http.StatusBadRequest)
		return
	}

	results := make(chan shared.FileMapBodies, 1)

	err := queueProjectMapJob(projectMapJob{
		inputs:  req.MapInputs,
		ctx:     r.Context(),
		results: results,
	})
	if err != nil {
		log.Println("GetFileMapHandler: map queue is full")
		http.Error(w, "Too many project map jobs, please try again later", http.StatusTooManyRequests)
		return
	}

	select {
	case <-r.Context().Done():
		http.Error(w, "Request was cancelled", http.StatusRequestTimeout)
		return
	case maps := <-results:
		if maps == nil {
			http.Error(w, "Mapping timed out", http.StatusRequestTimeout)
			return
		}

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

		log.Printf("GetFileMapHandler success - writing response bytes: %d", len(respBytes))
	}
}

func LoadCachedFileMapHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for LoadCachedFileMapHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branchName := vars["branch"]
	log.Println("planId: ", planId, "branchName: ", branchName)

	plan := authorizePlan(w, planId, auth)

	if plan == nil {
		return
	}

	var req shared.LoadCachedFileMapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding request: %v", err), http.StatusBadRequest)
		return
	}

	cachedMetaByPath := map[string]*shared.Context{}
	cachedMapsByPath := map[string]*db.CachedMap{}
	var mu sync.Mutex
	errCh := make(chan error, len(req.FilePaths))

	for _, path := range req.FilePaths {
		go func(path string) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in LoadCachedFileMapHandler: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in LoadCachedFileMapHandler: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()

			cachedContext, err := db.GetCachedMap(plan.OrgId, plan.ProjectId, path)
			if err != nil {
				errCh <- fmt.Errorf("error getting cached map: %v", err)
				return
			}
			if cachedContext != nil {
				mu.Lock()
				cachedMetaByPath[path] = cachedContext.ToMeta().ToApi()
				cachedMapsByPath[path] = &db.CachedMap{
					MapParts:  cachedContext.MapParts,
					MapShas:   cachedContext.MapShas,
					MapTokens: cachedContext.MapTokens,
					MapSizes:  cachedContext.MapSizes,
				}
				mu.Unlock()
			}
			errCh <- nil
		}(path)
	}

	for range req.FilePaths {
		err := <-errCh
		if err != nil {
			log.Printf("Error getting cached map: %v", err)
			http.Error(w, fmt.Sprintf("Error getting cached map: %v", err), http.StatusInternalServerError)
			return
		}
	}

	resp := shared.LoadCachedFileMapResponse{}

	var loadRes *shared.LoadContextResponse
	if len(cachedMetaByPath) == 0 {
		log.Println("no cached maps found")
	} else {
		log.Println("cached map found")

		cachedByPath := map[string]bool{}
		for _, cachedContext := range cachedMetaByPath {
			cachedByPath[cachedContext.FilePath] = true
		}
		resp.CachedByPath = cachedByPath

		var loadReq shared.LoadContextRequest
		for _, cachedContext := range cachedMetaByPath {
			loadReq = append(loadReq, &shared.LoadContextParams{
				ContextType: shared.ContextMapType,
				Name:        cachedContext.Name,
				FilePath:    cachedContext.FilePath,
				Body:        cachedContext.Body,
			})
		}

		loadRes, _ = loadContexts(loadContextsParams{
			w:                w,
			r:                r,
			auth:             auth,
			loadReq:          &loadReq,
			plan:             plan,
			branchName:       branchName,
			cachedMapsByPath: cachedMapsByPath,
		})

		if loadRes == nil {
			log.Println("LoadCachedFileMapHandler - loadRes is nil")
			return
		}

		resp.LoadRes = loadRes
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error marshalling response: %v", err)
		http.Error(w, fmt.Sprintf("Error marshalling response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(bytes)
}
