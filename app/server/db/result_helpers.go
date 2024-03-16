package db

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
)

func StorePlanResult(result *PlanFileResult) error {
	now := time.Now()
	if result.Id == "" {
		result.Id = uuid.New().String()
		result.CreatedAt = now
	}
	result.UpdatedAt = now

	bytes, err := json.MarshalIndent(result, "", "  ")

	if err != nil {
		return fmt.Errorf("error marshalling result: %v", err)
	}

	resultsDir := getPlanResultsDir(result.OrgId, result.PlanId)

	err = os.MkdirAll(resultsDir, 0755)

	if err != nil {
		return fmt.Errorf("error creating results dir: %v", err)
	}

	err = os.WriteFile(filepath.Join(resultsDir, result.Id+".json"), bytes, 0644)

	if err != nil {
		return fmt.Errorf("error writing result file: %v", err)
	}

	return nil

}

type CurrentPlanStateParams struct {
	OrgId                    string
	PlanId                   string
	PlanFileResults          []*PlanFileResult
	ConvoMessageDescriptions []*ConvoMessageDescription
	Contexts                 []*Context
}

func GetCurrentPlanState(params CurrentPlanStateParams) (*shared.CurrentPlanState, error) {
	orgId := params.OrgId
	planId := params.PlanId

	var dbPlanFileResults []*PlanFileResult
	var convoMessageDescriptions []*shared.ConvoMessageDescription
	contextsByPath := map[string]*Context{}

	errCh := make(chan error)

	go func() {
		if params.PlanFileResults == nil {
			res, err := GetPlanFileResults(orgId, planId)
			dbPlanFileResults = res

			if err != nil {
				errCh <- fmt.Errorf("error getting plan file results: %v", err)
				return
			}
		} else {
			dbPlanFileResults = params.PlanFileResults
		}

		errCh <- nil
	}()

	go func() {
		if params.ConvoMessageDescriptions == nil {
			res, err := GetConvoMessageDescriptions(orgId, planId)
			if err != nil {
				errCh <- fmt.Errorf("error getting latest plan build description: %v", err)
				return
			}

			for _, desc := range res {
				convoMessageDescriptions = append(convoMessageDescriptions, desc.ToApi())
			}
		} else {
			for _, desc := range params.ConvoMessageDescriptions {
				convoMessageDescriptions = append(convoMessageDescriptions, desc.ToApi())
			}
		}

		errCh <- nil
	}()

	go func() {
		var contexts []*Context
		if params.Contexts == nil {
			res, err := GetPlanContexts(orgId, planId, true)
			if err != nil {
				errCh <- fmt.Errorf("error getting contexts: %v", err)
				return
			}
			contexts = res

			log.Println("Got contexts:", len(contexts))
		} else {
			contexts = params.Contexts
		}

		for _, context := range contexts {
			if context.FilePath != "" {
				contextsByPath[context.FilePath] = context
			}
		}

		errCh <- nil
	}()

	for i := 0; i < 3; i++ {
		err := <-errCh
		if err != nil {
			return nil, err
		}
	}

	var apiPlanFileResults []*shared.PlanFileResult
	pendingResultPaths := map[string]bool{}

	for _, dbPlanFileResult := range dbPlanFileResults {
		apiResult := dbPlanFileResult.ToApi()
		apiPlanFileResults = append(apiPlanFileResults, apiResult)

		if apiResult.IsPending() {
			pendingResultPaths[apiResult.Path] = true
		}
	}
	planResult := GetPlanResult(apiPlanFileResults)

	pendingContextsByPath := map[string]*shared.Context{}
	for path, context := range contextsByPath {
		pendingContextsByPath[path] = context.ToApi()
	}

	log.Println("Pending contexts by path:", len(pendingContextsByPath))

	planState := &shared.CurrentPlanState{
		PlanResult:               planResult,
		ConvoMessageDescriptions: convoMessageDescriptions,
		ContextsByPath:           pendingContextsByPath,
	}

	currentPlanFiles, err := planState.GetFiles()

	if err != nil {
		return nil, fmt.Errorf("error getting current plan files: %v", err)
	}

	planState.CurrentPlanFiles = currentPlanFiles

	return planState, nil
}

