package lib

import (
	"strings"
)

func GetSectionEnds(text string, lines []string) []int {
	var indexes []int
	currentPosition := 0 // Variable to keep track of the current position in the text

	for _, line := range lines {
		// Finding the index of the line in the text, starting from currentPosition
		index := strings.Index(text[currentPosition:], line)

		if index != -1 {
			// Since we are slicing the text, we need to add currentPosition to get the actual index
			index += currentPosition

			// Updating the currentPosition to be after the current match
			currentPosition = index + len(line)

			// Appending the actual index of the match
			indexes = append(indexes, index)
		}
	}

	return indexes[1:] // Removing the first index since it's always 0
}

func CleanSectionJson(json []byte) []byte {
	s := string(json)

	strings.ReplaceAll(s, `\n`, "")
	strings.ReplaceAll(s, `\t`, "")

	return []byte(s)
}
