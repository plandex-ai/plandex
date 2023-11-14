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

func writePlanRes(content string) error {
	var planRes shared.PlanResult
	err := json.Unmarshal([]byte(content), &planRes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal plan file: %s", err)
	}

	path := filepath.Join(ResultsSubdir, planRes.Path, planRes.Ts+".json")
	err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create  directory: %s", err)
	}

	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		// fmt.Printf("failed to write plan file '%s': %v", path, err)
		return fmt.Errorf("failed to write plan result '%s': %v", path, err)
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
