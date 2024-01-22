package lib

import (
	"fmt"
	"os"
	"plandex/fs"
)

func SetCurrentPlan(id string) error {
	if fs.HomePlandexDir == "" {
		return fmt.Errorf("HomePlandexDir not set")
	}

	if CurrentProjectId == "" || HomeCurrentPlanPath == "" {
		return fmt.Errorf("no current project")
	}

	err := os.WriteFile(HomeCurrentPlanPath, []byte(fmt.Sprintf(`{"id": "%s"}`, id)), 0644)
	if err != nil {
		return err
	}

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

	return nil
}
