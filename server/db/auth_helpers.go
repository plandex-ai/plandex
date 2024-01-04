package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

func GenAuthToken(userId string, tx *sql.Tx) (string, error) {
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
