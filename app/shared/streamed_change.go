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

type StreamedChangeWithLineNumsUpdated struct {
	Summary                    string                `json:"summary"`
	Reasoning                  string                `json:"reasoning"`
	Section                    string                `json:"section"`
	NewReasoning               string                `json:"newReasoning"`
	OrderReasoning             string                `json:"orderReasoning"`
	StructureReasoning         string                `json:"structureReasoning"`
	ClosingSyntaxReasoning     string                `json:"closingSyntaxReasoning"`
	InsertBefore               *InsertBefore         `json:"insertBefore"`
	InsertAfter                *InsertAfter          `json:"insertAfter"`
	HasChange                  bool                  `json:"hasChange"`
	Old                        StreamedChangeSection `json:"old"`
	StartLineIncludedReasoning string                `json:"startLineIncludedReasoning"`
	StartLineIncluded          bool                  `json:"startLineIncluded"`
	EndLineIncludedReasoning   string                `json:"endLineIncludedReasoning"`
	EndLineIncluded            bool                  `json:"endLineIncluded"`
	New                        StreamedChangeSection `json:"new"`
}

type InsertBefore struct {
	ShouldInsertBefore bool   `json:"shouldInsertBefore"`
	Line               string `json:"line"`
}

type InsertAfter struct {
	ShouldInsertAfter bool   `json:"shouldInsertAfter"`
	Line              string `json:"line"`
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

func (streamedChangeSection StreamedChangeSection) GetLines() (int, int, error) {
	return streamedChangeSection.GetLinesWithPrefix("pdx-")
}

func (streamedChangeSection StreamedChangeSection) GetLinesWithPrefix(prefix string) (int, int, error) {
	var startLine, endLine int
	var err error

	if streamedChangeSection.EntireFile {
		return 1, -1, nil
	}

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
