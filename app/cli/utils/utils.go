package utils

import (
	"time"
)

func EnsureMinDuration(start time.Time, minDuration time.Duration) {
	elapsed := time.Since(start)
	if elapsed < minDuration {
		time.Sleep(minDuration - elapsed)
	}
}

func EnsureValidPath(path string) string {
	// Remove trailing slash if it's not the root path
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	return path
}
