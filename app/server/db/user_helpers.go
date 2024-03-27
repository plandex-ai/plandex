package db

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

func GetUser(userId string) (*User, error) {
	var user User
	err := Conn.Get(&user, "SELECT * FROM users WHERE id = $1", userId)

	if err != nil {
		return nil, fmt.Errorf("error getting user: %v", err)
	}

	return &user, nil
}

func GetUserByEmail(email string) (*User, error) {
	var user User
	err := Conn.Get(&user, "SELECT * FROM users WHERE email = $1", email)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("error getting user: %v", err)
	}

	return &user, nil
}

func GetUsersForDomain(domain string) ([]*User, error) {
	var users []*User
	err := Conn.Select(&users, "SELECT * FROM users WHERE domain = $1", domain)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("error getting users for domain: %v", err)
	}

	return users, nil
}

func GetOrgUser(userId, orgId string) (*OrgUser, error) {
	var orgUser OrgUser
	err := Conn.Get(&orgUser, "SELECT * FROM orgs_users WHERE user_id = $1 AND org_id = $2", userId, orgId)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("error getting org user: %v", err)
	}

	return &orgUser, nil
}

func ListOrgUsers(orgId string) ([]*OrgUser, error) {
	var orgUsers []*OrgUser
	err := Conn.Select(&orgUsers, "SELECT * FROM orgs_users WHERE org_id = $1", orgId)

	if err != nil {
		return nil, fmt.Errorf("error listing org users: %v", err)
	}

	return orgUsers, nil
}

func ListUsers(orgId string) ([]*User, error) {
	var users []*User

	orgUsers, err := ListOrgUsers(orgId)

	if err != nil {
		return nil, fmt.Errorf("error listing users: %v", err)
	}

	userIds := make([]string, len(orgUsers))
	for i, ou := range orgUsers {
		userIds[i] = ou.UserId
	}

	err = Conn.Select(&users, "SELECT * FROM users WHERE id = ANY($1)", pq.Array(userIds))

	if err != nil {
		return nil, fmt.Errorf("error listing users: %v", err)
	}

	return users, nil
}

func CreateUser(user *User, tx *sql.Tx) error {
	return tx.QueryRow("INSERT INTO users (name, email, domain, is_trial) VALUES ($1, $2, $3, $4) RETURNING id", user.Name, user.Email, user.Domain, user.IsTrial).Scan(&user.Id)
}

func NumUsersWithRole(orgId, roleId string) (int, error) {
	var count int
	err := Conn.Get(&count, "SELECT COUNT(*) FROM orgs_users WHERE org_id = $1 AND org_role_id = $2", orgId, roleId)

	if err != nil {
		return 0, fmt.Errorf("error counting users with role: %v", err)
	}

	return count, nil
}
