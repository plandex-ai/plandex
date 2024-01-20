package db

import "fmt"

func GetOrgOwnerRoleId() (string, error) {
	var roleId string
	err := Conn.Get(&roleId, "SELECT id FROM org_roles WHERE name = 'owner'")

	if err != nil {
		return "", fmt.Errorf("error getting owner role id: %v", err)
	}

	return roleId, nil
}