func GetConvoMessageDescriptions(orgId, planId string) ([]*ConvoMessageDescription, error) {
	var descriptions []*ConvoMessageDescription
	descriptionsDir := getPlanDescriptionsDir(orgId, planId)
	files, err := os.ReadDir(descriptionsDir)

	if err != nil {

		if os.IsNotExist(err) {
			return descriptions, nil
		}

		return nil, fmt.Errorf("error reading descriptions dir: %v", err)
	}

	errCh := make(chan error, len(files))
	descCh := make(chan *ConvoMessageDescription, len(files))

	for _, file := range files {
		go func(file os.DirEntry) {
			path := filepath.Join(descriptionsDir, file.Name())

			bytes, err := os.ReadFile(path)

			if err != nil {
				errCh <- fmt.Errorf("error reading description file %s: %v", file.Name(), err)
				return
			}

			var description ConvoMessageDescription
			err = json.Unmarshal(bytes, &description)

			if err != nil {
				log.Println("Error unmarshalling description file:", path)
				log.Println("bytes:")
				log.Println(string(bytes))

				errCh <- fmt.Errorf("error unmarshalling description file %s: %v", path, err)
				return
			}

			descCh <- &description
		}(file)
	}

	for i := 0; i < len(files); i++ {
		select {
		case err := <-errCh:
			return nil, fmt.Errorf("error reading description files: %v", err)
		case description := <-descCh:
			if description.MadePlan && description.AppliedAt == nil {
				descriptions = append(descriptions, description)
			}
		}
	}

	sort.Slice(descriptions, func(i, j int) bool {
		return descriptions[i].CreatedAt.Before(descriptions[j].CreatedAt)
	})

	return descriptions, nil
}

func GetPlanFileResults(orgId, planId string) ([]*PlanFileResult, error) {
	var results []*PlanFileResult

	resultsDir := getPlanResultsDir(orgId, planId)

	files, err := os.ReadDir(resultsDir)

	if err != nil {
		if os.IsNotExist(err) {
			return results, nil
		}

		return nil, fmt.Errorf("error reading results dir: %v", err)
	}

	errCh := make(chan error, len(files))
	resultCh := make(chan *PlanFileResult, len(files))

	for _, file := range files {
		go func(file os.DirEntry) {

			bytes, err := os.ReadFile(filepath.Join(resultsDir, file.Name()))

			if err != nil {
				errCh <- fmt.Errorf("error reading result file: %v", err)
				return
			}

			var result PlanFileResult
			err = json.Unmarshal(bytes, &result)

			if err != nil {
				errCh <- fmt.Errorf("error unmarshalling result file: %v", err)
				return
			}

			resultCh <- &result
		}(file)
	}

	for i := 0; i < len(files); i++ {
		select {
		case err := <-errCh:
			return nil, fmt.Errorf("error reading result files: %v", err)
		case result := <-resultCh:
			results = append(results, result)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.Before(results[j].CreatedAt)
	})

	return results, nil
}

func GetPlanResult(planFileResults []*shared.PlanFileResult) *shared.PlanResult {
	resByPath := make(shared.PlanFileResultsByPath)
	replacementsByPath := make(map[string][]*shared.Replacement)
	var paths []string

	for _, planFileRes := range planFileResults {
		if planFileRes.IsPending() {
			_, hasPath := resByPath[planFileRes.Path]

			resByPath[planFileRes.Path] = append(resByPath[planFileRes.Path], planFileRes)

			if !hasPath {
				paths = append(paths, planFileRes.Path)
			}
		}
	}

	for _, results := range resByPath {
		for _, planRes := range results {
			replacementsByPath[planRes.Path] = append(replacementsByPath[planRes.Path], planRes.Replacements...)
		}
	}

	// sort paths ascending
	sort.Slice(paths, func(i, j int) bool {
		return paths[i] < paths[j]
	})

	return &shared.PlanResult{
		FileResultsByPath:  resByPath,
		SortedPaths:        paths,
		ReplacementsByPath: replacementsByPath,
		Results:            planFileResults,
	}
}

