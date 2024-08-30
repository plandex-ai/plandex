package db

import (
	"fmt"
	"log"
)

var orgOwnerRoleId string
var orgMemberRoleId string

func GetOrgOwnerRoleId() (string, error) {
	if orgOwnerRoleId == "" {
		err := cacheOrgOwnerRoleId()
		if err != nil {
			return "", fmt.Errorf("error getting org owner role id: %v", err)
		}
	}

	if orgOwnerRoleId == "" {
		return "", fmt.Errorf("org owner role id is empty")
	}

	return orgOwnerRoleId, nil
}

func GetOrgMemberRoleId() (string, error) {
	if orgMemberRoleId == "" {
		err := cacheOrgMemberRoleId()
		if err != nil {
			return "", fmt.Errorf("error getting org member role id: %v", err)
		}
	}

	if orgMemberRoleId == "" {
		return "", fmt.Errorf("org member role id is empty")
	}

	return orgMemberRoleId, nil
}

func GetOrgOwners(orgId string) ([]*User, error) {
	var users []*User
	err := Conn.Select(&users, "SELECT * FROM users WHERE id IN (SELECT user_id FROM orgs_users WHERE org_id = $1 AND org_role_id = $2)", orgId, orgOwnerRoleId)

	if err != nil {
		return nil, fmt.Errorf("error getting org owners: %v", err)
	}

	return users, nil
}

func CacheOrgRoleIds() error {
	err := cacheOrgOwnerRoleId()
	if err != nil {
		return fmt.Errorf("error getting org owner role id: %v", err)

	}

	if orgOwnerRoleId == "" {
		log.Println("org owner role id is empty at startup")
	}

	err = cacheOrgMemberRoleId()
	if err != nil {
		return fmt.Errorf("error getting org member role id: %v", err)
	}

	if orgMemberRoleId == "" {
		log.Println("org member role id is empty at startup")
	}

	return nil
}

func cacheOrgOwnerRoleId() error {
	var roleId string
	err := Conn.Get(&roleId, "SELECT id FROM org_roles WHERE name = 'owner'")

	if err != nil {
		return fmt.Errorf("error getting owner role id: %v", err)
	}

	orgOwnerRoleId = roleId

	return nil
}

func cacheOrgMemberRoleId() error {
	var roleId string
	err := Conn.Get(&roleId, "SELECT id FROM org_roles WHERE name = 'member'")

	if err != nil {
		return fmt.Errorf("error getting member role id: %v", err)
	}

	orgMemberRoleId = roleId

	return nil
}
