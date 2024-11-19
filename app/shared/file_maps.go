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
		fileHeading := fmt.Sprintf("### %s\n", path)
		combinedMap.WriteString(fileHeading)
		combinedMap.WriteString(body)
		combinedMap.WriteString("\n")
	}
	return combinedMap.String()
}