func ApplyPlan(orgId, userId, branchName string, plan *Plan) error {
	planId := plan.Id

	resultsDir := getPlanResultsDir(orgId, planId)

	errCh := make(chan error)

	var results []*PlanFileResult
	var convoMessageDescriptions []*ConvoMessageDescription
	contextsById := make(map[string]*Context)
	contextsByPath := make(map[string]*Context)

	go func() {
		res, err := GetPlanFileResults(orgId, planId)
		if err != nil {
			errCh <- fmt.Errorf("error getting plan file results: %v", err)
			return
		}
		results = res
		errCh <- nil
	}()

	go func() {
		res, err := GetConvoMessageDescriptions(orgId, planId)
		if err != nil {
			errCh <- fmt.Errorf("error getting latest plan build description: %v", err)
			return
		}
		convoMessageDescriptions = res
		errCh <- nil
	}()

	go func() {
		res, err := GetPlanContexts(orgId, planId, false)
		if err != nil {
			errCh <- fmt.Errorf("error getting contexts: %v", err)
			return
		}

		for _, context := range res {
			contextsById[context.Id] = context
			if context.FilePath != "" {
				contextsByPath[context.FilePath] = context
			}
		}

		errCh <- nil
	}()

	for i := 0; i < 3; i++ {
		err := <-errCh
		if err != nil {
			return err
		}
	}

	var pendingDbResults []*PlanFileResult

	for _, result := range results {
		apiResult := result.ToApi()
		if apiResult.IsPending() {
			pendingDbResults = append(pendingDbResults, result)
		}
	}

	pendingNewFilesSet := make(map[string]bool)
	pendingUpdatedFilesSet := make(map[string]bool)
	for _, result := range pendingDbResults {
		if len(result.Replacements) == 0 && result.Content != "" {
			pendingNewFilesSet[result.Path] = true
		} else if !pendingNewFilesSet[result.Path] {
			pendingUpdatedFilesSet[result.Path] = true
		}
	}

	var loadContextRes *shared.LoadContextResponse
	var updateContextRes *shared.UpdateContextResponse

	var currentPlanState *shared.CurrentPlanState
	if len(pendingNewFilesSet) > 0 || len(pendingUpdatedFilesSet) > 0 {
		res, err := GetCurrentPlanState(CurrentPlanStateParams{
			OrgId:                    orgId,
			PlanId:                   plan.Id,
			PlanFileResults:          results,
			ConvoMessageDescriptions: convoMessageDescriptions,
		})

		if err != nil {
			return fmt.Errorf("error getting current plan state: %v", err)
		}

		currentPlanState = res
	}

	errCh = make(chan error)
	now := time.Now()

	for _, result := range pendingDbResults {
		go func(result *PlanFileResult) {
			result.AppliedAt = &now

			bytes, err := json.MarshalIndent(result, "", "  ")

			if err != nil {
				errCh <- fmt.Errorf("error marshalling result: %v", err)
				return
			}

			err = os.WriteFile(filepath.Join(resultsDir, result.Id+".json"), bytes, 0644)

			if err != nil {
				errCh <- fmt.Errorf("error writing result file: %v", err)
				return
			}

			errCh <- nil

		}(result)
	}

	for _, description := range convoMessageDescriptions {
		go func(description *ConvoMessageDescription) {
			description.AppliedAt = &now

			err := StoreDescription(description)

			if err != nil {
				errCh <- fmt.Errorf("error storing convo message description: %v", err)
				return
			}

			errCh <- nil
		}(description)
	}

	if len(pendingNewFilesSet) > 0 {
		go func() {
			loadReq := shared.LoadContextRequest{}
			for path := range pendingNewFilesSet {
				loadReq = append(loadReq, &shared.LoadContextParams{
					ContextType: shared.ContextFileType,
					Name:        path,
					FilePath:    path,
					Body:        currentPlanState.CurrentPlanFiles.Files[path],
				})
			}

			res, _, err := LoadContexts(
				LoadContextsParams{
					OrgId:                    orgId,
					UserId:                   userId,
					Plan:                     plan,
					BranchName:               branchName,
					Req:                      &loadReq,
					SkipConflictInvalidation: true, // no need to invalidate conflicts when applying plan--and fixes race condition since invalidation check loads description
				},
			)

			if err != nil {
				errCh <- fmt.Errorf("error loading context: %v", err)
				return
			}

			loadContextRes = res
			errCh <- nil
		}()
	}

	if len(pendingUpdatedFilesSet) > 0 {
		go func() {
			updateReq := shared.UpdateContextRequest{}
			for path := range pendingUpdatedFilesSet {
				context := contextsByPath[path]
				updateReq[context.Id] = &shared.UpdateContextParams{
					Body: currentPlanState.CurrentPlanFiles.Files[path],
				}
			}

			res, err := UpdateContexts(
				UpdateContextsParams{
					OrgId:                    orgId,
					Plan:                     plan,
					BranchName:               branchName,
					Req:                      &updateReq,
					SkipConflictInvalidation: true, // no need to invalidate conflicts when applying plan--and fixes race condition since invalidation check loads description
				},
			)

			if err != nil {
				errCh <- fmt.Errorf("error updating context: %v", err)
				return
			}

			updateContextRes = res
			errCh <- nil

		}()

	}

	numRoutines := len(pendingDbResults) +
		len(convoMessageDescriptions)
	if len(pendingNewFilesSet) > 0 {
		numRoutines++
	}
	if len(pendingUpdatedFilesSet) > 0 {
		numRoutines++
	}

	for i := 0; i < numRoutines; i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error applying plan: %v", err)
		}
	}

	msg := "✅ Marked pending results as applied"

	if loadContextRes != nil && !loadContextRes.MaxTokensExceeded {
		msg += "\n\n" + loadContextRes.Msg
	}

	if updateContextRes != nil && !updateContextRes.MaxTokensExceeded {
		msg += "\n\n" + updateContextRes.Msg
	}

	err := GitAddAndCommit(orgId, plan.Id, branchName, msg)

	if err != nil {
		return fmt.Errorf("error committing plan: %v", err)
	}

	return nil
}

