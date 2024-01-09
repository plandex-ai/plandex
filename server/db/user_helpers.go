package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/plandex/plandex/shared"
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

func GetOrgsForUser(userId string) ([]*Org, error) {
	var orgs []*Org
	err := Conn.Select(&orgs, "SELECT o.* FROM orgs o JOIN orgs_users ou ON o.id = ou.org_id WHERE ou.user_id = $1", userId)

	if err != nil {
		return nil, fmt.Errorf("error getting orgs for user: %v", err)
	}

	return orgs, nil
}

func ValidateOrgMembership(userId string, orgId string) (bool, error) {
	var count int
	err := Conn.QueryRow("SELECT COUNT(*) FROM orgs_users WHERE user_id = $1 AND org_id = $2", userId, orgId).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("error validating org membership: %v", err)
	}

	return count > 0, nil
}

func CreateOrg(req *shared.CreateOrgRequest, userId string) (*Org, error) {
	org := &Org{
		Name:               req.Name,
		Domain:             req.Domain,
		AutoAddDomainUsers: req.AutoAddDomainUsers,
		OwnerId:            userId,
	}

	// start a transaction
	tx, err := Conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %v", err)
	}

	// Ensure that rollback is attempted in case of failure
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("transaction rollback error: %v\n", rbErr)
			} else {
				log.Println("transaction rolled back")
			}
		}
	}()

	err = tx.QueryRow("INSERT INTO orgs (name, domain, auto_add_domain_users, owner_id) VALUES ($1, $2, $3, $4) RETURNING id", req.Name, req.Domain, req.AutoAddDomainUsers, userId).Scan(&org.Id)

	if err != nil {
		if IsNonUniqueErr(err) {
			// Handle the uniqueness constraint violation
			return nil, fmt.Errorf("an org with domain %s already exists", req.Domain)

		}

		return nil, fmt.Errorf("error creating org: %v", err)
	}

	_, err = tx.Exec("INSERT INTO orgs_users (org_id, user_id) VALUES ($1, $2)", org.Id, userId)

	if err != nil {
		return nil, fmt.Errorf("error adding org membership: %v", err)
	}

	err = tx.Commit()

	if err != nil {
		return nil, fmt.Errorf("error committing transaction: %v", err)
	}

	return org, nil
}
