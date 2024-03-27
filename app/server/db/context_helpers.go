package db

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
)

func GetPlanContexts(orgId, planId string, includeBody bool) ([]*Context, error) {
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

	errCh := make(chan error, len(files)/2)
	contextCh := make(chan *Context, len(files)/2)

	// read each context file
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".meta") {
			go func(file os.DirEntry) {
				context, err := GetContext(orgId, planId, strings.TrimSuffix(file.Name(), ".meta"), includeBody)

				if err != nil {
					errCh <- fmt.Errorf("error reading context file: %v", err)
					return
				}

				contextCh <- context
			}(file)
		}
	}

	for i := 0; i < len(files)/2; i++ {
		select {
		case err := <-errCh:
			return nil, fmt.Errorf("error reading context files: %v", err)
		case context := <-contextCh:
			contexts = append(contexts, context)
		}
	}

	// sort contexts by CreatedAt
	sort.Slice(contexts, func(i, j int) bool {
		return contexts[i].CreatedAt.Before(contexts[j].CreatedAt)
	})

	return contexts, nil
}

func GetContext(orgId, planId, contextId string, includeBody bool) (*Context, error) {
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

	return &context, nil
}

func ContextRemove(contexts []*Context) error {
	// remove files
	numFiles := len(contexts) * 2

	errCh := make(chan error, numFiles)
	for _, context := range contexts {
		contextDir := getPlanContextDir(context.OrgId, context.PlanId)
		for _, ext := range []string{".meta", ".body"} {
			go func(context *Context, dir, ext string) {
				errCh <- os.Remove(filepath.Join(dir, context.Id+ext))
			}(context, contextDir, ext)
		}
	}

	for i := 0; i < numFiles; i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error removing context file: %v", err)
		}
	}

	return nil
}

func StoreContext(context *Context) error {
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

	context.Body = originalBody

	return nil
}

type LoadContextsParams struct {
	Req                      *shared.LoadContextRequest
	OrgId                    string
	Plan                     *Plan
	BranchName               string
	UserId                   string
	SkipConflictInvalidation bool
}

