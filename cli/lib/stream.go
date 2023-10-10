package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/plandex/plandex/shared"
)

func receiveFileChunk(content string, desc *shared.PlanDescription, jsonBuffers map[string]string, numTokensByFile map[string]int, finishedByFile map[string]bool) (bool, error) {
	var chunk shared.PlanChunk
	err := json.Unmarshal([]byte(content), &chunk)
	if err != nil {
		return false, fmt.Errorf("error parsing plan chunk: %v", err)

	}

	var filePath string
	// if chunk.IsExec {
	// 	filePath = filepath.Join(PlanSubdir, "exec.sh")
	// } else {
	filePath = filepath.Join(PlanFilesDir, chunk.FilePath)
	// }

	buffer := jsonBuffers[chunk.FilePath]
	buffer += chunk.Content
	jsonBuffers[chunk.FilePath] = buffer

	numTokens := int(shared.GetNumTokens(chunk.Content))
	numTokensByFile[chunk.FilePath] += numTokens

	var streamed shared.StreamedFile
	err = json.Unmarshal([]byte(jsonBuffers[chunk.FilePath]), &streamed)

	if err == nil {
		// full file content received
		err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		if err != nil {
			return false, fmt.Errorf("failed to create directory: %s\n", err)

		}
		file, err := os.OpenFile(filePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY,
			0644)
		if err != nil {
			return false, fmt.Errorf("failed to open plan file '%s': %v\n", filePath,
				err)
		}
		defer file.Close()
		_, err = file.WriteString(streamed.Content)

		if err != nil {
			return false, fmt.Errorf("failed to write plan file '%s': %v\n", filePath, err)
		}

		currentPlanTokensByFilePath := make(map[string]int)
		currrentPlanTokensPath := filepath.Join(CurrentPlanRootDir, "tokens.json")

		// Read the existing token counts if the file exists.
		if _, err := os.Stat(currrentPlanTokensPath); !os.IsNotExist(err) {
			fileBytes, err := os.ReadFile(currrentPlanTokensPath)
			if err != nil {
				return false, fmt.Errorf("failed to read current plan token count file: %s", err)
			}
			err = json.Unmarshal(fileBytes, &currentPlanTokensByFilePath)
			if err != nil {
				return false, fmt.Errorf("failed to parse current plan token count json: %s", err)
			}
		}

		// Update token count for this file.
		currentPlanTokensByFilePath[chunk.FilePath] = numTokensByFile[chunk.FilePath]

		// Write the updated token counts.
		tokensFileBytes, err := json.Marshal(currentPlanTokensByFilePath)
		if err != nil {
			return false, fmt.Errorf("failed to marshal current plan token count json: %s", err)
		}

		// Open the file for writing with truncation.
		tokensFile, err := os.OpenFile(currrentPlanTokensPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return false, fmt.Errorf("failed to open current plan token count file for writing: %s", err)
		}
		defer tokensFile.Close()

		_, err = tokensFile.Write(tokensFileBytes)
		if err != nil {
			return false, fmt.Errorf("failed to write current plan token count file: %s", err)
		}

		// fmt.Println("Wrote to file " + filePath)

		finishedByFile[chunk.FilePath] = true

		return true, nil

	} else {
		return false, nil
	}

}
