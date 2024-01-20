package db

import (
	"encoding/json"
	"fmt"
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
	Contexts                 []*Context
	PlanFileResults          []*PlanFileResult
	PendingBuildDescriptions []*ConvoMessageDescription
}

func GetCurrentPlanState(params CurrentPlanStateParams) (*shared.CurrentPlanState, error) {
	orgId := params.OrgId
	planId := params.PlanId

	var dbPlanFileResults []*PlanFileResult
	var pendingBuildDescriptions []*shared.ConvoMessageDescription
	var contexts []*Context

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
		if params.PendingBuildDescriptions == nil {
			res, err := GetPendingBuildDescriptions(orgId, planId)
			if err != nil {
				errCh <- fmt.Errorf("error getting latest plan build description: %v", err)
				return
			}

			for _, desc := range res {
				pendingBuildDescriptions = append(pendingBuildDescriptions, desc.ToApi())
			}
		} else {
			for _, desc := range params.PendingBuildDescriptions {
				pendingBuildDescriptions = append(pendingBuildDescriptions, desc.ToApi())
			}
		}

		errCh <- nil
	}()

	go func() {
		if params.Contexts == nil {
			res, err := GetPlanContexts(orgId, planId, true)
			if err != nil {
				errCh <- fmt.Errorf("error getting plan contexts: %v", err)
				return
			}
			contexts = res
		} else {
			contexts = params.Contexts
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

	for _, dbPlanFileResult := range dbPlanFileResults {
		apiPlanFileResults = append(apiPlanFileResults, dbPlanFileResult.ToApi())
	}
	planResult := GetPlanResult(apiPlanFileResults)

	var apiContexts []*shared.Context
	apiContextsByPath := make(map[string]*shared.Context)

	for _, context := range contexts {
		apiContexts = append(apiContexts, context.ToApi())

		if context.FilePath != "" {
			apiContextsByPath[context.FilePath] = context.ToApi()
		}
	}

	planState := &shared.CurrentPlanState{
		PlanResult:               planResult,
		Contexts:                 apiContexts,
		ContextsByPath:           apiContextsByPath,
		PendingBuildDescriptions: pendingBuildDescriptions,
	}

	currentPlanFiles, err := planState.GetFiles()

	if err != nil {
		return nil, fmt.Errorf("error getting current plan files: %v", err)
	}

	planState.CurrentPlanFiles = currentPlanFiles

	return planState, nil
}

func GetPendingBuildDescriptions(orgId, planId string) ([]*ConvoMessageDescription, error) {
	descriptionsDir := getPlanDescriptionsDir(orgId, planId)

	files, err := os.ReadDir(descriptionsDir)

	if err != nil {
		return nil, fmt.Errorf("error reading descriptions dir: %v", err)
	}

	var descriptions []*ConvoMessageDescription
	errCh := make(chan error, len(files))
	descCh := make(chan *ConvoMessageDescription, len(files))

	for _, file := range files {
		go func(file os.DirEntry) {
			bytes, err := os.ReadFile(filepath.Join(descriptionsDir, file.Name()))

			if err != nil {
				errCh <- fmt.Errorf("error reading description file: %v", err)
				return
			}

			var description ConvoMessageDescription
			err = json.Unmarshal(bytes, &description)

			if err != nil {
				errCh <- fmt.Errorf("error unmarshalling description file: %v", err)
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

func ApplyPlan(orgId, planId string) error {
	resultsDir := getPlanResultsDir(orgId, planId)

	errCh := make(chan error)

	var results []*PlanFileResult
	var pendingBuildDescriptions []*ConvoMessageDescription
	var contexts []*Context

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
		res, err := GetPendingBuildDescriptions(orgId, planId)
		if err != nil {
			errCh <- fmt.Errorf("error getting latest plan build description: %v", err)
			return
		}
		pendingBuildDescriptions = res
		errCh <- nil
	}()

	go func() {
		res, err := GetPlanContexts(orgId, planId, false)
		if err != nil {
			errCh <- fmt.Errorf("error getting plan contexts: %v", err)
			return
		}
		contexts = res
		errCh <- nil
	}()

	for i := 0; i < 3; i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error applying plan: %v", err)
		}
	}

	planState, err := GetCurrentPlanState(CurrentPlanStateParams{
		OrgId:                    orgId,
		PlanId:                   planId,
		Contexts:                 contexts,
		PlanFileResults:          results,
		PendingBuildDescriptions: pendingBuildDescriptions,
	})

	if err != nil {
		return fmt.Errorf("error getting current plan state: %v", err)
	}

	var pendingDbResults []*PlanFileResult

	for _, result := range results {
		apiResult := result.ToApi()
		if apiResult.IsPending() {
			pendingDbResults = append(pendingDbResults, result)
		}
	}

	errCh = make(chan error, len(pendingDbResults)+len(pendingBuildDescriptions))
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

	for _, description := range pendingBuildDescriptions {
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

	for i := 0; i < len(pendingDbResults)+len(pendingBuildDescriptions); i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error applying plan: %v", err)
		}
	}

	msg := "Marked pending results as applied." + "\n\n" + planState.PendingChangesSummary()

	err = GitAddAndCommit(orgId, planId, msg)

	if err != nil {
		return fmt.Errorf("error committing plan: %v", err)
	}

	return nil
}

func RejectAllResults(orgId, planId string) error {
	resultsDir := getPlanResultsDir(orgId, planId)

	files, err := os.ReadDir(resultsDir)

	if err != nil {
		return fmt.Errorf("error reading results dir: %v", err)
	}

	errCh := make(chan error, len(files))
	now := time.Now()

	for _, file := range files {
		resultId := strings.TrimSuffix(file.Name(), ".json")

		go func(resultId string) {
			err := RejectPlanFileResult(orgId, planId, resultId, now)

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

func RejectPlanFileResult(orgId, planId, resultId string, now time.Time) error {
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

	result.RejectedAt = &now

	bytes, err = json.MarshalIndent(result, "", "  ")

	if err != nil {
		return fmt.Errorf("error marshalling result: %v", err)
	}

	err = os.WriteFile(filepath.Join(resultsDir, resultId+".json"), bytes, 0644)

	if err != nil {
		return fmt.Errorf("error writing result file: %v", err)
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
