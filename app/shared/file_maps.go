package shared

import (
	"fmt"
	"sort"
	"strings"
)

func (m FileMapBodies) CombinedMap() string {
	var combinedMap strings.Builder
	paths := make([]string, 0, len(m))
	for path := range m {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		body := m[path]
		body = strings.TrimSpace(body)
		fileHeading := MapFileHeading(path)
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
	heading := MapFileHeading(path)
	body := m[path]
	body = strings.TrimSpace(body)
	combined := heading + body + "\n"
	return GetNumTokensEstimate(combined)
}

func MapFileHeading(path string) string {
	return fmt.Sprintf("\n### %s\n", path)
}