func RejectAllResults(orgId, planId string) error {
	resultsDir := getPlanResultsDir(orgId, planId)

	files, err := os.ReadDir(resultsDir)

	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("error reading results dir: %v", err)
	}

	errCh := make(chan error, len(files))
	now := time.Now()

	for _, file := range files {
		resultId := strings.TrimSuffix(file.Name(), ".json")

		go func(resultId string) {
			err := RejectPlanFile(orgId, planId, resultId, now)

			if err != nil {
				errCh <- fmt.Errorf("error rejecting result: %v", err)
				return
			}

			errCh <- nil
		}(resultId)
	}

	for i := 0; i < len(files); i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error rejecting plan: %v", err)
		}
	}

	return nil
}

func DeletePendingResultsForPaths(orgId, planId string, paths map[string]bool) error {
	// log.Println("Deleting pending results for paths")
	resultsDir := getPlanResultsDir(orgId, planId)
	files, err := os.ReadDir(resultsDir)

	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("error reading results dir: %v", err)
	}

	errCh := make(chan error, len(files))

	for _, file := range files {
		resultId := strings.TrimSuffix(file.Name(), ".json")

		go func(resultId string) {
			bytes, err := os.ReadFile(filepath.Join(resultsDir, resultId+".json"))

			if err != nil {
				errCh <- fmt.Errorf("error reading result file: %v", err)
				return
			}

			var result PlanFileResult
			err = json.Unmarshal(bytes, &result)

			if err != nil {
				errCh <- fmt.Errorf("error unmarshalling result file: %v", err)
				return
			}

			// log.Printf("Checking pending result: %s", resultId)

			if result.ToApi().IsPending() && paths[result.Path] {
				log.Printf("Deleting pending result: %s", resultId)

				err = os.Remove(filepath.Join(resultsDir, resultId+".json"))

				if err != nil {
					errCh <- fmt.Errorf("error deleting result file: %v", err)
					return
				}
			}

			errCh <- nil
		}(resultId)
	}

	for i := 0; i < len(files); i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error deleting pending results: %v", err)
		}
	}

	return nil
}

func RejectPlanFile(orgId, planId, file string, now time.Time) error {
	resultsDir := getPlanResultsDir(orgId, planId)
	results, err := GetPlanFileResults(orgId, planId)

	if err != nil {
		return fmt.Errorf("error getting plan file results: %v", err)
	}

	errCh := make(chan error, len(results))

	for _, result := range results {
		go func(result *PlanFileResult) {
			if result.Path == file && result.AppliedAt == nil && result.RejectedAt == nil {
				result.RejectedAt = &now
			}

			bytes, err := json.MarshalIndent(result, "", "  ")

			if err != nil {
				errCh <- fmt.Errorf("error marshalling result: %v", err)
			}

			err = os.WriteFile(filepath.Join(resultsDir, result.Id+".json"), bytes, 0644)

			if err != nil {
				errCh <- fmt.Errorf("error writing result file: %v", err)
			}

			errCh <- nil
		}(result)
	}

	for i := 0; i < len(results); i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error rejecting plan: %v", err)
		}
	}

	return nil
}

func RejectReplacement(orgId, planId, resultId, replacementId string) error {
	resultsDir := getPlanResultsDir(orgId, planId)

	bytes, err := os.ReadFile(filepath.Join(resultsDir, resultId+".json"))

	if err != nil {
		return fmt.Errorf("error reading result file: %v", err)
	}

	var result PlanFileResult
	err = json.Unmarshal(bytes, &result)

	if err != nil {
		return fmt.Errorf("error unmarshalling result file: %v", err)
	}

	if result.RejectedAt != nil {
		return nil
	}

	now := time.Now()

	foundReplacement := false
	for _, replacement := range result.Replacements {
		if replacement.Id == replacementId {
			replacement.RejectedAt = &now
			foundReplacement = true
			break
		}
	}

	if !foundReplacement {
		return fmt.Errorf("replacement not found: %s", replacementId)
	}

	return nil
}
