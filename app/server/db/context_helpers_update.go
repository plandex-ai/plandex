package db

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	shared "plandex-shared"
	"runtime"
	"runtime/debug"
	"sync"
)

type UpdateContextsParams struct {
	Req                      *shared.UpdateContextRequest
	OrgId                    string
	Plan                     *Plan
	BranchName               string
	ContextsById             map[string]*Context
	SkipConflictInvalidation bool
}

func UpdateContexts(params UpdateContextsParams) (*shared.UpdateContextResponse, error) {
	req := params.Req
	orgId := params.OrgId
	plan := params.Plan
	planId := plan.Id
	branchName := params.BranchName

	branch, err := GetDbBranch(planId, branchName)
	if err != nil {
		return nil, fmt.Errorf("error getting branch: %v", err)
	}

	if branch == nil {
		return nil, fmt.Errorf("branch not found")
	}

	totalTokens := branch.ContextTokens
	totalPlannerTokens := totalTokens

	totalMapTokens := 0

	totalBasicPlannerTokens := 0
	for _, context := range params.ContextsById {
		if context.ContextType != shared.ContextMapType && !context.AutoLoaded {
			totalBasicPlannerTokens += context.NumTokens
		}
	}

	settings, err := GetPlanSettings(plan, true)
	if err != nil {
		return nil, fmt.Errorf("error getting settings: %v", err)
	}

	planConfig, err := GetPlanConfig(planId)
	if err != nil {
		return nil, fmt.Errorf("error getting plan config: %v", err)
	}

	plannerMaxTokens := settings.GetPlannerEffectiveMaxTokens()
	contextLoaderMaxTokens := settings.GetArchitectEffectiveMaxTokens()

	if planConfig.AutoLoadContext {
		existingContexts, err := GetPlanContexts(orgId, planId, false, false)
		if err != nil {
			return nil, fmt.Errorf("error getting existing contexts: %v", err)
		}

		for _, context := range existingContexts {
			if context.ContextType == shared.ContextMapType {
				totalMapTokens += context.NumTokens
				totalPlannerTokens -= context.NumTokens
			}
		}
	}

	aggregateTokensDiff := 0
	aggregateBasicTokensDiff := 0
	tokenDiffsById := make(map[string]int)

	var contextsById map[string]*Context
	if params.ContextsById == nil {
		contextsById = make(map[string]*Context)
	} else {
		contextsById = params.ContextsById
	}

	var totalContextCount int
	var totalBodySize int64

	for _, context := range contextsById {
		totalContextCount++
		totalBodySize += context.BodySize
	}

	for id, params := range *req {
		size := int64(len(params.Body))

		if size > shared.MaxContextBodySize {
			return nil, fmt.Errorf("context body is too large: %d", size)
		}

		if context, ok := contextsById[id]; ok {
			totalBodySize += size - context.BodySize
		} else {
			totalContextCount++
			totalBodySize += size
		}
	}

	if totalContextCount > shared.MaxContextCount {
		return nil, fmt.Errorf("too many contexts to update (found %d, limit is %d)", totalContextCount, shared.MaxContextCount)
	}

	if totalBodySize > shared.MaxContextBodySize {
		return nil, fmt.Errorf("total context body size exceeds limit (size %.2f MB, limit %d MB)", float64(totalBodySize)/1024/1024, int(shared.MaxContextBodySize)/1024/1024)
	}

	var updatedContexts []*shared.Context

	numFiles := 0
	numUrls := 0
	numTrees := 0
	numMaps := 0

	var mu sync.Mutex
	errCh := make(chan error, len(*req))

	for id, params := range *req {
		go func(id string, params *shared.UpdateContextParams) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in UpdateContexts: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in UpdateContexts: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			var context *Context
			if _, ok := contextsById[id]; ok {
				context = contextsById[id]
			} else {
				var err error
				context, err = GetContext(orgId, planId, id, true, true)

				if err != nil {
					errCh <- fmt.Errorf("error getting context: %v", err)
					return
				}
				// log.Println("Got context", context.Id, "numTokens", context.NumTokens)
			}

			mu.Lock()
			defer mu.Unlock()

			contextsById[id] = context
			updatedContexts = append(updatedContexts, context.ToApi())

			if context.ContextType != shared.ContextMapType {
				var updateNumTokens int
				var err error

				if context.ContextType == shared.ContextImageType {
					updateNumTokens, err = shared.GetImageTokens(params.Body, context.ImageDetail)
					if err != nil {
						errCh <- fmt.Errorf("error getting num tokens: %v", err)
						return
					}
				} else {
					updateNumTokens = shared.GetNumTokensEstimate(params.Body)
					// log.Println("len(params.Body)", len(params.Body))
				}

				// log.Println("Updating context", id, "updateNumTokens", updateNumTokens)

				tokenDiff := updateNumTokens - context.NumTokens
				tokenDiffsById[id] = tokenDiff
				aggregateTokensDiff += tokenDiff
				totalTokens += tokenDiff
				totalPlannerTokens += tokenDiff
				if !context.AutoLoaded {
					totalBasicPlannerTokens += tokenDiff
					aggregateBasicTokensDiff += tokenDiff
				}
				context.NumTokens = updateNumTokens
			}

			switch context.ContextType {
			case shared.ContextFileType:
				numFiles++
			case shared.ContextURLType:
				numUrls++
			case shared.ContextDirectoryTreeType:
				numTrees++
			case shared.ContextMapType:
				numMaps++
			}

			errCh <- nil
		}(id, params)
	}

	for i := 0; i < len(*req); i++ {
		err := <-errCh
		if err != nil {
			return nil, fmt.Errorf("error getting context: %v", err)
		}
	}

	if planConfig.AutoLoadContext {
		if totalBasicPlannerTokens > plannerMaxTokens {
			return &shared.UpdateContextResponse{
				TokensAdded:       aggregateTokensDiff,
				TotalTokens:       totalBasicPlannerTokens,
				MaxTokens:         plannerMaxTokens,
				MaxTokensExceeded: true,
			}, nil
		}
	}
	filesToLoad := map[string]string{}
	for _, context := range updatedContexts {
		if context.ContextType == shared.ContextFileType {
			filesToLoad[context.FilePath] = (*req)[context.Id].Body
		}
	}

	if !params.SkipConflictInvalidation {
		err = invalidateConflictedResults(invalidateConflictedResultsParams{
			orgId:         orgId,
			planId:        planId,
			filesToUpdate: filesToLoad,
		})
		if err != nil {
			return nil, fmt.Errorf("error invalidating conflicted results: %v", err)
		}
	}

	errCh = make(chan error, len(*req))

	for id, params := range *req {
		go func(id string, params *shared.UpdateContextParams) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in UpdateContexts: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in UpdateContexts: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			context := contextsById[id]

			if context.ContextType == shared.ContextMapType {
				oldNumTokens := context.NumTokens

				for path, part := range params.MapBodies {
					if len(part) > shared.MaxContextMapSingleInputSize {
						errCh <- fmt.Errorf("map input %s is too large: %d", path, len(part))
						return
					}

					if context.MapParts == nil {
						context.MapParts = make(shared.FileMapBodies)
					}
					if context.MapShas == nil {
						context.MapShas = make(map[string]string)
					}
					if context.MapTokens == nil {
						context.MapTokens = make(map[string]int)
					}
					if context.MapSizes == nil {
						context.MapSizes = make(map[string]int64)
					}

					// prevNumTokens := context.MapTokens[path]

					context.MapParts[path] = part
					context.MapShas[path] = params.InputShas[path]
					context.MapTokens[path] = params.InputTokens[path]
					context.MapSizes[path] = params.InputSizes[path]
				}

				for _, path := range params.RemovedMapPaths {
					delete(context.MapParts, path)
					delete(context.MapShas, path)
					delete(context.MapTokens, path)
					delete(context.MapSizes, path)
				}

				if len(context.MapParts) > shared.MaxContextMapPaths {
					errCh <- fmt.Errorf("map has too many paths: %d", len(context.MapParts))
					return
				}

				totalMapSize := 0
				for _, part := range context.MapParts {
					totalMapSize += len(part)
				}
				if totalMapSize > shared.MaxContextBodySize {
					errCh <- fmt.Errorf("map total size is too large: %d", totalMapSize)
					return
				}

				context.Body = context.MapParts.CombinedMap(context.MapTokens)
				newNumTokens := shared.GetNumTokensEstimate(context.Body)
				tokenDiff := newNumTokens - oldNumTokens

				mu.Lock()
				tokenDiffsById[id] = tokenDiff
				aggregateTokensDiff += tokenDiff
				totalTokens += tokenDiff
				if planConfig.AutoLoadContext {
					totalMapTokens += tokenDiff
				} else {
					totalPlannerTokens += tokenDiff
				}
				mu.Unlock()

				context.NumTokens = newNumTokens
			} else {
				context.Body = params.Body
				hash := sha256.Sum256([]byte(context.Body))
				context.Sha = hex.EncodeToString(hash[:])
			}

			// log.Println("storing context", id)
			// log.Printf("context name: %s, sha: %s\n", context.Name, context.Sha)

			err := StoreContext(context, false)

			if err != nil {
				errCh <- fmt.Errorf("error storing context: %v", err)
				return
			}

			// log.Println("stored context", id)
			// log.Println()

			errCh <- nil
		}(id, params)
	}

	for i := 0; i < len(*req); i++ {
		err := <-errCh
		if err != nil {
			return nil, fmt.Errorf("error storing context: %v", err)
		}
	}

	if planConfig.AutoLoadContext {
		if totalMapTokens > contextLoaderMaxTokens {
			return &shared.UpdateContextResponse{
				TokensAdded:       aggregateTokensDiff,
				TotalTokens:       totalTokens,
				MaxTokens:         contextLoaderMaxTokens,
				MaxTokensExceeded: true,
			}, nil
		}
	}

	updateRes := &shared.ContextUpdateResult{
		UpdatedContexts: updatedContexts,
		TokenDiffsById:  tokenDiffsById,
		TokensDiff:      aggregateTokensDiff,
		TotalTokens:     totalTokens,
		NumFiles:        numFiles,
		NumUrls:         numUrls,
		NumTrees:        numTrees,
		NumMaps:         numMaps,
		MaxTokens:       plannerMaxTokens,
	}

	err = AddPlanContextTokens(planId, branchName, aggregateTokensDiff)
	if err != nil {
		return nil, fmt.Errorf("error adding plan context tokens: %v", err)
	}

	commitMsg := shared.SummaryForUpdateContext(shared.SummaryForUpdateContextParams{
		NumFiles:    numFiles,
		NumTrees:    numTrees,
		NumUrls:     numUrls,
		NumMaps:     numMaps,
		TokensDiff:  aggregateTokensDiff,
		TotalTokens: totalTokens,
	}) + "\n\n" + shared.TableForContextUpdate(updateRes)
	return &shared.LoadContextResponse{
		TokensAdded: aggregateTokensDiff,
		TotalTokens: totalTokens,
		Msg:         commitMsg,
	}, nil
}
