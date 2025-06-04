package db

import (
	"context"
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
	"time"

	shared "plandex-shared"

	"github.com/google/uuid"
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

	log.Printf("Storing plan result: %s - %s", result.Path, result.Id)

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

func GetFullCurrentPlanStateParams(orgId, planId string) (CurrentPlanStateParams, error) {
	errCh := make(chan error, 3)

	var results []*PlanFileResult
	var convoMessageDescriptions []*ConvoMessageDescription
	var contexts []*Context

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in GetFullCurrentPlanStateParams: %v\n%s", r, debug.Stack())
				errCh <- fmt.Errorf("panic in GetFullCurrentPlanStateParams: %v\n%s", r, debug.Stack())
				runtime.Goexit() // don't allow outer function to continue and double-send to channel
			}
		}()

		res, err := GetPlanFileResults(orgId, planId)
		if err != nil {
			errCh <- fmt.Errorf("error getting plan file results: %v", err)
			return
		}
		results = res
		errCh <- nil
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in GetFullCurrentPlanStateParams: %v\n%s", r, debug.Stack())
				errCh <- fmt.Errorf("panic in GetFullCurrentPlanStateParams: %v\n%s", r, debug.Stack())
				runtime.Goexit() // don't allow outer function to continue and double-send to channel
			}
		}()
		res, err := GetConvoMessageDescriptions(orgId, planId)
		if err != nil {
			errCh <- fmt.Errorf("error getting latest plan build description: %v", err)
			return
		}
		convoMessageDescriptions = res
		errCh <- nil
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in GetFullCurrentPlanStateParams: %v\n%s", r, debug.Stack())
				errCh <- fmt.Errorf("panic in GetFullCurrentPlanStateParams: %v\n%s", r, debug.Stack())
				runtime.Goexit() // don't allow outer function to continue and double-send to channel
			}
		}()
		res, err := GetPlanContexts(orgId, planId, true, false)
		if err != nil {
			errCh <- fmt.Errorf("error getting contexts: %v", err)
			return
		}

		contexts = res

		errCh <- nil
	}()

	for i := 0; i < 3; i++ {
		err := <-errCh
		if err != nil {
			return CurrentPlanStateParams{}, err
		}
	}

	return CurrentPlanStateParams{
		OrgId:                    orgId,
		PlanId:                   planId,
		PlanFileResults:          results,
		ConvoMessageDescriptions: convoMessageDescriptions,
		Contexts:                 contexts,
	}, nil
}

