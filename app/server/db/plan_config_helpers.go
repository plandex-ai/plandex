package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/plandex/plandex/shared"
)

func GetPlanConfig(planId string) (*shared.PlanConfig, error) {
	query := "SELECT plan_config FROM plans WHERE id = $1"

	var config shared.PlanConfig
	err := Conn.Get(&config, query, planId)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty config if not found
			return &shared.PlanConfig{}, nil
		}
		return nil, fmt.Errorf("error getting plan config: %v", err)
	}

	return &config, nil
}

func StorePlanConfig(planId string, config *shared.PlanConfig) error {
	query := `
		UPDATE plans 
		SET plan_config = $1, updated_at = $2
		WHERE id = $3
	`

	updatedAt := time.Now()
	_, err := Conn.Exec(query, config, updatedAt, planId)

	if err != nil {
		return fmt.Errorf("error storing plan config: %v", err)
	}

	return nil
}

func GetDefaultPlanConfig(userId string) (*shared.PlanConfig, error) {
	query := "SELECT plan_config FROM default_plan_config WHERE user_id = $1"

	var config shared.PlanConfig
	err := Conn.Get(&config, query, userId)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty config if not found
			return &shared.PlanConfig{}, nil
		}
		return nil, fmt.Errorf("error getting default plan config: %v", err)
	}

	return &config, nil
}

func GetDefaultPlanConfigForUpdate(userId string, tx *sqlx.Tx) (*shared.PlanConfig, error) {
	query := "SELECT plan_config FROM default_plan_config WHERE user_id = $1 FOR UPDATE"

	var config shared.PlanConfig
	err := tx.Get(&config, query, userId)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty config if not found
			return &shared.PlanConfig{}, nil
		}
		return nil, fmt.Errorf("error getting default plan config for update: %v", err)
	}

	return &config, nil
}

func StoreDefaultPlanConfig(userId string, config *shared.PlanConfig, tx *sqlx.Tx) error {
	query := `
		INSERT INTO default_plan_config (user_id, plan_config, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE
		SET plan_config = EXCLUDED.plan_config,
		    updated_at = EXCLUDED.updated_at
	`

	updatedAt := time.Now()
	_, err := tx.Exec(query, userId, config, updatedAt)

	if err != nil {
		return fmt.Errorf("error storing default plan config: %v", err)
	}

	return nil
}
