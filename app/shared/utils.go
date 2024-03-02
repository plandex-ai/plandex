package shared

import (
	"crypto/rand"
	"regexp"
	"strings"
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

func Dasherize(s string) string {
	regex := regexp.MustCompile("([A-Z][a-z0-9]*)")
	indexes := regex.FindAllStringIndex(s, -1)
	if indexes == nil {
		return strings.ToLower(s)
	}

	var parts []string
	lastStart := 0
	for _, loc := range indexes {
		if lastStart != loc[0] {
			parts = append(parts, s[lastStart:loc[0]])
		}
		parts = append(parts, s[loc[0]:loc[1]])
		lastStart = loc[1]
	}
	if lastStart < len(s) {
		parts = append(parts, s[lastStart:])
	}

	s = strings.ToLower(strings.Join(parts, "-"))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	return s
}

func Compact(s string) string {
	return strings.ReplaceAll(Dasherize(s), "-", "")
}

func Capitalize(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
