package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex/types"
	"regexp"
	"sort"

	"github.com/plandex/plandex/shared"
)

func SetCurrentPlan(name string) error {
	if PlandexDir == "" {
		return fmt.Errorf("PlandexDir not set")
	}

	currentPlanFilePath := filepath.Join(PlandexDir, "current_plan.json")
	err := os.WriteFile(currentPlanFilePath, []byte(fmt.Sprintf(`{"name": "%s"}`, name)), 0644)
	if err != nil {
		return err
	}

	return nil
}

func ClearCurrentPlan() error {
	if PlandexDir == "" {
		return fmt.Errorf("PlandexDir not set")
	}

	currentPlanFilePath := filepath.Join(PlandexDir, "current_plan.json")
	err := os.Remove(currentPlanFilePath)
	if err != nil {
		return err
	}

	return nil
}

var draftRegex = regexp.MustCompile("^draft(\\.[0-9]+)?$")

func ClearDraftPlans() error {
	if PlandexDir == "" {
		return fmt.Errorf("PlandexDir not set")
	}

	files, err := os.ReadDir(PlandexDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if draftRegex.MatchString(f.Name()) {
			err = os.RemoveAll(filepath.Join(PlandexDir, f.Name()))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func CurrentPlanIsDraft() bool {
	if PlandexDir == "" {
		return false
	}
	if CurrentPlanName == "" {
		return false
	}

	return draftRegex.MatchString(CurrentPlanName)
}

func RenameCurrentDraftPlan(name string) error {
	if PlandexDir == "" {
		return fmt.Errorf("PlandexDir not set")
	}
	if CurrentPlanName == "" {
		return fmt.Errorf("CurrentPlanName not set")
	}

	if CurrentPlanIsDraft() {
		name, err := DedupPlanName(name)
		if err != nil {
			return fmt.Errorf("failed to deduplicate plan name: %w", err)
		}

		oldPath := filepath.Join(PlandexDir, CurrentPlanName)
		newPath := filepath.Join(PlandexDir, name)

		err = os.Rename(oldPath, newPath)
		if err != nil {
			return fmt.Errorf("failed to rename plan directory: %w", err)
		}

		err = SetCurrentPlan(name)
		if err != nil {
			return fmt.Errorf("failed to set current plan: %w", err)
		}

		// Fixes current plan paths (CurrentPlanName, CurrentPlanRootDir, etc.)
		err = LoadCurrentPlan()
		if err != nil {
			return fmt.Errorf("failed to load current plan: %w", err)
		}

		planState, err := GetPlanState()
		if err != nil {
			return fmt.Errorf("failed to get plan state: %w", err)
		}

		planState.Name = name
		err = SetPlanState(planState, shared.StringTs())
		if err != nil {
			return fmt.Errorf("failed to set plan state: %w", err)
		}
	}

	return nil
}

func DedupPlanName(name string) (string, error) {
	if PlandexDir == "" {
		return "", fmt.Errorf("PlandexDir not set")
	}

	// If 'name' directory already exists, tack on an integer to differentiate
	planDir := filepath.Join(PlandexDir, name)
	_, err := os.Stat(planDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
	}
	exists := !os.IsNotExist(err)

	postfix := 1
	var nameWithPostfix string
	for exists {
		postfix += 1
		nameWithPostfix = fmt.Sprintf("%s.%d", name, postfix)
		planDir = filepath.Join(PlandexDir, nameWithPostfix)
		_, err = os.Stat(planDir)
		if err != nil {
			if !os.IsNotExist(err) {
				return "", err
			}
		}
		exists = !os.IsNotExist(err)
	}
	if nameWithPostfix != "" {
		name = nameWithPostfix
	}

	return name, nil
}

func GetPlanState() (*types.PlanState, error) {
	return GetPlanStateForPlan(CurrentPlanName)
}

func GetPlanStateForPlan(name string) (*types.PlanState, error) {
	if PlandexDir == "" {
		return nil, fmt.Errorf("PlandexDir not set")
	}

	var state types.PlanState
	planStatePath := filepath.Join(PlandexDir, name, "plan.json")
	fileBytes, err := os.ReadFile(planStatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plan state file: %s", err)
	}
	err = json.Unmarshal(fileBytes, &state)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan state json: %s", err)
	}

	return &state, nil
}

func SetPlanState(state *types.PlanState, updatedAt string) error {
	if updatedAt != "" {
		state.UpdatedAt = updatedAt
	}

	planStatePath := filepath.Join(CurrentPlanRootDir, "plan.json")
	planStateBytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan state: %s", err)
	}
	err = os.WriteFile(planStatePath, planStateBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write plan state: %s", err)
	}
	return nil
}

func GetPlans() ([]*types.PlanState, error) {
	if PlandexDir == "" {
		return nil, fmt.Errorf("PlandexDir not set")
	}

	var plans []*types.PlanState

	planContents, err := os.ReadDir(PlandexDir)

	if err != nil {
		return nil, fmt.Errorf("failed to read plandex directory: %s", err)
	}

	planCh := make(chan *types.PlanState, len(planContents))
	errCh := make(chan error, 1)

	for _, planFileOrDir := range planContents {
		if planFileOrDir.IsDir() {
			go func(planDir os.DirEntry) {
				planName := planDir.Name()

				planState, err := GetPlanStateForPlan(planName)
				if err != nil {
					errCh <- fmt.Errorf("failed to get plan state for plan %s: %s", planName, err)
					return
				}

				planCh <- planState
			}(planFileOrDir)
		} else {
			planCh <- nil
		}
	}

	for range planContents {
		select {
		case plan := <-planCh:
			if plan != nil {
				plans = append(plans, plan)
			}
		case err := <-errCh:
			return nil, err
		}
	}

	// sort plans by UpdatedAt descending
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].UpdatedAt > plans[j].UpdatedAt
	})

	return plans, nil
}
