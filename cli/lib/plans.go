package lib

import (
	"fmt"
	"os"
)

func SetCurrentPlan(id string) error {
	if HomePlandexDir == "" {
		return fmt.Errorf("HomePlandexDir not set")
	}

	if CurrentProjectId == "" || HomeCurrentPlanPath == "" {
		return fmt.Errorf("No current project")
	}

	err := os.WriteFile(HomeCurrentPlanPath, []byte(fmt.Sprintf(`{"id": "%s"}`, id)), 0644)
	if err != nil {
		return err
	}

	return nil
}

func ClearCurrentPlan() error {
	if HomePlandexDir == "" {
		return fmt.Errorf("HomePlandexDir not set")
	}

	if CurrentProjectId == "" || HomeCurrentPlanPath == "" {
		return fmt.Errorf("No current project")
	}

	err := os.Remove(HomeCurrentPlanPath)
	if err != nil {
		return err
	}

	return nil
}
