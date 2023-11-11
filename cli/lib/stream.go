package lib

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/plandex/plandex/shared"
)

func updateTokenCounts(content string, numStreamedTokensByPath map[string]int, finishedByPath map[string]bool) error {
	var planTokenCount shared.PlanTokenCount
	err := json.Unmarshal([]byte(content), &planTokenCount)
	if err != nil {
		return fmt.Errorf("error parsing plan token count update: %v", err)
	}
	numStreamedTokensByPath[planTokenCount.Path] += planTokenCount.NumTokens

	if planTokenCount.Finished {
		finishedByPath[planTokenCount.Path] = true
	}

	return nil
}

func writePlanFile(content string) error {
	var planFile shared.PlanFile
	err := json.Unmarshal([]byte(content), &planFile)
	if err != nil {
		return fmt.Errorf("failed to unmarshal plan file: %s", err)
	}

	// spew.Dump(planFile)

	writeToPath := filepath.Join(DraftFilesDir, planFile.Path)

	// fmt.Printf("Writing plan file '%s'...\n", writeToPath)

	err = os.MkdirAll(filepath.Dir(writeToPath), os.ModePerm)
	if err != nil {
		fmt.Printf("failed to create directory: %s", err)
		return fmt.Errorf("failed to create directory: %s", err)

	}

	err = os.WriteFile(writeToPath, []byte(planFile.Content), 0644)
	if err != nil {
		fmt.Printf("failed to write plan file '%s': %v", writeToPath, err)
		return fmt.Errorf("failed to write plan file '%s': %v", writeToPath, err)
	}

	return nil
}

func readUntilSeparator(reader *bufio.Reader, separator string) (string, error) {
	var result []byte
	sepBytes := []byte(separator)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return string(result), err
		}
		result = append(result, b)
		if len(result) >= len(sepBytes) && bytes.HasSuffix(result, sepBytes) {
			return string(result[:len(result)-len(separator)]), nil
		}
	}
}
