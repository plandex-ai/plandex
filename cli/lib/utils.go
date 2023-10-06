package lib

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/eiannone/keyboard"
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

func getUserInput() (rune, error) {
	if err := keyboard.Open(); err != nil {
		return 0, fmt.Errorf("failed to open keyboard: %s\n", err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	char, _, err := keyboard.GetKey()
	if err != nil {
		return 0, fmt.Errorf("failed to read keypress: %s\n", err)
	}

	return char, nil
}
