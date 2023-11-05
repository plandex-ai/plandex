package lib

import (
	"regexp"
	"strconv"
	"time"
)

func EnsureMinDuration(start time.Time, minDuration time.Duration) {
	elapsed := time.Since(start)
	if elapsed < minDuration {
		time.Sleep(minDuration - elapsed)
	}
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
