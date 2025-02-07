package db

import (
	"fmt"

	shared "plandex-shared"

	"github.com/jmoiron/sqlx"
)

func GetPlanConfig(planId string) (*shared.PlanConfig, error) {
	query := "SELECT plan_config FROM plans WHERE id = $1"

	var config shared.PlanConfig
	err := Conn.Get(&config, query, planId)

	if err != nil {
		return nil, fmt.Errorf("error getting plan config: %v", err)
	}

	return &config, nil
}

func StorePlanConfig(planId string, config *shared.PlanConfig) error {
	query := `
		UPDATE plans 
		SET plan_config = $1
		WHERE id = $2
	`

	_, err := Conn.Exec(query, config, planId)

	if err != nil {
		return fmt.Errorf("error storing plan config: %v", err)
	}

	return nil
}

func GetDefaultPlanConfig(userId string) (*shared.PlanConfig, error) {
	query := "SELECT default_plan_config FROM users WHERE id = $1"

	var config shared.PlanConfig
	err := Conn.Get(&config, query, userId)

	if err != nil {
		return nil, fmt.Errorf("error getting default plan config: %v", err)
	}

	return &config, nil
}

func StoreDefaultPlanConfig(userId string, config *shared.PlanConfig, tx *sqlx.Tx) error {
	query := `
		UPDATE users SET default_plan_config = $1 WHERE id = $2
	`

	_, err := tx.Exec(query, config, userId)

	if err != nil {
		return fmt.Errorf("error storing default plan config: %v", err)
	}

	return nil
}
