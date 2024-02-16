package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex/fs"
	"plandex/types"
	"sync"
)

func WriteCurrentPlan(id string) error {
	if fs.HomePlandexDir == "" {
		return fmt.Errorf("HomePlandexDir not set")
	}

	if CurrentProjectId == "" || HomeCurrentPlanPath == "" {
		return fmt.Errorf("no current project")
	}

	settings := types.CurrentPlanSettings{
		Id: id,
	}
	bytes, err := json.Marshal(settings)

	if err != nil {
		return fmt.Errorf("error marshalling current plan: %v", err)
	}

	err = os.WriteFile(HomeCurrentPlanPath, bytes, 0644)
	if err != nil {
		return fmt.Errorf("error writing current plan: %v", err)
	}

	CurrentPlanId = id

	return nil
}

func ClearCurrentPlan() error {
	if fs.HomePlandexDir == "" {
		return fmt.Errorf("HomePlandexDir not set")
	}

	if CurrentProjectId == "" || HomeCurrentPlanPath == "" {
		return fmt.Errorf("no current project")
	}

	err := os.Remove(HomeCurrentPlanPath)
	if err != nil {
		return err
	}

	CurrentPlanId = ""

	return nil
}

func WriteCurrentBranch(branch string) error {
	if fs.HomePlandexDir == "" {
		return fmt.Errorf("HomePlandexDir not set")
	}

	if CurrentProjectId == "" || HomeCurrentPlanPath == "" {
		return fmt.Errorf("no current project")
	}

	if CurrentPlanId == "" {
		return fmt.Errorf("no current plan")
	}

	settings := types.PlanSettings{
		Branch: branch,
	}

	bytes, err := json.Marshal(settings)

	if err != nil {
		return fmt.Errorf("error marshalling current plan settings: %v", err)
	}

	dir := filepath.Join(fs.HomePlandexDir, CurrentProjectId, CurrentPlanId)

	err = os.MkdirAll(dir, os.ModePerm)

	if err != nil {
		return fmt.Errorf("error creating plan dir: %v", err)
	}

	path := filepath.Join(dir, "settings.json")

	err = os.WriteFile(path, bytes, 0644)

	if err != nil {
		return fmt.Errorf("error writing current plan settings: %v", err)
	}

	CurrentBranch = branch

	return nil
}

func GetCurrentBranchNamesByPlanId(planIds []string) (map[string]string, error) {
	if fs.HomePlandexDir == "" {
		return nil, fmt.Errorf("HomePlandexDir not set")
	}

	if CurrentProjectId == "" || HomeCurrentPlanPath == "" {
		return nil, fmt.Errorf("no current project")
	}

	var mu sync.Mutex
	branches := make(map[string]string)
	errCh := make(chan error, len(planIds))
	for _, planId := range planIds {
		go func(planId string) {
			branch, err := getPlanCurrentBranch(planId)
			if err != nil {
				errCh <- fmt.Errorf("error getting plan current branch: %v", err)
			} else {
				mu.Lock()
				defer mu.Unlock()
				branches[planId] = branch
				errCh <- nil
			}
		}(planId)
	}

	for i := 0; i < len(planIds); i++ {
		err := <-errCh
		if err != nil {
			return nil, err
		}
	}

	return branches, nil
}

func getPlanCurrentBranch(planId string) (string, error) {
	if fs.HomePlandexDir == "" {
		return "", fmt.Errorf("HomePlandexDir not set")
	}

	if CurrentProjectId == "" || HomeCurrentPlanPath == "" {
		return "", fmt.Errorf("no current project")
	}

	path := filepath.Join(fs.HomePlandexDir, CurrentProjectId, planId, "settings.json")

	// Check if settings.json exists
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		return "main", nil
	} else if err != nil {
		return "", fmt.Errorf("error checking if settings.json exists: %v", err)
	}

	bytes, err := os.ReadFile(path)

	if err != nil {
		return "", fmt.Errorf("error reading plan settings: %v", err)
	}

	var settings types.PlanSettings
	err = json.Unmarshal(bytes, &settings)

	if err != nil {
		return "", fmt.Errorf("error unmarshalling plan settings: %v", err)
	}

	return settings.Branch, nil
}
