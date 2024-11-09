package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex/auth"
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

	settings := types.PlanSettingsByAccount{
		auth.Current.UserId: &types.PlanSettings{
			Branch: branch,
		},
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

	path := filepath.Join(dir, "settings-v2.json")

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

	v2Path := filepath.Join(fs.HomePlandexDir, CurrentProjectId, planId, "settings-v2.json")

	var settings *types.PlanSettings

	// check if settings-v2.json exists
	_, err := os.Stat(v2Path)
	if err == nil {
		// read settings-v2.json
		var settingsByAccount types.PlanSettingsByAccount
		bytes, err := os.ReadFile(v2Path)
		if err != nil {
			return "", fmt.Errorf("error reading settings-v2.json: %v", err)
		}
		err = json.Unmarshal(bytes, &settingsByAccount)
		if err != nil {
			return "", fmt.Errorf("error unmarshalling settings-v2.json: %v", err)
		}

		settings = settingsByAccount[auth.Current.UserId]
	} else if os.IsNotExist(err) {
		return "main", nil
	} else {
		return "", fmt.Errorf("error checking if settings-v2.json exists: %v", err)
	}

	if settings == nil {
		return "main", nil
	}

	return settings.Branch, nil
}
