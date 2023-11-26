package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/plandex/plandex/shared"
)

func DropChangesWithOutput(name string) error {
	plandexDir, _, err := FindOrCreatePlandex()
	if err != nil {
		return fmt.Errorf("error finding or creating plandex dir: %w", err)
	}

	if name == "" {
		return fmt.Errorf("no plan specified and no current plan")
	}

	rootDir := filepath.Join(plandexDir, name)

	_, err = os.Stat(rootDir)

	if os.IsNotExist(err) {
		return fmt.Errorf("plan with name '%s' doesn't exist", name)
	} else if err != nil {
		return fmt.Errorf("error checking if plan exists: %w", err)
	}

	res, err := GetCurrentPlanState()

	if err != nil {
		return fmt.Errorf("error getting current plan files: %w", err)
	}

	planResByPath := res.PlanResByPath
	ts := shared.StringTs()
	numRejected := planResByPath.SetRejected(ts)
	errCh := make(chan error, numRejected)

	for _, planResults := range planResByPath {
		for _, planResult := range planResults {
			if planResult.RejectedAt != ts {
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

	for i := 0; i < numRejected; i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error wrting plan result file after SetApplied: %v", err)
		}
	}

	fmt.Println("🗑️ Pending changes dropped")

	return nil
}

func DropChange(name, path string, replacement *shared.Replacement) error {
	plandexDir, _, err := FindOrCreatePlandex()
	if err != nil {
		return fmt.Errorf("error finding or creating plandex dir: %w", err)
	}

	if name == "" {
		return fmt.Errorf("no plan specified and no current plan")
	}

	rootDir := filepath.Join(plandexDir, name)

	_, err = os.Stat(rootDir)

	if os.IsNotExist(err) {
		return fmt.Errorf("plan with name '%s' doesn't exist", name)
	} else if err != nil {
		return fmt.Errorf("error checking if plan exists: %w", err)
	}

	res, err := GetCurrentPlanState()

	if err != nil {
		return fmt.Errorf("error getting current plan files: %w", err)
	}

	planResByPath := res.PlanResByPath
	planResults := planResByPath[path]
	ts := shared.StringTs()

	for _, planResult := range planResults {
		for _, resultReplacement := range planResult.Replacements {
			if resultReplacement.Id == replacement.Id {
				replacement.SetRejected(ts)
				break
			}
		}

		bytes, err := json.Marshal(planResult)
		if err != nil {
			return fmt.Errorf("error marshalling plan result: %v", err)
		}

		path := filepath.Join(ResultsSubdir, planResult.Path, planResult.Ts+".json")

		err = os.WriteFile(path, bytes, 0644)
		if err != nil {
			return fmt.Errorf("error writing plan result: %v", err)
		}
	}

	return fmt.Errorf("change with ID '%s' doesn't exist in plan '%s' at path '%s'", replacement.Id, name, path)

}