func GetCurrentPlanState(params CurrentPlanStateParams) (*shared.CurrentPlanState, error) {
	orgId := params.OrgId
	planId := params.PlanId

	var dbPlanFileResults []*PlanFileResult
	var convoMessageDescriptions []*shared.ConvoMessageDescription
	contextsByPath := map[string]*Context{}
	planApplies := []*shared.PlanApply{}
	errCh := make(chan error, 4)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in GetCurrentPlanState: %v\n%s", r, debug.Stack())
				errCh <- fmt.Errorf("panic in GetCurrentPlanState: %v\n%s", r, debug.Stack())
				runtime.Goexit() // don't allow outer function to continue and double-send to channel
			}
		}()
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
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in GetCurrentPlanState: %v\n%s", r, debug.Stack())
				errCh <- fmt.Errorf("panic in GetCurrentPlanState: %v\n%s", r, debug.Stack())
				runtime.Goexit() // don't allow outer function to continue and double-send to channel
			}
		}()
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
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in GetCurrentPlanState: %v\n%s", r, debug.Stack())
				errCh <- fmt.Errorf("panic in GetCurrentPlanState: %v\n%s", r, debug.Stack())
				runtime.Goexit() // don't allow outer function to continue and double-send to channel
			}
		}()
		var contexts []*Context
		if params.Contexts == nil {
			res, err := GetPlanContexts(orgId, planId, true, false)
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

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in GetCurrentPlanState: %v\n%s", r, debug.Stack())
				errCh <- fmt.Errorf("panic in GetCurrentPlanState: %v\n%s", r, debug.Stack())
				runtime.Goexit() // don't allow outer function to continue and double-send to channel
			}
		}()
		res, err := GetPlanApplies(orgId, planId)
		if err != nil {
			errCh <- fmt.Errorf("error getting plan applies: %v", err)
			return
		}

		for _, apply := range res {
			planApplies = append(planApplies, apply.ToApi())
		}

		errCh <- nil
	}()

	for i := 0; i < 4; i++ {
		err := <-errCh
		if err != nil {
			return nil, err
		}
	}

	var apiPlanFileResults []*shared.PlanFileResult
	pendingResultPaths := map[string]bool{}

	for _, dbPlanFileResult := range dbPlanFileResults {
		// log.Printf("Plan file result: %s", dbPlanFileResult.Id)

		apiResult := dbPlanFileResult.ToApi()
		apiPlanFileResults = append(apiPlanFileResults, apiResult)

		if apiResult.IsPending() {
			// log.Printf("Pending result: %s", dbPlanFileResult.Id)

			pendingResultPaths[apiResult.Path] = true
		} else {
			// log.Printf("Not pending result: %s", apiResult.Id)
			// log.Printf("Applied at: %v", apiResult.AppliedAt)
			// log.Printf("Rejected at: %v", apiResult.RejectedAt)
			// log.Printf("Content: %v", apiResult.Content != "")
			// log.Printf("Num Replacement: %d", len(apiResult.Replacements))
			// log.Printf("Num Pending Replacements: %v", apiResult.NumPendingReplacements())
		}
	}
	planResult := GetPlanResult(apiPlanFileResults)

	pendingContextsByPath := map[string]*shared.Context{}
	for path, context := range contextsByPath {
		pendingContextsByPath[path] = context.ToApi()
	}

	// log.Println("Pending contexts by path:", len(pendingContextsByPath))

	planState := &shared.CurrentPlanState{
		PlanResult:               planResult,
		ConvoMessageDescriptions: convoMessageDescriptions,
		ContextsByPath:           pendingContextsByPath,
		PlanApplies:              planApplies,
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
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in GetConvoMessageDescriptions: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in GetConvoMessageDescriptions: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
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
			if description.WroteFiles && description.AppliedAt == nil {
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
		// log.Printf("Result file: %s", file.Name())

		go func(file os.DirEntry) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in GetPlanFileResults: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in GetPlanFileResults: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()

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

func GetPlanFileResultById(orgId, planId, resultId string) (*PlanFileResult, error) {
	resultsDir := getPlanResultsDir(orgId, planId)

	bytes, err := os.ReadFile(filepath.Join(resultsDir, resultId+".json"))

	if err != nil {
		return nil, fmt.Errorf("error reading result file: %v", err)
	}

	var result PlanFileResult
	err = json.Unmarshal(bytes, &result)

	if err != nil {
		return nil, fmt.Errorf("error unmarshalling result file: %v", err)
	}

	return &result, nil
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
				// log.Printf("Adding res path: %s", planFileRes.Path)
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

type ApplyPlanParams struct {
	OrgId                  string
	UserId                 string
	BranchName             string
	Plan                   *Plan
	CurrentPlanState       *shared.CurrentPlanState
	CurrentPlanStateParams *CurrentPlanStateParams
	CommitMsg              string
}

func ApplyPlan(repo *GitRepo, ctx context.Context, params ApplyPlanParams) error {
	orgId := params.OrgId
	userId := params.UserId
	branchName := params.BranchName
	plan := params.Plan
	currentPlanState := params.CurrentPlanState
	currentPlanParams := params.CurrentPlanStateParams
	planId := plan.Id
	resultsDir := getPlanResultsDir(orgId, planId)

	var pendingDbResults []*PlanFileResult

	planFileResults := currentPlanParams.PlanFileResults
	convoMessageDescriptions := currentPlanParams.ConvoMessageDescriptions
	contexts := currentPlanParams.Contexts

	contextsByPath := make(map[string]*Context)
	for _, context := range contexts {
		if context.FilePath != "" {
			contextsByPath[context.FilePath] = context
		}
	}

	for _, result := range planFileResults {
		apiResult := result.ToApi()
		if apiResult.IsPending() {
			pendingDbResults = append(pendingDbResults, result)
		}
	}

	log.Printf("Pending db results: %d", len(pendingDbResults))

	pendingNewFilesSet := make(map[string]bool)
	pendingUpdatedFilesSet := make(map[string]bool)
	for _, result := range pendingDbResults {
		if result.Path == "_apply.sh" {
			continue
		}

		if len(result.Replacements) == 0 && result.Content != "" {
			pendingNewFilesSet[result.Path] = true
		} else if !pendingNewFilesSet[result.Path] {
			pendingUpdatedFilesSet[result.Path] = true
		}
	}

	var loadContextRes *shared.LoadContextResponse
	var updateContextRes *shared.UpdateContextResponse

	numRoutines := len(pendingDbResults) +
		len(convoMessageDescriptions)

	if len(pendingNewFilesSet) > 0 {
		numRoutines++
	}
	if len(pendingUpdatedFilesSet) > 0 {
		numRoutines++
	}

	errCh := make(chan error, numRoutines)
	now := time.Now()

	for _, result := range pendingDbResults {
		go func(result *PlanFileResult) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in ApplyPlan: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in ApplyPlan: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
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
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in ApplyPlan: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in ApplyPlan: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
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
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in ApplyPlan: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in ApplyPlan: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			loadReq := shared.LoadContextRequest{}
			for path := range pendingNewFilesSet {
				loadReq = append(loadReq, &shared.LoadContextParams{
					ContextType: shared.ContextFileType,
					Name:        path,
					FilePath:    path,
					Body:        currentPlanState.CurrentPlanFiles.Files[path],
				})
			}

			if len(loadReq) > 0 {
				res, _, err := LoadContexts(
					ctx,
					LoadContextsParams{
						OrgId:                    orgId,
						UserId:                   userId,
						Plan:                     plan,
						BranchName:               branchName,
						Req:                      &loadReq,
						SkipConflictInvalidation: true, // no need to invalidate conflicts when applying plan--and fixes race condition since invalidation check loads description
						AutoLoaded:               true,
					},
				)

				if err != nil {
					errCh <- fmt.Errorf("error loading context: %v", err)
					return
				}

				loadContextRes = res
			}

			errCh <- nil
		}()
	}

	if len(pendingUpdatedFilesSet) > 0 {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in ApplyPlan: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in ApplyPlan: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			updateReq := shared.UpdateContextRequest{}
			for path := range pendingUpdatedFilesSet {
				context := contextsByPath[path]
				updateReq[context.Id] = &shared.UpdateContextParams{
					Body: currentPlanState.CurrentPlanFiles.Files[path],
				}
			}

			if len(updateReq) > 0 {
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
			}
			errCh <- nil

		}()

	}

	for i := 0; i < numRoutines; i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error applying plan: %v", err)
		}
	}

	// Store the PlanApply record
	planApply := &PlanApply{
		Id:        uuid.New().String(),
		OrgId:     orgId,
		PlanId:    planId,
		UserId:    userId,
		CommitMsg: params.CommitMsg,
		CreatedAt: now,
	}

	// Collect the IDs from the pending results and descriptions
	var resultIds []string
	var descriptionIds []string
	var messageIds []string

	for _, result := range pendingDbResults {
		resultIds = append(resultIds, result.Id)
	}
	for _, desc := range convoMessageDescriptions {
		descriptionIds = append(descriptionIds, desc.Id)
		messageIds = append(messageIds, desc.ConvoMessageId)
	}

	planApply.PlanFileResultIds = resultIds
	planApply.ConvoMessageDescriptionIds = descriptionIds
	planApply.ConvoMessageIds = messageIds

	// Store the PlanApply object
	bytes, err := json.MarshalIndent(planApply, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling plan apply: %v", err)
	}

	appliesDir := getPlanAppliesDir(orgId, planId)
	err = os.MkdirAll(appliesDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating applies dir: %v", err)
	}

	err = os.WriteFile(filepath.Join(appliesDir, planApply.Id+".json"), bytes, 0644)
	if err != nil {
		return fmt.Errorf("error writing plan apply file: %v", err)
	}

	msg := "✅ Marked pending results as applied"

	currentFiles := currentPlanState.CurrentPlanFiles.Files
	var sortedFiles []string
	for path := range currentFiles {
		sortedFiles = append(sortedFiles, path)
	}
	sort.Strings(sortedFiles)
	for _, path := range sortedFiles {
		msg += fmt.Sprintf("\n • 📄 %s", path)
	}
	msg += "\n" + "✏️  " + params.CommitMsg

	if loadContextRes != nil && !loadContextRes.MaxTokensExceeded {
		msg += "\n\n" + loadContextRes.Msg
	}

	if updateContextRes != nil && !updateContextRes.MaxTokensExceeded {
		msg += "\n\n" + updateContextRes.Msg
	}

	err = repo.GitAddAndCommit(branchName, msg)

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
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in RejectAllResults: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in RejectAllResults: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
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
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in DeletePendingResultsForPaths: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in DeletePendingResultsForPaths: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
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

