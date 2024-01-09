package shared

import (
	"crypto/rand"
	"time"
)

const TsFormat = "2006-01-02T15:04:05.999Z"

func StringTs() string {
	return time.Now().UTC().Format(TsFormat)
}

var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func GetRandomAlphanumeric(n int) ([]byte, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	for i, b := range bytes {
		bytes[i] = letters[int(b)%len(letters)]
	}
	return bytes, nil
}
