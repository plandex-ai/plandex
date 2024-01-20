package db

import (
	"fmt"
	"time"
)

func StorePlanBuild(build *PlanBuild) error {
	query := `INSERT INTO plan_builds (org_id, plan_id, convo_message_id) VALUES (:org_id, :plan_id, :convo_message_id) RETURNING id, created_at, updated_at`

	row, err := Conn.NamedQuery(query, build)

	if err != nil {
		return fmt.Errorf("error storing plan build: %v", err)
	}

	defer row.Close()

	if row.Next() {
		var createdAt, updatedAt time.Time
		var id string
		if err := row.Scan(&id, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("error storing plan build: %v", err)
		}

		build.Id = id
		build.CreatedAt = createdAt
		build.UpdatedAt = updatedAt
	}

	return nil
}

func SetBuildError(build *PlanBuild) error {
	_, err := Conn.Exec("UPDATE plan_builds SET error = $1, error_path = $2 WHERE id = $3", build.Error, build.ErrorPath, build.Id)

	if err != nil {
		return fmt.Errorf("error setting build error: %v", err)
	}

	return nil
}
