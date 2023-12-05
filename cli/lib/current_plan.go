package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex/types"
	"sort"

	"github.com/plandex/plandex/shared"
)

func GetCurrentPlanFiles() (*shared.CurrentPlanFiles, error) {
	res, err := GetCurrentPlanState()
	if err != nil {
		return nil, err
	}
	return res.CurrentPlanFiles, err
}

func GetCurrentPlanState() (*types.CurrentPlanState, error) {
	return GetCurrentPlanStateBeforeReplacement("")
}

func GetCurrentPlanStateBeforeReplacement(id string) (*types.CurrentPlanState, error) {
	errCh := make(chan error, 1)
	planResInfoCh := make(chan *types.PlanResultsInfo, 1)
	contextCh := make(chan shared.ModelContext, 1)

	go func() {
		modelContext, err := GetAllContext(false)
		if err != nil {
			errCh <- fmt.Errorf("error loading context: %v", err)
			return
		}
		contextCh <- modelContext
	}()

	go func() {
		planResInfo, err := GetPlanResultsInfo()
		if err != nil {
			errCh <- fmt.Errorf("error getting plan results: %v", err)
			return
		}

		planResInfoCh <- planResInfo
	}()

	var planResInfo *types.PlanResultsInfo
	var modelContext shared.ModelContext

	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			return nil, err
		case planResInfo = <-planResInfoCh:
		case modelContext = <-contextCh:
		}
	}

	files := make(map[string]string)
	shas := make(map[string]string)

	for _, contextPart := range modelContext {
		if contextPart.FilePath == "" {
			continue
		}

		_, hasPath := planResInfo.PlanResByPath[contextPart.FilePath]

		// fmt.Printf("hasPath: %v\n", hasPath)

		if hasPath {
			files[contextPart.FilePath] = contextPart.Body
			shas[contextPart.FilePath] = contextPart.Sha
		}
	}

	for path, planResults := range planResInfo.PlanResByPath {
		updated := files[path]

		// fmt.Printf("path: %s\n", path)
		// fmt.Printf("updated: %s\n", updated)

	PlanResLoop:
		for _, planRes := range planResults {
			if !planRes.IsPending() {
				continue
			}

			if len(planRes.Replacements) == 0 {
				if updated != "" {
					return nil, fmt.Errorf("plan updates out of order: %s", path)
				}

				updated = planRes.Content
				files[path] = updated
				continue
			}

			contextSha := shas[path]

			if contextSha != "" && planRes.ContextSha != contextSha {
				return nil, fmt.Errorf("result sha doesn't match context sha: %s", path)
			}

			if len(planRes.Replacements) == 0 {
				continue
			}

			replacements := []*shared.Replacement{}
			for _, replacement := range planRes.Replacements {
				if replacement.Id == id {
					break PlanResLoop
				}
				replacements = append(replacements, replacement)
			}

			var allSucceeded bool
			updated, allSucceeded = shared.ApplyReplacements(updated, replacements, false)

			if !allSucceeded {
				return nil, fmt.Errorf("plan replacement failed: %s", path)
			}
		}

		files[path] = updated
	}

	return &types.CurrentPlanState{
		CurrentPlanFiles: &shared.CurrentPlanFiles{Files: files, ContextShas: shas},
		ModelContext:     modelContext,
		ContextByPath:    modelContext.ByPath(),
		PlanResultsInfo:  *planResInfo,
	}, nil
}

func SetPendingResultsApplied(planResByPath shared.PlanResultsByPath) error {
	ts := shared.StringTs()
	numPending := planResByPath.NumPending()
	planResByPath.SetApplied(ts)

	errCh := make(chan error, numPending)

	for _, planResults := range planResByPath {
		for _, planResult := range planResults {
			// only write results that were just applied
			if planResult.AppliedAt != ts {
				continue
			}

			go func(planResult *shared.PlanResult) {
				bytes, err := json.Marshal(planResult)
				if err != nil {
					errCh <- fmt.Errorf("error marshalling plan result: %v", err)
					return
				}

				path := filepath.Join(ResultsSubdir, planResult.Path, planResult.Ts+".json")

				err = os.WriteFile(path, bytes, 0644)
				if err != nil {
					errCh <- fmt.Errorf("error writing plan result: %v", err)
					return
				}

				errCh <- nil

			}(planResult)
		}
	}

	for i := 0; i < numPending; i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error wrting plan result file after SetApplied: %v", err)
		}
	}

	return nil
}

func GetPlanResultsInfo() (*types.PlanResultsInfo, error) {
	resByPath := make(shared.PlanResultsByPath)
	replacementsByPath := make(map[string][]*shared.Replacement)
	var paths []string

	_, err := os.Stat(ResultsSubdir)
	resDirExists := !os.IsNotExist(err)

	if !resDirExists {
		return &types.PlanResultsInfo{
			PlanResByPath:      resByPath,
			ReplacementsByPath: replacementsByPath,
		}, nil
	}

	err = filepath.Walk(ResultsSubdir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error reading results dir: %v", err)
		}

		if info.IsDir() {
			return nil
		}

		bytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading file %s: %v", path, err)
		}

		var planRes shared.PlanResult
		err = json.Unmarshal(bytes, &planRes)
		if err != nil {
			return fmt.Errorf("error unmarshalling plan result JSON from file %s: %v", path, err)
		}

		if planRes.IsPending() {
			_, hasPath := resByPath[planRes.Path]

			resByPath[planRes.Path] = append(resByPath[planRes.Path], &planRes)

			if !hasPath {
				paths = append(paths, planRes.Path)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error reading results dir: %v", err)
	}

	for _, results := range resByPath {
		// sort results by timestamp ascending
		sort.Slice(results, func(i, j int) bool {
			return results[i].Ts < results[j].Ts
		})

		for _, planRes := range results {
			replacementsByPath[planRes.Path] = append(replacementsByPath[planRes.Path], planRes.Replacements...)
		}
	}

	// sort paths ascending
	sort.Slice(paths, func(i, j int) bool {
		return paths[i] < paths[j]
	})

	return &types.PlanResultsInfo{
		PlanResByPath:      resByPath,
		SortedPaths:        paths,
		ReplacementsByPath: replacementsByPath,
	}, nil
}
