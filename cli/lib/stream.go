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

	numTokens := int(GetNumTokens(chunk.Content))
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

		// fmt.Println("Wrote to file " + filePath)

		finishedByFile[chunk.FilePath] = true

		return true, nil

	} else {
		return false, nil
	}

}
