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

const tokenExpirationDays = 90

func CreateAuthToken(userId string, tx *sql.Tx) (token, id string, err error) {
	uid := uuid.New()
	bytes := uid[:]
	hashBytes := sha256.Sum256(bytes)
	hash := hex.EncodeToString(hashBytes[:])

	err = tx.QueryRow("INSERT INTO auth_tokens (user_id, token_hash) VALUES ($1, $2) RETURNING id", userId, hash).Scan(&id)

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
	err = Conn.Get(&authToken, "SELECT * FROM auth_tokens WHERE token_hash = $1 AND created_at > $2 AND deleted_at IS NULL", tokenHash, time.Now().AddDate(0, 0, -tokenExpirationDays))

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("invalid token")
		}

		return nil, fmt.Errorf("error validating token: %v", err)
	}

	return &authToken, nil
}

func CreateEmailVerification(email string, userId, pinHash string) error {
	_, err := Conn.Exec("INSERT INTO email_verifications (email, pin_hash, user_id) VALUES ($1, $2, $3)", email, pinHash, userId)

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

	var authTokenId string

	query := `SELECT id, auth_token_id 
              FROM email_verifications
              WHERE pin_hash = $1 
							AND email = $2
              AND created_at > $3`

	err = Conn.QueryRow(query, pinHash, email, time.Now().Add(-emailVerificationExpirationMinutes*time.Minute)).Scan(&id, &authTokenId)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("invalid pin")
		}
		return "", fmt.Errorf("error validating email verification: %v", err)
	}

	if authTokenId != "" {
		return "", errors.New("pin already verified")
	}

	return id, nil
}
