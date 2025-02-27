package shared

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

type StreamedChangeSection struct {
	StartLine       int    `json:"startLine"`
	EndLine         int    `json:"endLine"`
	StartLineString string `json:"startLineString"`
	EndLineString   string `json:"endLineString"`
}

type StreamedChangeWithLineNums struct {
	Old               StreamedChangeSection `json:"old"`
	StartLineIncluded bool                  `json:"startLineIncluded"`
	EndLineIncluded   bool                  `json:"endLineIncluded"`
	New               string                `json:"new"`
}

func (streamedChangeSection StreamedChangeSection) GetLines() (int, int, error) {
	return streamedChangeSection.GetLinesWithPrefix("pdx-")
}

func (streamedChangeSection StreamedChangeSection) GetLinesWithPrefix(prefix string) (int, int, error) {
	var startLine, endLine int
	var err error

	if streamedChangeSection.StartLineString == "" {
		log.Printf("StartLineString is empty\n")
		// spew.Dump(streamedChangeSection)
		startLine = streamedChangeSection.StartLine
	} else {
		startLine, err = ExtractLineNumberWithPrefix(streamedChangeSection.StartLineString, prefix)

		if err != nil {
			log.Printf("Error extracting start line number: %v\n", err)
			return 0, 0, fmt.Errorf("error extracting start line number: %v", err)
		}
	}

	if streamedChangeSection.EndLineString == "" {
		log.Printf("EndLineString is empty\n")
		// spew.Dump(streamedChangeSection)
		if streamedChangeSection.EndLine > 0 {
			endLine = streamedChangeSection.EndLine
		} else {
			endLine = startLine
		}
	} else {
		endLine, err = ExtractLineNumberWithPrefix(streamedChangeSection.EndLineString, prefix)

		if err != nil {
			log.Printf("Error extracting end line number: %v\n", err)
			return 0, 0, fmt.Errorf("error extracting end line number: %v", err)
		}
	}

	log.Printf("StartLine: %d, EndLine: %d\n", startLine, endLine)

	if startLine > endLine {
		log.Printf("Start line is greater than end line: %d > %d\n", startLine, endLine)
		return 0, 0, fmt.Errorf("start line is greater than end line: %d > %d", startLine, endLine)
	}

	if startLine < 1 {
		log.Printf("Start line is less than 1: %d\n", startLine)
		return 0, 0, fmt.Errorf("start line is less than 1: %d", startLine)
	}

	return startLine, endLine, nil
}

func ExtractLineNumber(line string) (int, error) {
	return ExtractLineNumberWithPrefix(line, "pdx-")
}

func ExtractLineNumberWithPrefix(line, prefix string) (int, error) {
	// Split the line at the first space to isolate the line number
	parts := strings.SplitN(line, " ", 2)

	// Remove the colon from the line number part
	lineNumberStr := strings.TrimSuffix(parts[0], ":")
	lineNumberStr = strings.TrimPrefix(lineNumberStr, prefix)
	if lineNumberStr == "" {
		return 0, fmt.Errorf("no line number found")
	}

	// Convert the line number part to an integer
	lineNumber, err := strconv.Atoi(lineNumberStr)
	if err != nil {
		return 0, fmt.Errorf("invalid line number: %v", err)
	}

	return lineNumber, nil
}
