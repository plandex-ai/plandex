package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	shared "plandex-shared"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// Ctx is a context.Context - to avoid confusion with Plandex contexts
type Ctx context.Context

type LoadContextsParams struct {
	Req                      *shared.LoadContextRequest
	OrgId                    string
	Plan                     *Plan
	BranchName               string
	UserId                   string
	SkipConflictInvalidation bool
	CachedMapsByPath         map[string]*CachedMap
	AutoLoaded               bool
}

func LoadContexts(ctx Ctx, params LoadContextsParams) (*shared.LoadContextResponse, []*Context, error) {
	// startTime := time.Now()
	// showElapsed := func(msg string) {
	// 	elapsed := time.Since(startTime)
	// 	log.Println("LoadContexts", msg, "elapsed: %s\n", elapsed)
	// }

	// log.Println("LoadContexts - params", spew.Sdump(params))

	req := params.Req
	orgId := params.OrgId
	plan := params.Plan
	planId := plan.Id
	branchName := params.BranchName
	userId := params.UserId
	autoLoaded := params.AutoLoaded

	filesToLoad := map[string]string{}
	for _, context := range *req {
		if context.ContextType == shared.ContextFileType {
			filesToLoad[context.FilePath] = context.Body
		}
	}

	if !params.SkipConflictInvalidation {
		err := invalidateConflictedResults(invalidateConflictedResultsParams{
			orgId:         orgId,
			planId:        planId,
			filesToUpdate: filesToLoad,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("error invalidating conflicted results: %v", err)
		}
	}

	tokensAdded := 0
	basicTokensAdded := 0

	paramsByTempId := make(map[string]*shared.LoadContextParams)
	numTokensByTempId := make(map[string]int)

	branch, err := GetDbBranch(planId, branchName)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting branch: %v", err)
	}
	totalTokens := branch.ContextTokens
	totalPlannerTokens := totalTokens
	totalBasicPlannerTokens := 0
	totalMapTokens := 0

	settings, err := GetPlanSettings(plan, true)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting settings: %v", err)
	}

	planConfig, err := GetPlanConfig(planId)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting plan config: %v", err)
	}

	plannerMaxTokens := settings.GetPlannerEffectiveMaxTokens()
	contextLoaderMaxTokens := settings.GetArchitectEffectiveMaxTokens()

	mapContextsByFilePath := make(map[string]Context)

	existingContexts, err := GetPlanContexts(orgId, planId, false, false)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting existing contexts: %v", err)
	}

	// check overall context limits - these should be getting enforced by the client, so just error out if exceeded
	numExistingContexts := len(existingContexts)
	if numExistingContexts+len(*req) > shared.MaxContextCount {
		return nil, nil, fmt.Errorf("too many contexts: %d", numExistingContexts+len(*req))
	}

	var totalContextSize int64
	for _, context := range existingContexts {
		totalContextSize += context.BodySize
	}
	for _, context := range *req {
		size := int64(len(context.Body))
		totalContextSize += size
		if size > shared.MaxContextBodySize {
			return nil, nil, fmt.Errorf("context body is too large: %d", size)
		}
	}

	if totalContextSize > shared.MaxTotalContextSize {
		return nil, nil, fmt.Errorf("total context size is too large: %d", totalContextSize)
	}

	existingContextsByName := make(map[string]bool)
	for _, context := range existingContexts {
		composite := strings.Join([]string{context.Name, string(context.ContextType)}, "|")
		existingContextsByName[composite] = true

		if planConfig.AutoLoadContext && context.ContextType == shared.ContextMapType {
			totalMapTokens += context.NumTokens
			totalPlannerTokens -= context.NumTokens
		}

		if !context.AutoLoaded && context.ContextType != shared.ContextMapType {
			totalBasicPlannerTokens += context.NumTokens
		}
	}

	var filteredReq []*shared.LoadContextParams
	for _, context := range *req {
		composite := strings.Join([]string{context.Name, string(context.ContextType)}, "|")
		if !existingContextsByName[composite] {
			filteredReq = append(filteredReq, context)
		}
	}

	*req = filteredReq

	for _, contextParams := range *req {
		tempId := uuid.New().String()

		var numTokens int
		var err error

		var isMap bool

		if contextParams.ContextType == shared.ContextMapType && (len(contextParams.MapBodies) > 0 || params.CachedMapsByPath != nil) {
			isMap = true
			var mappedFiles shared.FileMapBodies
			if params.CachedMapsByPath != nil && params.CachedMapsByPath[contextParams.FilePath] != nil {
				log.Println("Using cached map for", contextParams.FilePath)
				mappedFiles = params.CachedMapsByPath[contextParams.FilePath].MapParts
			} else {
				log.Println("Using map bodies for", contextParams.FilePath)
				mappedFiles = contextParams.MapBodies

				// check size and num path limits - these should be getting enforced by the client, so just error out if exceeded
				if len(mappedFiles) > shared.MaxContextMapPaths {
					return nil, nil, fmt.Errorf("map has too many paths: %d", len(mappedFiles))
				}

				totalMapSize := 0
				for _, body := range mappedFiles {
					numBytes := len(body)
					totalMapSize += numBytes
					if numBytes > shared.MaxContextMapSingleInputSize {
						return nil, nil, fmt.Errorf("map input %s is too large: %d", contextParams.FilePath, numBytes)
					}
				}
				if totalMapSize > shared.MaxContextBodySize {
					return nil, nil, fmt.Errorf("map is too large: %d", totalMapSize)
				}

			}

			var mapShas map[string]string
			var mapTokens map[string]int
			var mapSizes map[string]int64

			if params.CachedMapsByPath != nil && params.CachedMapsByPath[contextParams.FilePath] != nil {
				mapShas = params.CachedMapsByPath[contextParams.FilePath].MapShas
				mapTokens = params.CachedMapsByPath[contextParams.FilePath].MapTokens
				mapSizes = params.CachedMapsByPath[contextParams.FilePath].MapSizes
			} else {
				mapShas = contextParams.InputShas
				mapTokens = contextParams.InputTokens
				mapSizes = contextParams.InputSizes
			}

			combinedBody := mappedFiles.CombinedMap(mapTokens)
			numTokens = shared.GetNumTokensEstimate(combinedBody)

			autoLoaded = autoLoaded || contextParams.AutoLoaded

			log.Println("LoadContexts - map - autoLoaded", autoLoaded)

			newContext := Context{
				// Id generated by db layer
				OrgId:       orgId,
				OwnerId:     userId,
				PlanId:      planId,
				ProjectId:   plan.ProjectId,
				ContextType: shared.ContextMapType,
				Name:        contextParams.Name,
				Url:         contextParams.Url,
				FilePath:    contextParams.FilePath,
				NumTokens:   numTokens,
				Body:        combinedBody,
				MapParts:    mappedFiles,
				MapShas:     mapShas,
				MapTokens:   mapTokens,
				MapSizes:    mapSizes,
				AutoLoaded:  autoLoaded || contextParams.AutoLoaded,
			}

			mapContextsByFilePath[contextParams.FilePath] = newContext

		} else if contextParams.ContextType == shared.ContextImageType {
			numTokens, err = shared.GetImageTokens(contextParams.Body, contextParams.ImageDetail)
			if err != nil {
				return nil, nil, fmt.Errorf("error getting image num tokens: %v", err)
			}
		} else {
			numTokens = shared.GetNumTokensEstimate(contextParams.Body)
		}

		paramsByTempId[tempId] = contextParams
		numTokensByTempId[tempId] = numTokens
		totalTokens += numTokens

		// maps don't count toward the token limit if auto-loading
		if planConfig.AutoLoadContext && isMap {
			tokensAdded += numTokens
			totalMapTokens += numTokens
		} else if autoLoaded {
			tokensAdded += numTokens
			totalPlannerTokens += numTokens
		} else {
			tokensAdded += numTokens
			totalPlannerTokens += numTokens
			totalBasicPlannerTokens += numTokens
			basicTokensAdded += numTokens
		}
	}

	// showElapsed("Loaded reqs")
	if planConfig.AutoLoadContext {
		if totalMapTokens > contextLoaderMaxTokens {
			return &shared.LoadContextResponse{
				TokensAdded:       tokensAdded,
				TotalTokens:       totalMapTokens,
				MaxTokens:         contextLoaderMaxTokens,
				MaxTokensExceeded: true,
			}, nil, nil
		}

		if totalBasicPlannerTokens > plannerMaxTokens {
			return &shared.LoadContextResponse{
				TokensAdded:       basicTokensAdded,
				TotalTokens:       totalBasicPlannerTokens,
				MaxTokens:         plannerMaxTokens,
				MaxTokensExceeded: true,
			}, nil, nil
		}
	} else {
		if totalTokens > plannerMaxTokens {
			return &shared.LoadContextResponse{
				TokensAdded:       tokensAdded,
				TotalTokens:       totalTokens,
				MaxTokens:         plannerMaxTokens,
				MaxTokensExceeded: true,
			}, nil, nil
		}
	}

	var dbContexts []*Context
	var apiContexts []*shared.Context
	var mu sync.Mutex

	errCh := make(chan error, len(paramsByTempId))
	for tempId, loadParams := range paramsByTempId {

		go func(tempId string, loadParams *shared.LoadContextParams) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in LoadContexts: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in LoadContexts: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			hash := sha256.Sum256([]byte(loadParams.Body))
			sha := hex.EncodeToString(hash[:])

			var context Context
			if mapContext, ok := mapContextsByFilePath[loadParams.FilePath]; ok {
				context = mapContext
			} else {
				// log.Println("tempId", tempId, "params.FilePath", params.FilePath, "sha", sha)
				// log.Println("params.Body", params.Body)

				context = Context{
					// Id generated by db layer
					OrgId:           orgId,
					OwnerId:         userId,
					PlanId:          planId,
					ProjectId:       plan.ProjectId,
					ContextType:     loadParams.ContextType,
					Name:            loadParams.Name,
					Url:             loadParams.Url,
					FilePath:        loadParams.FilePath,
					NumTokens:       numTokensByTempId[tempId],
					Sha:             sha,
					Body:            loadParams.Body,
					ForceSkipIgnore: loadParams.ForceSkipIgnore,
					ImageDetail:     loadParams.ImageDetail,
					AutoLoaded:      autoLoaded || loadParams.AutoLoaded,
				}
			}

			err := StoreContext(&context, params.CachedMapsByPath != nil)

			if err != nil {
				errCh <- fmt.Errorf("error storing context: %v", err)
				return
			}

			mu.Lock()
			dbContexts = append(dbContexts, &context)
			apiContext := context.ToApi()
			apiContext.Body = ""
			apiContexts = append(apiContexts, apiContext)
			mu.Unlock()

			errCh <- nil
		}(tempId, loadParams)
	}

	for i := 0; i < len(paramsByTempId); i++ {
		err := <-errCh
		if err != nil {
			return nil, nil, fmt.Errorf("error storing context: %v", err)
		}
	}

	err = AddPlanContextTokens(planId, branchName, tokensAdded)
	if err != nil {
		return nil, nil, fmt.Errorf("error adding plan context tokens: %v", err)
	}

	commitMsg := shared.SummaryForLoadContext(apiContexts, tokensAdded, totalTokens)

	if len(apiContexts) > 0 {
		commitMsg += "\n\n" + shared.TableForLoadContext(apiContexts, false)
	}

	return &shared.LoadContextResponse{
		TokensAdded: tokensAdded,
		TotalTokens: totalTokens,
		Msg:         commitMsg,
	}, dbContexts, nil
}
