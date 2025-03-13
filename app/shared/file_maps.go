package shared

import (
	"fmt"
	"sort"
	"strings"
)

func (m FileMapBodies) CombinedMap(tokensByPath map[string]int) string {
	var combinedMap strings.Builder
	paths := make([]string, 0, len(m))
	for path := range m {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		body := m[path]
		body = strings.TrimSpace(body)
		fileHeading := MapFileHeading(path, tokensByPath[path])
		combinedMap.WriteString(fileHeading)
		if body == "" {
			combinedMap.WriteString("[NO MAP]")
		} else {
			combinedMap.WriteString(body)
		}
		combinedMap.WriteString("\n")
	}
	return combinedMap.String()
}

func (m FileMapBodies) TokenEstimateForPath(path string) int {
	body := m[path]
	body = strings.TrimSpace(body)
	bodyTokens := GetNumTokensEstimate(body + "\n")

	heading := MapFileHeading(path, bodyTokens)
	headingTokens := GetNumTokensEstimate(heading)

	return bodyTokens + headingTokens
}

func MapFileHeading(path string, tokens int) string {
	return fmt.Sprintf("\n### %s (%d 🪙)\n\n", path, tokens)
}
