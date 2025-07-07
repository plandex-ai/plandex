package db

import (
	"database/sql"
	"fmt"
	shared "plandex-shared"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func GetUser(userId string) (*User, error) {
	var user User
	err := Conn.Get(&user, "SELECT * FROM users WHERE id = $1", userId)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

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

func GetOrgUserConfig(userId, orgId string) (*shared.OrgUserConfig, error) {
	var orgUserConfig shared.OrgUserConfig
	err := Conn.Get(&orgUserConfig, "SELECT config FROM orgs_users WHERE user_id = $1 AND org_id = $2", userId, orgId)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("error getting org user config: %v", err)
	}

	return &orgUserConfig, nil
}

func UpdateOrgUserConfig(userId, orgId string, config *shared.OrgUserConfig) error {
	_, err := Conn.Exec("UPDATE orgs_users SET config = $1 WHERE user_id = $2 AND org_id = $3", config, userId, orgId)

	if err != nil {
		return fmt.Errorf("error updating org user config: %v", err)
	}

	return nil
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

func CreateUser(name, email string, tx *sqlx.Tx) (*User, error) {
	emailSplit := strings.Split(email, "@")
	if len(emailSplit) != 2 {
		return nil, fmt.Errorf("invalid email: %v", email)
	}
	domain := emailSplit[1]

	user := User{
		Name:   name,
		Email:  email,
		Domain: domain,
	}

	err := tx.QueryRow("INSERT INTO users (name, email, domain) VALUES ($1, $2, $3) RETURNING id", user.Name, user.Email, user.Domain).Scan(&user.Id)

	if err != nil {
		if IsNonUniqueErr(err) {
			return nil, fmt.Errorf("user already exists for email: %v", email)
		}
		return nil, fmt.Errorf("error creating user: %v", err)
	}

	return &user, nil
}

func NumUsersWithRole(orgId, roleId string) (int, error) {
	var count int
	err := Conn.Get(&count, "SELECT COUNT(*) FROM orgs_users WHERE org_id = $1 AND org_role_id = $2", orgId, roleId)

	if err != nil {
		return 0, fmt.Errorf("error counting users with role: %v", err)
	}

	return count, nil
}
