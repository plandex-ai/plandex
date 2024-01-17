package db

import (
	"database/sql"
	"fmt"
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

func ListUsers(orgId string) ([]*User, error) {
	var users []*User
	err := Conn.Select(&users, "SELECT u.* FROM users u INNER JOIN orgs_users ou ON u.id = ou.user_id WHERE ou.org_id = $1", orgId)

	if err != nil {
		return nil, fmt.Errorf("error listing users: %v", err)
	}

	return users, nil
}

func CreateUser(user *User, tx *sql.Tx) error {
	return tx.QueryRow("INSERT INTO users (name, email, domain) VALUES ($1, $2, $3) RETURNING id", user.Name, user.Email, user.Domain).Scan(&user.Id)
}
