package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const tokenExpirationDays = 90 // (trial tokens don't expire)

func CreateAuthToken(userId string, isTrial bool, tx *sql.Tx) (token, id string, err error) {
	uid := uuid.New()
	bytes := uid[:]
	hashBytes := sha256.Sum256(bytes)
	hash := hex.EncodeToString(hashBytes[:])

	err = tx.QueryRow("INSERT INTO auth_tokens (user_id, token_hash, is_trial) VALUES ($1, $2, $3) RETURNING id", userId, hash, isTrial).Scan(&id)

	if err != nil {
		return "", "", fmt.Errorf("error creating auth token: %v", err)
	}

	return uid.String(), id, nil
}

func ValidateAuthToken(token string) (*AuthToken, error) {
	uid, err := uuid.Parse(token)

	if err != nil {
		return nil, errors.New("invalid token")
	}

	bytes := uid[:]
	hashBytes := sha256.Sum256(bytes)
	tokenHash := hex.EncodeToString(hashBytes[:])

	var authToken AuthToken
	// trial tokens don't expire
	err = Conn.Get(&authToken, "SELECT * FROM auth_tokens WHERE token_hash = $1 AND (created_at > $2 OR is_trial = TRUE) AND deleted_at IS NULL", tokenHash, time.Now().AddDate(0, 0, -tokenExpirationDays))

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("invalid token")
		}

		return nil, fmt.Errorf("error validating token: %v", err)
	}

	return &authToken, nil
}

func CreateEmailVerification(email string, userId, pinHash string) error {
	var err error
	if userId == "" {
		_, err = Conn.Exec("INSERT INTO email_verifications (email, pin_hash) VALUES ($1, $2)", email, pinHash)
	} else {
		_, err = Conn.Exec("INSERT INTO email_verifications (email, pin_hash, user_id) VALUES ($1, $2, $3)", email, pinHash, userId)
	}

	if err != nil {
		return fmt.Errorf("error creating email verification: %v", err)
	}

	return nil
}

// email verifications expire in 5 minutes
const emailVerificationExpirationMinutes = 5

func ValidateEmailVerification(email, pin string) (id string, err error) {
	pinHashBytes := sha256.Sum256([]byte(pin))
	pinHash := hex.EncodeToString(pinHashBytes[:])

	var authTokenId *string

	query := `SELECT id, auth_token_id 
              FROM email_verifications
              WHERE pin_hash = $1 
							AND email = $2
              AND created_at > $3`

	err = Conn.QueryRow(query, pinHash, email, time.Now().Add(-emailVerificationExpirationMinutes*time.Minute)).Scan(&id, &authTokenId)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("invalid or expired pin")
		}
		return "", fmt.Errorf("error validating email verification: %v", err)
	}

	if authTokenId != nil {
		return "", errors.New("pin already verified")
	}

	return id, nil
}

func GetUserPermissions(userId, orgId string) ([]string, error) {
	var permissions []string

	query := `
    SELECT p.name, p.resource_id 
    FROM permissions p
    JOIN org_roles_permissions orp ON p.id = orp.permission_id
    JOIN orgs_users ou ON orp.org_role_id = ou.org_role_id
    WHERE ou.user_id = $1 AND ou.org_id = $2
    `

	rows, err := Conn.Query(query, userId, orgId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var permission string
		var resourceId sql.NullString
		if err := rows.Scan(&permission, &resourceId); err != nil {
			return nil, err
		}

		toAdd := permission
		if resourceId.Valid {
			toAdd = toAdd + "|" + resourceId.String
		}

		permissions = append(permissions, toAdd)
	}

	// Check for errors from iterating over rows.
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}
