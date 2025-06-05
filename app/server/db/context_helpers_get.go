package db

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
)

func GetPlanContexts(orgId, planId string, includeBody, includeMapParts bool) ([]*Context, error) {
	var contexts []*Context
	contextDir := getPlanContextDir(orgId, planId)

	// get all context files
	files, err := os.ReadDir(contextDir)
	if err != nil {
		if os.IsNotExist(err) {
			return contexts, nil
		}

		return nil, fmt.Errorf("error reading context dir: %v", err)
	}

	errCh := make(chan error, len(files))
	var mu sync.Mutex

	// read each context file
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".meta") {
			go func(file os.DirEntry) {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("panic in GetPlanContexts: %v\n%s", r, debug.Stack())
						errCh <- fmt.Errorf("panic in GetPlanContexts: %v\n%s", r, debug.Stack())
						runtime.Goexit() // don't allow outer function to continue and double-send to channel
					}
				}()
				context, err := GetContext(orgId, planId, strings.TrimSuffix(file.Name(), ".meta"), includeBody, includeMapParts)

				mu.Lock()
				defer mu.Unlock()
				contexts = append(contexts, context)

				if err != nil {
					errCh <- fmt.Errorf("error reading context file: %v", err)
					return
				}

				errCh <- nil
			}(file)
		} else {
			// only processing meta files here, so just send nil for accurate count
			errCh <- nil
		}
	}

	for i := 0; i < len(files); i++ {
		err := <-errCh
		if err != nil {
			return nil, fmt.Errorf("error reading context files: %v", err)
		}
	}

	// sort contexts by CreatedAt
	sort.Slice(contexts, func(i, j int) bool {
		return contexts[i].CreatedAt.Before(contexts[j].CreatedAt)
	})

	return contexts, nil
}

func GetContext(orgId, planId, contextId string, includeBody, includeMapParts bool) (*Context, error) {
	contextDir := getPlanContextDir(orgId, planId)

	// read the meta file
	metaPath := filepath.Join(contextDir, contextId+".meta")

	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("error reading context meta file: %v", err)
	}

	var context Context
	err = json.Unmarshal(metaBytes, &context)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling context meta file: %v", err)
	}

	if includeBody {
		// read the body file
		bodyPath := filepath.Join(contextDir, strings.TrimSuffix(contextId, ".meta")+".body")
		bodyBytes, err := os.ReadFile(bodyPath)

		if err != nil {
			return nil, fmt.Errorf("error reading context body file: %v", err)
		}

		context.Body = string(bodyBytes)
	}

	if includeMapParts {
		// read the map parts file
		mapPartsPath := filepath.Join(contextDir, strings.TrimSuffix(contextId, ".meta")+".map-parts")
		mapPartsBytes, err := os.ReadFile(mapPartsPath)
		if !os.IsNotExist(err) {
			if err != nil {
				return nil, fmt.Errorf("error reading context map parts file: %v", err)
			}

			err = json.Unmarshal(mapPartsBytes, &context.MapParts)
			if err != nil {
				return nil, fmt.Errorf("error unmarshalling context map parts file: %v", err)
			}
		}
	}

	return &context, nil
}
