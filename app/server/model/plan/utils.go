package plan

import (
	"plandex-server/types"
	"strings"
)

func StripBackticksWrapper(s string) string {
	check := strings.TrimSpace(s)
	split := strings.Split(check, "\n")

	if len(split) > 2 {
		firstLine := strings.TrimSpace(split[0])
		secondLine := strings.TrimSpace(split[1])
		lastLine := strings.TrimSpace(split[len(split)-1])
		if types.LineMaybeHasFilePath(firstLine) && strings.HasPrefix(secondLine, "```") {
			if lastLine == "```" {
				return strings.Join(split[1:len(split)-1], "\n")
			}
		} else if strings.HasPrefix(firstLine, "```") && lastLine == "```" {
			return strings.Join(split[1:len(split)-1], "\n")
		}
	}

	return s
}
