package db

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plandex-server/syntax/file_map"
	"sort"
	"strings"
	"sync"
	"time"

	shared "plandex-shared"

	"github.com/google/uuid"
)

type Ctx context.Context

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

func ContextRemove(orgId, planId string, contexts []*Context) error {
	return contextRemove(contextRemoveParams{
		orgId:    orgId,
		planId:   planId,
		contexts: contexts,
	})
}

type contextRemoveParams struct {
	orgId        string
	planId       string
	contexts     []*Context
	descriptions []*ConvoMessageDescription
	currentPlan  *shared.CurrentPlanState
}

func contextRemove(params contextRemoveParams) error {
	orgId := params.orgId
	planId := params.planId
	contexts := params.contexts

	// remove files
	numFiles := 0

	filesToUpdate := make(map[string]string)

	errCh := make(chan error, numFiles)
	for _, context := range contexts {
		filesToUpdate[context.FilePath] = ""
		contextDir := getPlanContextDir(orgId, planId)
		for _, ext := range []string{".meta", ".body", ".map-parts"} {
			numFiles++
			go func(context *Context, dir, ext string) {
				errCh <- os.Remove(filepath.Join(dir, context.Id+ext))
			}(context, contextDir, ext)
		}
	}

	for i := 0; i < numFiles; i++ {
		err := <-errCh
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error removing context file: %v", err)
		}
	}

	err := invalidateConflictedResults(invalidateConflictedResultsParams{
		orgId:         orgId,
		planId:        planId,
		filesToUpdate: filesToUpdate,
		descriptions:  params.descriptions,
		currentPlan:   params.currentPlan,
	})
	if err != nil {
		return fmt.Errorf("error invalidating conflicted results: %v", err)
	}

	return nil
}

type ClearContextParams struct {
	OrgId       string
	PlanId      string
	SkipMaps    bool
	SkipPending bool
}

func ClearContext(params ClearContextParams) error {
	orgId := params.OrgId
	planId := params.PlanId
	skipMaps := params.SkipMaps
	skipPending := params.SkipPending

	contexts, err := GetPlanContexts(orgId, planId, false, false)
	if err != nil {
		return fmt.Errorf("error getting plan contexts: %v", err)
	}

	var descriptions []*ConvoMessageDescription
	var currentPlan *shared.CurrentPlanState

	if skipPending {
		var err error
		descriptions, err = GetConvoMessageDescriptions(orgId, planId)
		if err != nil {
			return fmt.Errorf("error getting pending build descriptions: %v", err)
		}

		currentPlan, err = GetCurrentPlanState(CurrentPlanStateParams{
			OrgId:                    orgId,
			PlanId:                   planId,
			ConvoMessageDescriptions: descriptions,
		})

		if err != nil {
			return fmt.Errorf("error getting current plan state: %v", err)
		}
	}

	toRemove := []*Context{}

	for _, context := range contexts {
		shouldSkip := false

		if !(skipMaps && context.ContextType == shared.ContextMapType) {
			shouldSkip = true
		}

		if skipPending && currentPlan.CurrentPlanFiles.Files[context.FilePath] != "" {
			shouldSkip = true
		}

		if !shouldSkip {
			toRemove = append(toRemove, context)
		}
	}

	if len(toRemove) > 0 {
		err := contextRemove(contextRemoveParams{
			orgId:        orgId,
			planId:       planId,
			contexts:     toRemove,
			descriptions: descriptions,
			currentPlan:  currentPlan,
		})
		if err != nil {
			return fmt.Errorf("error removing non-map contexts: %v", err)
		}
	}

	return nil
}

