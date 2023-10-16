package shared

import (
	"regexp"
	"strconv"
	"strings"
)

func GetFullSections(text string, sectionEnds []int) []string {
	var sections []string
	startIndex := 0

	for _, endIndex := range sectionEnds {
		// If endIndex goes beyond text length, clip it to text length
		if endIndex > len(text) {
			endIndex = len(text)
		}

		// Extract section and append to sections
		section := text[startIndex:endIndex]
		sections = append(sections, section)

		// Update startIndex for the next iteration
		startIndex = endIndex
	}

	// Adding the last section if there's any remaining text
	if startIndex < len(text) {
		sections = append(sections, text[startIndex:])
	}

	return sections
}

var sectionRegex = regexp.MustCompile(`-\d+$`)

func SplitSectionPath(path string) (string, int, error) {
	var sectionNum int = -1
	var err error
	sectionNumStr := strings.ReplaceAll(sectionRegex.FindString(path), "-", "")

	if sectionNumStr == "" {
		return path, -1, nil
	}

	if sectionNumStr != "" {
		sectionNum, err = strconv.Atoi(sectionNumStr)
		if err != nil {
			return "", -1, err
		}
	}

	return path[:len(path)-(len(sectionNumStr)+1)], sectionNum, nil
}
