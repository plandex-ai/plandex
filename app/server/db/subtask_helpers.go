package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func GetPlanSubtasks(orgId, planId string) ([]*Subtask, error) {
	planDir := getPlanDir(orgId, planId)
	subtasksPath := filepath.Join(planDir, "subtasks.json")

	bytes, err := os.ReadFile(subtasksPath)

	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("error reading subtasks: %v", err)
	}

	var subtasks []*Subtask
	err = json.Unmarshal(bytes, &subtasks)

	if err != nil {
		return nil, fmt.Errorf("error unmarshalling subtasks: %v", err)
	}

	return subtasks, nil
}

func StorePlanSubtasks(orgId, planId string, subtasks []*Subtask) error {
	planDir := getPlanDir(orgId, planId)

	bytes, err := json.Marshal(subtasks)

	if err != nil {
		return fmt.Errorf("error marshalling subtasks: %v", err)
	}

	err = os.WriteFile(filepath.Join(planDir, "subtasks.json"), bytes, os.ModePerm)

	if err != nil {
		return fmt.Errorf("error writing subtasks: %v", err)
	}

	return nil
}
