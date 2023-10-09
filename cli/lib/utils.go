package lib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
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

func cleanJsonFragment(s string) string {
	// replace escaped quotes with unescaped quotes
	s = regexp.MustCompile(`\\\"`).ReplaceAllString(s, `"`)

	// replace escaped backslashes with unescaped backslashes
	s = regexp.MustCompile(`\\\\`).ReplaceAllString(s, `\`)

	// replace escaped newlines with unescaped newlines
	s = regexp.MustCompile(`\\n`).ReplaceAllString(s, "\n")

	// replace escaped tabs with unescaped tabs
	s = regexp.MustCompile(`\\t`).ReplaceAllString(s, "\t")

	// replace escaped carriage returns with unescaped carriage returns
	s = regexp.MustCompile(`\\r`).ReplaceAllString(s, "\r")

	// replace escaped backspaces with unescaped backspaces
	s = regexp.MustCompile(`\\b`).ReplaceAllString(s, "\b")

	// replace escaped form feeds with unescaped form feeds
	s = regexp.MustCompile(`\\f`).ReplaceAllString(s, "\f")

	// replace escaped forward slashes with unescaped forward slashes
	s = regexp.MustCompile(`\\\/`).ReplaceAllString(s, "/")

	// replace escaped unicode characters with unescaped unicode characters
	s = regexp.MustCompile(`\\u([0-9a-fA-F]{4})`).ReplaceAllStringFunc(s, func(match string) string {
		unicode, _ := strconv.ParseInt(match[2:], 16, 32)
		return string(rune(unicode))
	})

	return s
}
