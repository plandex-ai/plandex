package shared

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
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

type LineNumberedTextType string

func AddLineNums(s string) LineNumberedTextType {
	return LineNumberedTextType(AddLineNumsWithPrefix(s, "pdx-"))
}

func AddLineNumsWithPrefix(s, prefix string) LineNumberedTextType {
	var res string
	for i, line := range strings.Split(s, "\n") {
		res += fmt.Sprintf("%s%d: %s\n", prefix, i+1, line)
	}
	return LineNumberedTextType(res)
}

func RemoveLineNums(s LineNumberedTextType) string {
	return RemoveLineNumsWithPrefix(s, "pdx-")
}

func RemoveLineNumsWithPrefix(s LineNumberedTextType, prefix string) string {
	return regexp.MustCompile(fmt.Sprintf(`(?m)^%s\d+: `, prefix)).ReplaceAllString(string(s), "")
}

// indexRunes searches for the slice of runes `needle` in the slice of runes `haystack`
// and returns the index of the first rune of `needle` in `haystack`, or -1 if `needle` is not present.
func IndexRunes(haystack []rune, needle []rune) int {
	if len(needle) == 0 {
		return 0
	}
	if len(haystack) == 0 {
		return -1
	}

	// Search for the needle
	for i := 0; i <= len(haystack)-len(needle); i++ {
		found := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}

	return -1
}

func ReplaceReverse(s, old, new string, n int) string {
	// If n is negative, there is no limit to the number of replacements
	if n == 0 {
		return s
	}

	if n < 0 {
		return strings.Replace(s, old, new, -1)
	}

	// If n is positive, replace the last n occurrences of old with new
	var res string
	for i := 0; i < n; i++ {
		idx := strings.LastIndex(s, old)
		if idx == -1 {
			break
		}
		res = s[:idx] + new + s[idx+len(old):]
		s = res
	}
	return res
}

func NormalizeEOL(data []byte) []byte {
	if !looksTextish(data) {
		return data
	}

	// CRLF -> LF
	n := bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\n'})

	// treat stray CR as newline as well
	n = bytes.ReplaceAll(n, []byte{'\r'}, []byte{'\n'})
	return n
}

// looksTextish checks some very cheap heuristics:
//  1. no NUL bytes      → probably not binary
//  2. valid UTF-8       → BOMs are OK
//  3. printable ratio   → ≥ 90 % of runes are >= 0x20 or common whitespace
func looksTextish(b []byte) bool {
	if bytes.IndexByte(b, 0x00) != -1 { // 1
		return false
	}
	if !utf8.Valid(b) { // 2
		return false
	}

	printable := 0
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		b = b[size:]
		switch {
		case r == '\n', r == '\r', r == '\t':
			printable++
		case r >= 0x20 && r != 0x7f:
			printable++
		}
	}
	return float64(printable)/float64(len(b)) > 0.90 // 3
}
