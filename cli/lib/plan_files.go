package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/plandex/plandex/shared"
)

func GetCurrentPlanFiles() (*shared.CurrentPlanFiles, error) {
	currentPlanFiles, _, _, err := GetCurrentPlanStateWithContext()
	return currentPlanFiles, err
}

func GetCurrentPlanStateWithContext() (*shared.CurrentPlanFiles, shared.PlanResultsByPath, shared.ModelContext, error) {
	errCh := make(chan error, 1)
	planResByPathCh := make(chan shared.PlanResultsByPath, 1)
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
		planResByPath, err := getPlanResultsByPath()
		if err != nil {
			errCh <- fmt.Errorf("error getting plan results: %v", err)
			return
		}
		planResByPathCh <- planResByPath
	}()

	var planResByPath shared.PlanResultsByPath
	var modelContext shared.ModelContext

	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			return nil, nil, nil, err
		case planResByPath = <-planResByPathCh:
		case modelContext = <-contextCh:
		}
	}

	files := make(map[string]string)
	shas := make(map[string]string)

	for _, contextPart := range modelContext {
		if contextPart.FilePath == "" {
			continue
		}

		// fmt.Printf("contextPart: %s\n", contextPart.FilePath)

		_, hasPath := planResByPath[contextPart.FilePath]

		// fmt.Printf("hasPath: %v\n", hasPath)

		if hasPath {
			files[contextPart.FilePath] = contextPart.Body
			shas[contextPart.FilePath] = contextPart.Sha
		}
	}

	for path, planResults := range planResByPath {
		updated := files[path]
		lastInsertedIdx := 0

		// fmt.Printf("path: %s\n", path)
		// fmt.Printf("updated: %s\n", updated)

		for _, planRes := range planResults {
			if !planRes.IsPending() {
				continue
			}

			if planRes.Content != "" {
				if updated != "" {
					return nil, nil, nil, fmt.Errorf("plan updates out of order: %s", path)
				}

				updated = planRes.Content
				files[path] = updated
				continue
			}

			contextSha := shas[path]
			if contextSha != "" && planRes.ContextSha != contextSha {
				return nil, nil, nil, fmt.Errorf("result sha doesn't match context sha: %s", path)
			}

			if len(planRes.Replacements) == 0 {
				continue
			}

			for i, replacement := range planRes.Replacements {
				if replacement.Failed || replacement.RejectedAt != "" {
					continue
				}

				pre := updated[:lastInsertedIdx]
				sub := updated[lastInsertedIdx:]
				originalIdx := strings.Index(sub, replacement.Old)

				if originalIdx == -1 {
					// replacement failed, return error (checked server-side so this shouldn't happen)

					// fmt.Printf("replacement failed. path: %s, ts: %s, i: %d", path, planRes.Ts, i)
					// fmt.Println("Replacement: " + replacement.Old + " -> " + replacement.New)
					// fmt.Println("Pre: " + pre)
					// fmt.Println("Sub: " + sub)
					// fmt.Println("Idx: " + fmt.Sprintf("%d", lastInsertedIdx))
					// fmt.Println("Updated: " + updated)

					return nil, nil, nil, fmt.Errorf("replacement failed. path: %s, ts: %s, i: %d", path, planRes.Ts, i)
				} else {
					replaced := strings.Replace(sub, replacement.Old, replacement.New, 1)
					updated = pre + replaced
					lastInsertedIdx = lastInsertedIdx + originalIdx + len(replacement.New)
				}
			}
		}

		files[path] = updated
	}

	return &shared.CurrentPlanFiles{Files: files, ContextShas: shas}, planResByPath, modelContext, nil
}

func getPlanResultsByPath() (shared.PlanResultsByPath, error) {
	resByPath := make(shared.PlanResultsByPath)

	_, err := os.Stat(ResultsSubdir)
	resDirExists := !os.IsNotExist(err)

	if !resDirExists {
		return resByPath, nil
	}

	// resFiles, err := os.ReadDir(ResultsSubdir)
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

		resByPath[planRes.Path] = append(resByPath[planRes.Path], &planRes)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error reading results dir: %v", err)
	}

	// sort results by timestamp
	for _, results := range resByPath {
		sort.Slice(results, func(i, j int) bool {
			return results[i].Ts < results[j].Ts
		})
	}

	return resByPath, nil
}

func SetPendingResultsApplied(planResByPath shared.PlanResultsByPath) error {
	ts := shared.StringTs()
	planResByPath.SetApplied(ts)
	numPending := planResByPath.NumPending()

	errCh := make(chan error, numPending)

	for _, planResults := range planResByPath {
		for _, planResult := range planResults {
			if !planResult.IsPending() {
				continue
			}

			go func(planResult *shared.PlanResult) {
				bytes, err := json.Marshal(planResult)
				if err != nil {
					errCh <- fmt.Errorf("error marshalling plan result: %v", err)
					return
				}

				err = os.WriteFile(filepath.Join(ResultsSubdir, planResult.Path, planResult.Ts), bytes, 0644)
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
