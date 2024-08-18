package db

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type CreateAccountResult struct {
	User  *User
	OrgId string
	Token string
}

func CreateAccount(name, email, emailVerificationId string, tx *sqlx.Tx) (*CreateAccountResult, error) {
	// create user
	user, err := CreateUser(name, email, tx)

	if err != nil {
		return nil, fmt.Errorf("error creating user: %v", err)
	}

	userId := user.Id
	domain := user.Domain

	// create auth token
	token, authTokenId, err := CreateAuthToken(userId, false, tx)

	if err != nil {
		return nil, fmt.Errorf("error creating auth token: %v", err)
	}

	// update email verification with user and auth token ids
	_, err = tx.Exec("UPDATE email_verifications SET user_id = $1, auth_token_id = $2 WHERE id = $3", userId, authTokenId, emailVerificationId)

	if err != nil {
		return nil, fmt.Errorf("error updating email verification: %v", err)
	}

	// add to org matching domain if one exists and auto add domain users is true for that org
	orgId, err := AddToOrgForDomain(domain, userId, tx)

	if err != nil {
		return nil, fmt.Errorf("error adding user to org for domain: %v", err)
	}

	return &CreateAccountResult{
		User:  user,
		OrgId: orgId,
		Token: token,
	}, nil
}
