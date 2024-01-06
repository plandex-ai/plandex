package db

import "fmt"

func ValidateProjectAccess(projectId, userId, orgId string) (bool, error) {

	var count int
	err := Conn.QueryRow("SELECT COUNT(*) FROM users_projects WHERE project_id = $1 AND user_id = $2 AND org_id = $3", projectId, userId, orgId).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("error validating project membership: %v", err)
	}

	return count > 0, nil

}
