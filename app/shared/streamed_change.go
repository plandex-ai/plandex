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
	EntireFile      bool   `json:"entireFile"`
}

type StreamedChangeWithLineNums struct {
	Summary                    string                `json:"summary"`
	HasChange                  bool                  `json:"hasChange"`
	Old                        StreamedChangeSection `json:"old"`
	StartLineIncludedReasoning string                `json:"startLineIncludedReasoning"`
	StartLineIncluded          bool                  `json:"startLineIncluded"`
	EndLineIncludedReasoning   string                `json:"endLineIncludedReasoning"`
	EndLineIncluded            bool                  `json:"endLineIncluded"`
	New                        string                `json:"new"`
}

// type StreamedChangeFull struct {
// 	Summary string `json:"summary"`
// 	Old     string `json:"old"`
// 	New     string `json:"new"`
// }

type StreamedVerifyFunction struct {
	Reasoning string `json:"reasoning"`
	IsCorrect bool   `json:"isCorrect"`
}

func (streamedChange StreamedChangeWithLineNums) GetLines() (int, int, error) {
	var startLine, endLine int
	var err error

	if streamedChange.Old.EntireFile {
		return 1, -1, nil
	}

	if streamedChange.Old.StartLineString == "" {
		startLine = streamedChange.Old.StartLine
	} else {
		startLine, err = extractLineNumber(streamedChange.Old.StartLineString)

		if err != nil {
			log.Printf("Error extracting start line number: %v\n", err)
			return 0, 0, fmt.Errorf("error extracting start line number: %v", err)
		}
	}

	if streamedChange.Old.EndLineString == "" {
		if streamedChange.Old.EndLine > 0 {
			endLine = streamedChange.Old.EndLine
		} else {
			endLine = startLine
		}
	} else {
		endLine, err = extractLineNumber(streamedChange.Old.EndLineString)

		if err != nil {
			log.Printf("Error extracting end line number: %v\n", err)
			return 0, 0, fmt.Errorf("error extracting end line number: %v", err)
		}
	}

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

func extractLineNumber(line string) (int, error) {
	// Split the line at the first space to isolate the line number
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid line format")
	}

	// Remove the colon from the line number part
	lineNumberStr := strings.TrimSuffix(parts[0], ":")
	lineNumberStr = strings.TrimPrefix(lineNumberStr, "pdx-")
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
