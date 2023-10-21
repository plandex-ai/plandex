package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
