package parse

import (
	"strings"
)

type FileMove struct {
	Source      string
	Destination string
}

// ParseMoveFiles parses the "### Move Files" section and returns a list of source/destination pairs
func ParseMoveFiles(content string) []FileMove {
	split := strings.Split(content, "### Move Files")
	if len(split) < 2 {
		return nil
	}

	var moves []FileMove
	lines := strings.Split(split[1], "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "-") {
			continue
		}

		// Remove the leading dash and trim
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimSpace(line)

		// Split on arrow, requiring backticks
		parts := strings.Split(line, "→")
		if len(parts) != 2 {
			continue
		}

		src := strings.TrimSpace(parts[0])
		dst := strings.TrimSpace(parts[1])

		// Remove backticks
		src = strings.Trim(src, "`")
		dst = strings.Trim(dst, "`")

		if src != "" && dst != "" {
			moves = append(moves, FileMove{
				Source:      src,
				Destination: dst,
			})
		}
	}

	return moves
}

// ParseRemoveFiles parses the "### Remove Files" section and returns a list of files to remove
func ParseRemoveFiles(content string) []string {
	split := strings.Split(content, "### Remove Files")
	if len(split) < 2 {
		return nil
	}

	var files []string
	lines := strings.Split(split[1], "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "-") {
			continue
		}

		// Remove the leading dash and trim
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimSpace(line)

		// Check for proper backtick format
		if !strings.HasPrefix(line, "`") || !strings.HasSuffix(line, "`") {
			continue
		}

		// Remove backticks
		line = strings.Trim(line, "`")

		if line != "" {
			files = append(files, line)
		}
	}

	return files
}

// ParseResetChanges parses the "### Reset Changes" section and returns a list of files to reset
func ParseResetChanges(content string) []string {
	split := strings.Split(content, "### Reset Changes")
	if len(split) < 2 {
		return nil
	}

	var files []string
	lines := strings.Split(split[1], "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "-") {
			continue
		}

		// Remove the leading dash and trim
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimSpace(line)

		// Check for proper backtick format
		if !strings.HasPrefix(line, "`") || !strings.HasSuffix(line, "`") {
			continue
		}

		// Remove backticks
		line = strings.Trim(line, "`")

		if line != "" {
			files = append(files, line)
		}
	}

	return files
}