func LoadContexts(params LoadContextsParams) (*shared.LoadContextResponse, []*Context, error) {
	req := params.Req
	orgId := params.OrgId
	plan := params.Plan
	planId := plan.Id
	branchName := params.BranchName
	userId := params.UserId

	filesToLoad := map[string]string{}
	for _, context := range *req {
		if context.ContextType == shared.ContextFileType {
			filesToLoad[context.FilePath] = context.Body
		}
	}

	if !params.SkipConflictInvalidation {
		err := invalidateConflictedResults(orgId, planId, filesToLoad)
		if err != nil {
			return nil, nil, fmt.Errorf("error invalidating conflicted results: %v", err)
		}
	}

	tokensAdded := 0

	paramsByTempId := make(map[string]*shared.LoadContextParams)
	numTokensByTempId := make(map[string]int)

	branch, err := GetDbBranch(planId, branchName)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting branch: %v", err)
	}
	totalTokens := branch.ContextTokens

	settings, err := GetPlanSettings(plan, true)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting settings: %v", err)
	}

	maxTokens := settings.GetPlannerEffectiveMaxTokens()

	for _, context := range *req {
		tempId := uuid.New().String()
		numTokens, err := shared.GetNumTokens(context.Body)

		if err != nil {
			return nil, nil, fmt.Errorf("error getting num tokens: %v", err)
		}

		paramsByTempId[tempId] = context
		numTokensByTempId[tempId] = numTokens

		tokensAdded += numTokens
		totalTokens += numTokens
	}

	if totalTokens > maxTokens {
		return &shared.LoadContextResponse{
			TokensAdded:       tokensAdded,
			TotalTokens:       totalTokens,
			MaxTokens:         maxTokens,
			MaxTokensExceeded: true,
		}, nil, nil
	}

	dbContextsCh := make(chan *Context)
	errCh := make(chan error)
	for tempId, params := range paramsByTempId {

		go func(tempId string, params *shared.LoadContextParams) {
			hash := sha256.Sum256([]byte(params.Body))
			sha := hex.EncodeToString(hash[:])

			context := Context{
				// Id generated by db layer
				OrgId:           orgId,
				OwnerId:         userId,
				PlanId:          planId,
				ContextType:     params.ContextType,
				Name:            params.Name,
				Url:             params.Url,
				FilePath:        params.FilePath,
				NumTokens:       numTokensByTempId[tempId],
				Sha:             sha,
				Body:            params.Body,
				ForceSkipIgnore: params.ForceSkipIgnore,
			}

			err := StoreContext(&context)

			if err != nil {
				errCh <- err
				return
			}

			dbContextsCh <- &context

		}(tempId, params)
	}

	var dbContexts []*Context
	var apiContexts []*shared.Context

	for i := 0; i < len(*req); i++ {
		select {
		case err := <-errCh:
			return nil, nil, fmt.Errorf("error storing context: %v", err)
		case dbContext := <-dbContextsCh:
			dbContexts = append(dbContexts, dbContext)
			apiContext := dbContext.ToApi()
			apiContext.Body = ""
			apiContexts = append(apiContexts, apiContext)
		}
	}

	err = AddPlanContextTokens(planId, branchName, tokensAdded)
	if err != nil {
		return nil, nil, fmt.Errorf("error adding plan context tokens: %v", err)
	}

	commitMsg := shared.SummaryForLoadContext(apiContexts, tokensAdded, totalTokens)

	if len(apiContexts) > 1 {
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

	settings, err := GetPlanSettings(plan, true)
	if err != nil {
		return nil, fmt.Errorf("error getting settings: %v", err)
	}

	maxTokens := settings.GetPlannerEffectiveMaxTokens()
	totalTokens := branch.ContextTokens

	tokensDiff := 0
	tokenDiffsById := make(map[string]int)

	var contextsById map[string]*Context
	if params.ContextsById == nil {
		contextsById = make(map[string]*Context)
	} else {
		contextsById = params.ContextsById
	}

	var updatedContexts []*shared.Context

	numFiles := 0
	numUrls := 0
	numTrees := 0

	var mu sync.Mutex
	errCh := make(chan error)

	for id, params := range *req {
		go func(id string, params *shared.UpdateContextParams) {

			var context *Context
			if _, ok := contextsById[id]; ok {
				context = contextsById[id]
			} else {
				var err error
				context, err = GetContext(orgId, planId, id, true)

				if err != nil {
					errCh <- fmt.Errorf("error getting context: %v", err)
					return
				}
			}

			mu.Lock()
			defer mu.Unlock()

			contextsById[id] = context
			updatedContexts = append(updatedContexts, context.ToApi())
			updateNumTokens, err := shared.GetNumTokens(params.Body)

			if err != nil {
				errCh <- fmt.Errorf("error getting num tokens: %v", err)
				return
			}

			tokenDiff := updateNumTokens - context.NumTokens
			tokenDiffsById[id] = tokenDiff
			tokensDiff += tokenDiff
			totalTokens += tokenDiff

			context.NumTokens = updateNumTokens

			switch context.ContextType {
			case shared.ContextFileType:
				numFiles++
			case shared.ContextURLType:
				numUrls++
			case shared.ContextDirectoryTreeType:
				numTrees++
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

	updateRes := &shared.ContextUpdateResult{
		UpdatedContexts: updatedContexts,
		TokenDiffsById:  tokenDiffsById,
		TokensDiff:      tokensDiff,
		TotalTokens:     totalTokens,
		NumFiles:        numFiles,
		NumUrls:         numUrls,
		NumTrees:        numTrees,
		MaxTokens:       maxTokens,
	}

	if totalTokens > maxTokens {
		return &shared.UpdateContextResponse{
			TokensAdded:       tokensDiff,
			TotalTokens:       totalTokens,
			MaxTokens:         maxTokens,
			MaxTokensExceeded: true,
		}, nil
	}

	filesToLoad := map[string]string{}
	for _, context := range updatedContexts {
		if context.ContextType == shared.ContextFileType {
			filesToLoad[context.FilePath] = (*req)[context.Id].Body
		}
	}

	if !params.SkipConflictInvalidation {
		err = invalidateConflictedResults(orgId, planId, filesToLoad)
		if err != nil {
			return nil, fmt.Errorf("error invalidating conflicted results: %v", err)
		}
	}

	errCh = make(chan error)

	for id, params := range *req {
		go func(id string, params *shared.UpdateContextParams) {

			context := contextsById[id]

			hash := sha256.Sum256([]byte(params.Body))
			sha := hex.EncodeToString(hash[:])

			context.Body = params.Body
			context.Sha = sha

			err := StoreContext(context)

			if err != nil {
				errCh <- fmt.Errorf("error storing context: %v", err)
				return
			}

			errCh <- nil
		}(id, params)
	}

	for i := 0; i < len(*req); i++ {
		err := <-errCh
		if err != nil {
			return nil, fmt.Errorf("error storing context: %v", err)
		}
	}

	err = AddPlanContextTokens(planId, branchName, tokensDiff)
	if err != nil {
		return nil, fmt.Errorf("error adding plan context tokens: %v", err)
	}

	commitMsg := shared.SummaryForUpdateContext(updateRes) + "\n\n" + shared.TableForContextUpdate(updateRes)

	return &shared.LoadContextResponse{
		TokensAdded: tokensDiff,
		TotalTokens: totalTokens,
		Msg:         commitMsg,
	}, nil
}

func invalidateConflictedResults(orgId, planId string, filesToLoad map[string]string) error {
	descriptions, err := GetConvoMessageDescriptions(orgId, planId)
	if err != nil {
		return fmt.Errorf("error getting pending build descriptions: %v", err)
	}

	currentPlan, err := GetCurrentPlanState(CurrentPlanStateParams{
		OrgId:                    orgId,
		PlanId:                   planId,
		ConvoMessageDescriptions: descriptions,
	})

	if err != nil {
		return fmt.Errorf("error getting current plan state: %v", err)
	}

	conflictPaths := currentPlan.PlanResult.FileResultsByPath.ConflictedPaths(filesToLoad)

	// log.Println("invalidateConflictedResults - Conflicted paths:", conflictPaths)

	if len(conflictPaths) > 0 {
		errCh := make(chan error)
		numRoutines := 0

		for _, desc := range descriptions {
			if !desc.DidBuild || desc.AppliedAt != nil {
				continue
			}

			for _, path := range desc.Files {
				if _, found := conflictPaths[path]; found {
					if desc.BuildPathsInvalidated == nil {
						desc.BuildPathsInvalidated = make(map[string]bool)
					}
					desc.BuildPathsInvalidated[path] = true

					// log.Printf("Invalidating build for path: %s, desc: %s\n", path, desc.Id)

					go func(desc *ConvoMessageDescription) {
						err := StoreDescription(desc)

						if err != nil {
							errCh <- fmt.Errorf("error storing description: %v", err)
							return
						}

						errCh <- nil
					}(desc)

					numRoutines++
				}
			}
		}

		go func() {
			err := DeletePendingResultsForPaths(orgId, planId, conflictPaths)

			if err != nil {
				errCh <- fmt.Errorf("error deleting pending results: %v", err)
				return
			}

			errCh <- nil
		}()
		numRoutines++

		for i := 0; i < numRoutines; i++ {
			err := <-errCh
			if err != nil {
				return fmt.Errorf("error storing description: %v", err)
			}
		}
	}

	return nil
}
