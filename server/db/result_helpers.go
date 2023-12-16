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

func GetCurrentPlanState(orgId, planId string, contexts []*Context) (*shared.CurrentPlanState, error) {

	var dbPlanFileResults []*PlanFileResult
	var latestBuildDesc *ConvoMessageDescription
	errCh := make(chan error, 2)

	go func() {
		res, err := GetPlanFileResults(orgId, planId)
		dbPlanFileResults = res

		if err != nil {
			errCh <- fmt.Errorf("error getting plan file results: %v", err)
			return
		}

		errCh <- nil
	}()

	go func() {
		res, err := GetLatestBuildDescription(planId)
		latestBuildDesc = res

		if err != nil {
			errCh <- fmt.Errorf("error getting latest plan build description: %v", err)
			return
		}

		errCh <- nil
	}()

	for i := 0; i < 2; i++ {
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
		PlanResult:             planResult,
		Contexts:               apiContexts,
		ContextsByPath:         apiContextsByPath,
		LatestBuildDescription: latestBuildDesc.ToApi(),
	}

	currentPlanFiles, err := planState.GetFiles()

	if err != nil {
		return nil, fmt.Errorf("error getting current plan files: %v", err)
	}

	planState.CurrentPlanFiles = currentPlanFiles

	return planState, nil
}

func GetLatestBuildDescription(planId string) (*ConvoMessageDescription, error) {
	var description ConvoMessageDescription

	err := Conn.Get(&description, "SELECT id, org_id, plan_id, convo_message_id, summarized_to_message_created_at, made_plan, commit_msg, files, error, created_at, updated_at FROM convo_message_descriptions WHERE plan_id = $1 AND made_plan = true ORDER BY created_at DESC LIMIT 1", planId)

	if err != nil {
		return nil, fmt.Errorf("error getting plan build description: %v", err)
	}

	return &description, nil
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
	}
}

func ApplyPlan(orgId, planId string) error {
	resultsDir := getPlanResultsDir(orgId, planId)

	files, err := os.ReadDir(resultsDir)

	if err != nil {
		return fmt.Errorf("error reading results dir: %v", err)
	}

	errCh := make(chan error, len(files))

	now := time.Now()

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

			if result.AppliedAt != nil {
				errCh <- nil
				return
			}

			result.AppliedAt = &now

			bytes, err = json.MarshalIndent(result, "", "  ")

			if err != nil {
				errCh <- fmt.Errorf("error marshalling result: %v", err)
				return
			}

			err = os.WriteFile(filepath.Join(resultsDir, file.Name()), bytes, 0644)

			if err != nil {
				errCh <- fmt.Errorf("error writing result file: %v", err)
				return
			}

			errCh <- nil

		}(file)
	}

	for i := 0; i < len(files); i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error applying plan: %v", err)
		}
	}

	_, err = Conn.Exec("UPDATE plans SET applied_at = NOW() WHERE id = $1", planId)

	if err != nil {
		return fmt.Errorf("error updating plan: %v", err)
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