func RejectPlanFiles(orgId, planId string, files []string, now time.Time) error {
	errCh := make(chan error, len(files))

	for _, file := range files {
		go func(file string) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in RejectPlanFiles: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in RejectPlanFiles: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			err := RejectPlanFile(orgId, planId, file, now)

			if err != nil {
				errCh <- err
				return
			}

			errCh <- nil
		}(file)
	}

	for i := 0; i < len(files); i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error rejecting plan files: %v", err)
		}
	}

	return nil
}

func RejectPlanFile(orgId, planId, filePathOrResultId string, now time.Time) error {
	resultsDir := getPlanResultsDir(orgId, planId)
	results, err := GetPlanFileResults(orgId, planId)

	if err != nil {
		return fmt.Errorf("error getting plan file results: %v", err)
	}

	errCh := make(chan error, len(results))

	for _, result := range results {
		go func(result *PlanFileResult) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in RejectPlanFile: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in RejectPlanFile: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			if (result.Path == filePathOrResultId || result.Id == filePathOrResultId) && result.AppliedAt == nil && result.RejectedAt == nil {
				result.RejectedAt = &now
			} else {
				errCh <- nil
				return
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

func GetPlanApplies(orgId, planId string) ([]*PlanApply, error) {
	appliesDir := getPlanAppliesDir(orgId, planId)
	files, err := os.ReadDir(appliesDir)

	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("error reading applies dir: %v", err)
	}

	planApplies := []*PlanApply{}
	var mu sync.Mutex

	errCh := make(chan error, len(files))

	for _, file := range files {
		go func(file os.DirEntry) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in GetPlanApplies: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in GetPlanApplies: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			bytes, err := os.ReadFile(filepath.Join(appliesDir, file.Name()))

			if err != nil {
				errCh <- fmt.Errorf("error reading apply file: %v", err)
				return
			}

			var apply PlanApply
			err = json.Unmarshal(bytes, &apply)

			if err != nil {
				errCh <- fmt.Errorf("error unmarshalling apply file: %v", err)
				return
			}

			mu.Lock()
			planApplies = append(planApplies, &apply)
			mu.Unlock()

			errCh <- nil
		}(file)
	}

	for i := 0; i < len(files); i++ {
		err := <-errCh
		if err != nil {
			return nil, fmt.Errorf("error getting plan applies: %v", err)
		}
	}

	return planApplies, nil
}