func StoreContext(context *Context, skipMapCache bool) error {
	// log.Println("Storing context", context.Id)
	// log.Println("Num tokens", context.NumTokens)

	contextDir := getPlanContextDir(context.OrgId, context.PlanId)

	err := os.MkdirAll(contextDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating context dir: %v", err)
	}

	ts := time.Now().UTC()
	if context.Id == "" {
		context.Id = uuid.New().String()
		context.CreatedAt = ts
	}
	context.UpdatedAt = ts

	metaFilename := context.Id + ".meta"
	metaPath := filepath.Join(contextDir, metaFilename)

	originalBody := context.Body
	originalBody = strings.ReplaceAll(originalBody, "\\`\\`\\`", "\\\\`\\\\`\\\\`")
	originalBody = strings.ReplaceAll(originalBody, "```", "\\`\\`\\`")

	bodyFilename := context.Id + ".body"
	bodyPath := filepath.Join(contextDir, bodyFilename)
	body := []byte(originalBody)
	context.Body = ""

	originalMapParts := context.MapParts
	var mapPath string
	var mapBytes []byte
	if len(context.MapParts) > 0 {
		mapFilename := context.Id + ".map-parts"
		mapPath = filepath.Join(contextDir, mapFilename)
		mapBytes, err = json.MarshalIndent(context.MapParts, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal map parts: %v", err)
		}
		context.MapParts = nil
	}

	// Convert the ModelContextPart to JSON
	data, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context context: %v", err)
	}

	// Write the body to the file
	if err = os.WriteFile(bodyPath, body, 0644); err != nil {
		return fmt.Errorf("failed to write context body to file %s: %v", bodyPath, err)
	}

	// Write the meta data to the file
	if err = os.WriteFile(metaPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write context meta to file %s: %v", metaPath, err)
	}

	if mapPath != "" {
		if err = os.WriteFile(mapPath, mapBytes, 0644); err != nil {
			return fmt.Errorf("failed to write context map to file %s: %v", mapPath, err)
		}
	}

	context.Body = originalBody
	context.MapParts = originalMapParts

	if mapPath != "" && !skipMapCache {
		log.Println("StoreContext - context.MapParts length", len(context.MapParts))

		mapCacheDir := getProjectMapCacheDir(context.OrgId, context.ProjectId)

		// ensure map cache dir exists
		err = os.MkdirAll(mapCacheDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error creating map cache dir: %v", err)
		}

		filePathHash := md5.Sum([]byte(context.FilePath))
		filePathHashStr := hex.EncodeToString(filePathHash[:])

		mapCachePath := filepath.Join(mapCacheDir, filePathHashStr+".json")

		log.Println("StoreContext - mapCachePath", mapCachePath)

		cachedContext := Context{
			ContextType: shared.ContextMapType,
			FilePath:    context.FilePath,
			Name:        context.Name,
			Body:        context.Body,
			NumTokens:   context.NumTokens,
			MapParts:    context.MapParts,
			MapShas:     context.MapShas,
			MapTokens:   context.MapTokens,
			UpdatedAt:   context.UpdatedAt,
		}

		cachedContextBytes, err := json.MarshalIndent(cachedContext, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal cached context: %v", err)
		}

		err = os.WriteFile(mapCachePath, cachedContextBytes, 0644)
		if err != nil {
			return fmt.Errorf("failed to write context map to file %s: %v", mapCachePath, err)
		}
	}

	return nil
}

func GetCachedMap(orgId, projectId, filePath string) (*Context, error) {
	mapCacheDir := getProjectMapCacheDir(orgId, projectId)

	filePathHash := md5.Sum([]byte(filePath))
	filePathHashStr := hex.EncodeToString(filePathHash[:])

	mapCachePath := filepath.Join(mapCacheDir, filePathHashStr+".json")

	log.Println("GetCachedMap - mapCachePath", mapCachePath)

	mapCacheBytes, err := os.ReadFile(mapCachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("error reading cached map: %v", err)
	}

	var context Context
	err = json.Unmarshal(mapCacheBytes, &context)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling cached map: %v", err)
	}

	return &context, nil
}

