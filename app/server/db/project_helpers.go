package db

import (
	"database/sql"
	"fmt"
)

func ProjectExists(orgId, projectId string) (bool, error) {
	var count int
	err := Conn.QueryRow("SELECT COUNT(*) FROM projects WHERE org_id = $1 AND id = $2", orgId, projectId).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("error checking if project exists: %v", err)
	}

	return count > 0, nil
}

func CreateProject(orgId, name string, tx *sql.Tx) (string, error) {
	var projectId string
	err := tx.QueryRow("INSERT INTO projects (org_id, name) VALUES ($1, $2) RETURNING id", orgId, name).Scan(&projectId)

	if err != nil {
		return "", fmt.Errorf("error creating project: %v", err)
	}

	return projectId, nil
}
