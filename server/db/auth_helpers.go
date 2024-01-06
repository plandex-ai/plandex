package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const tokenExpirationDays = 90

func CreateAuthToken(userId string, tx *sql.Tx) (string, error) {
	uid := uuid.New()
	bytes := uid[:]
	hashBytes := sha256.Sum256(bytes)
	hash := hex.EncodeToString(hashBytes[:])

	_, err := tx.Exec("INSERT INTO auth_tokens (user_id, token_hash) VALUES ($1, $2)", userId, hash)

	if err != nil {
		return "", fmt.Errorf("error creating auth token: %v", err)
	}

	return uid.String(), nil
}

func ValidateAuthToken(token string) (userId string, err error) {
	uid, err := uuid.Parse(token)

	if err != nil {
		return "", fmt.Errorf("error parsing token: %v", err)
	}

	bytes := uid[:]
	hashBytes := sha256.Sum256(bytes)
	hash := hex.EncodeToString(hashBytes[:])

	err = Conn.QueryRow("SELECT user_id FROM auth_tokens WHERE token_hash = $1 AND created_at > $2", hash, time.Now().AddDate(0, 0, -tokenExpirationDays)).Scan(&userId)

	if err != nil {
		return "", fmt.Errorf("error validating token: %v", err)
	}

	return userId, nil
}

func ValidateOrgMembership(userId string, orgId string) (bool, error) {
	var count int
	err := Conn.QueryRow("SELECT COUNT(*) FROM orgs_users WHERE user_id = $1 AND org_id = $2", userId, orgId).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("error validating org membership: %v", err)
	}

	return count > 0, nil
}
