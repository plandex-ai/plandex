package db

import (
	"fmt"
	"time"
)

func StorePlanBuild(build *PlanBuild) error {
	query := `INSERT INTO plan_builds (org_id, plan_id, convo_message_id, file_path) VALUES (:org_id, :plan_id, :convo_message_id, :file_path) RETURNING id, created_at, updated_at`

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
	_, err := Conn.Exec("UPDATE plan_builds SET error = $1 WHERE id = $2", build.Error, build.Id)

	if err != nil {
		return fmt.Errorf("error setting build error: %v", err)
	}

	return nil
}
