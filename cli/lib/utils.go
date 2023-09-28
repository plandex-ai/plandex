package lib

import (
	"path/filepath"
	"strings"
	"time"
)

func StringTs() string {
	return time.Now().Format("2006-01-02T15:04:05.999Z")
}

func EnsureMinDuration(start time.Time, minDuration time.Duration) {
	elapsed := time.Since(start)
	if elapsed < minDuration {
		time.Sleep(minDuration - elapsed)
	}
}

func GetFileNameWithoutExt(path string) string {
	name := path[:len(path)-len(filepath.Ext(path))]

	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, "'", "")
	name = strings.ReplaceAll(name, "`", "")
	name = strings.ReplaceAll(name, "\"", "")

	return name
}