type CachedMap struct {
	MapParts  shared.FileMapBodies
	MapShas   map[string]string
	MapTokens map[string]int
}

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

	// showElapsed("Filtered reqs")

	for _, context := range *req {
		tempId := uuid.New().String()

		var numTokens int
		var err error

		var isMap bool

		if context.ContextType == shared.ContextMapType && (len(context.MapInputs) > 0 || params.CachedMapsByPath != nil) {
			isMap = true
			var mappedFiles shared.FileMapBodies
			if params.CachedMapsByPath != nil && params.CachedMapsByPath[context.FilePath] != nil {
				log.Println("Using cached map for", context.FilePath)
				mappedFiles = params.CachedMapsByPath[context.FilePath].MapParts
			} else {
				log.Println("Processing map files for", context.FilePath)
				// showElapsed(context.FilePath + " - processing map files")
				// Process file maps
				mappedFiles, err = file_map.ProcessMapFiles(ctx, context.MapInputs)
				if err != nil {
					return nil, nil, fmt.Errorf("error processing map files: %v", err)
				}
				// showElapsed(context.FilePath + " - processed map files")
			}

			var mapShas map[string]string
			var mapTokens map[string]int

			if params.CachedMapsByPath != nil && params.CachedMapsByPath[context.FilePath] != nil {
				mapShas = params.CachedMapsByPath[context.FilePath].MapShas
				mapTokens = params.CachedMapsByPath[context.FilePath].MapTokens
			} else {
				mapShas = make(map[string]string, len(context.MapInputs))
				mapTokens = make(map[string]int, len(context.MapInputs))

				for path, input := range context.MapInputs {
					hash := sha256.Sum256([]byte(input))
					mapShas[path] = hex.EncodeToString(hash[:])
					mapBody := mappedFiles[path]
					mapTokens[path] = shared.GetNumTokensEstimate(mapBody)
				}
			}

			combinedBody := mappedFiles.CombinedMap()
			numTokens = shared.GetNumTokensEstimate(combinedBody)

			autoLoaded = autoLoaded || context.AutoLoaded

			log.Println("LoadContexts - map - autoLoaded", autoLoaded)

			newContext := Context{
				// Id generated by db layer
				OrgId:       orgId,
				OwnerId:     userId,
				PlanId:      planId,
				ProjectId:   plan.ProjectId,
				ContextType: shared.ContextMapType,
				Name:        context.Name,
				Url:         context.Url,
				FilePath:    context.FilePath,
				NumTokens:   numTokens,
				Body:        combinedBody,
				MapParts:    mappedFiles,
				MapShas:     mapShas,
				MapTokens:   mapTokens,
				AutoLoaded:  autoLoaded || context.AutoLoaded,
			}

			mapContextsByFilePath[context.FilePath] = newContext

		} else if context.ContextType == shared.ContextImageType {
			numTokens, err = shared.GetImageTokens(context.Body, context.ImageDetail)
			if err != nil {
				return nil, nil, fmt.Errorf("error getting image num tokens: %v", err)
			}
		} else {
			numTokens = shared.GetNumTokensEstimate(context.Body)
		}

		paramsByTempId[tempId] = context
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
		commitMsg += "\n\n" + shared.TableForLoadContext(apiContexts)
	}

	return &shared.LoadContextResponse{
		TokensAdded: tokensAdded,
		TotalTokens: totalTokens,
		Msg:         commitMsg,
	}, dbContexts, nil
}

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
		totalBodySize += int64(len(context.Body))
	}

	for id, params := range *req {
		if context, ok := contextsById[id]; ok {
			totalBodySize += int64(len(params.Body)) - int64(len(context.Body))
		} else {
			totalContextCount++
			totalBodySize += int64(len(params.Body))
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
			context := contextsById[id]

			if context.ContextType == shared.ContextMapType {
				oldNumTokens := context.NumTokens
				for path, part := range params.MapBodies {
					if context.MapParts == nil {
						context.MapParts = make(shared.FileMapBodies)
					}
					if context.MapShas == nil {
						context.MapShas = make(map[string]string)
					}
					if context.MapTokens == nil {
						context.MapTokens = make(map[string]int)
					}

					// prevNumTokens := context.MapTokens[path]

					context.MapParts[path] = part
					context.MapShas[path] = params.InputShas[path]

					numTokens := params.MapBodies.TokenEstimateForPath(path)
					context.MapTokens[path] = numTokens
				}

				for _, path := range params.RemovedMapPaths {
					delete(context.MapParts, path)
					delete(context.MapShas, path)
					delete(context.MapTokens, path)
				}

				context.Body = context.MapParts.CombinedMap()
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

	commitMsg := shared.SummaryForUpdateContext(updateRes) + "\n\n" + shared.TableForContextUpdate(updateRes)

	return &shared.LoadContextResponse{
		TokensAdded: aggregateTokensDiff,
		TotalTokens: totalTokens,
		Msg:         commitMsg,
	}, nil
}

type invalidateConflictedResultsParams struct {
	orgId         string
	planId        string
	filesToUpdate map[string]string
	descriptions  []*ConvoMessageDescription
	currentPlan   *shared.CurrentPlanState
}

func invalidateConflictedResults(params invalidateConflictedResultsParams) error {
	orgId := params.orgId
	planId := params.planId
	filesToUpdate := params.filesToUpdate

	var descriptions []*ConvoMessageDescription

	if params.descriptions == nil {
		var err error
		descriptions, err = GetConvoMessageDescriptions(orgId, planId)
		if err != nil {
			return fmt.Errorf("error getting pending build descriptions: %v", err)
		}
	} else {
		descriptions = params.descriptions
	}

	var currentPlan *shared.CurrentPlanState

	if params.currentPlan == nil {
		var err error
		currentPlan, err = GetCurrentPlanState(CurrentPlanStateParams{
			OrgId:                    orgId,
			PlanId:                   planId,
			ConvoMessageDescriptions: descriptions,
		})
		if err != nil {
			return fmt.Errorf("error getting current plan state: %v", err)
		}
	} else {
		currentPlan = params.currentPlan
	}

	conflictPaths := currentPlan.PlanResult.FileResultsByPath.ConflictedPaths(filesToUpdate)

	// log.Println("invalidateConflictedResults - Conflicted paths:", conflictPaths)

	if len(conflictPaths) > 0 {
		toUpdateDescs := []*ConvoMessageDescription{}

		for _, desc := range descriptions {
			if !desc.DidBuild || desc.AppliedAt != nil {
				continue
			}

			for _, op := range desc.Operations {
				if _, found := conflictPaths[op.Path]; found {
					if desc.BuildPathsInvalidated == nil {
						desc.BuildPathsInvalidated = make(map[string]bool)
					}
					desc.BuildPathsInvalidated[op.Path] = true
					toUpdateDescs = append(toUpdateDescs, desc)
				}
			}
		}

		numRoutines := len(toUpdateDescs) + 1
		errCh := make(chan error, numRoutines)

		for _, desc := range toUpdateDescs {
			go func(desc *ConvoMessageDescription) {
				err := StoreDescription(desc)

				if err != nil {
					errCh <- fmt.Errorf("error storing description: %v", err)
					return
				}

				errCh <- nil
			}(desc)
		}

		go func() {
			err := DeletePendingResultsForPaths(orgId, planId, conflictPaths)

			if err != nil {
				errCh <- fmt.Errorf("error deleting pending results: %v", err)
				return
			}

			errCh <- nil
		}()

		for i := 0; i < numRoutines; i++ {
			err := <-errCh
			if err != nil {
				return fmt.Errorf("error storing description: %v", err)
			}
		}
	}

	return nil
}
