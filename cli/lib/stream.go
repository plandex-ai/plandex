package lib

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/plandex/plandex/shared"
)

type receiveFileChunkParams struct {
	Content                 string
	JsonBuffers             map[string]string
	NumStreamedTokensByPath map[string]int
	FinishedByPath          map[string]bool
}

func receiveFileToken(params *receiveFileChunkParams) (bool, error) {
	content := params.Content
	jsonBuffers := params.JsonBuffers
	numStreamedTokensByPath := params.NumStreamedTokensByPath
	finishedByPath := params.FinishedByPath

	var chunk shared.PlanChunk
	err := json.Unmarshal([]byte(content), &chunk)
	if err != nil {
		return false, fmt.Errorf("error parsing plan chunk: %v", err)
	}

	// _, sectionNum, err := shared.SplitSectionPath(chunk.Path)
	// if err != nil {
	// 	return false, fmt.Errorf("error parsing section number: %v", err)
	// }
	// isSection := sectionNum >= 0

	// fmt.Println("path: " + chunk.Path)
	// fmt.Println("isSection: " + fmt.Sprintf("%t", isSection))
	// fmt.Println("sectionNum: " + fmt.Sprintf("%d", sectionNum))
	// fmt.Println("filePath: " + filePath)

	buffer := jsonBuffers[chunk.Path]
	buffer += chunk.Content
	jsonBuffers[chunk.Path] = buffer

	numStreamedTokensByPath[chunk.Path]++

	// log.Println("Received file chunk: " + chunk.Path)
	// log.Println("Number of tokens: " + fmt.Sprintf("%d", numStreamedTokensByPath[chunk.Path]))

	var streamedType string
	var streamed shared.StreamedFile
	var replacements shared.StreamedReplacements

	err = json.Unmarshal([]byte(jsonBuffers[chunk.Path]), &replacements)
	if err == nil {
		streamedType = "replacements"
	} else {
		err = json.Unmarshal([]byte(jsonBuffers[chunk.Path]), &streamed)
		if err == nil {
			streamedType = "file"
		}
	}

	if err == nil {

		// log.Println("Parsed JSON. Streamed type: " + streamedType)

		var writeToPath string

		writeToPath = filepath.Join(PlanFilesDir, chunk.Path)

		var content string

		if streamedType == "replacements" {

			exists := false
			if _, err := os.Stat(writeToPath); !os.IsNotExist(err) {
				exists = true
			}

			if exists {
				bytes, err := os.ReadFile(writeToPath)

				if err != nil {
					err = fmt.Errorf("failed to read file '%s': %v\n", writeToPath, err)
					log.Println(err)
					return false, fmt.Errorf("failed to read file '%s': %v\n", writeToPath, err)
				}

				content = string(bytes)
			} else {
				bytes, err := os.ReadFile(filepath.Join(ProjectRoot, chunk.Path))

				if err != nil {
					err = fmt.Errorf("failed to read file '%s': %v\n", chunk.Path, err)
					log.Println(err)
					return false, fmt.Errorf("failed to read file '%s': %v\n", chunk.Path, err)
				}

				content = string(bytes)
			}

			// log.Println("Content before replacements: " + content)

			// ensure replacements are ordered by index in content (error if not present)
			sort.Slice(replacements.Replacements, func(i, j int) bool {
				iIdx := strings.Index(content, replacements.Replacements[i].Old)
				jIdx := strings.Index(content, replacements.Replacements[j].Old)
				return iIdx < jIdx
			})

			lastInsertedIdx := 0
			for _, replacement := range replacements.Replacements {
				pre := content[:lastInsertedIdx]
				sub := content[lastInsertedIdx:]
				idx := strings.Index(sub, replacement.Old)
				if idx == -1 {
					err = fmt.Errorf("failed to find replacement string '%s' in file '%s'\n", replacement.Old, chunk.Path)
					log.Println(err)
					return false, err
				}

				updated := strings.Replace(sub, replacement.Old, replacement.New, 1)

				// log.Println("Replacement: " + replacement.Old + " -> " + replacement.New)
				// log.Println("Pre: " + pre)
				// log.Println("Sub: " + sub)
				// log.Println("Idx: " + fmt.Sprintf("%d", idx))
				// log.Println("Updated: " + updated)

				content = pre + updated

				lastInsertedIdx = lastInsertedIdx + idx + len(replacement.New)
			}

			// log.Println("Content after replacements: " + content)
		} else if streamedType == "file" {
			content = streamed.Content
		} else {
			err = fmt.Errorf("unknown streamed type: %s", streamedType)
			log.Println(err)
			return false, err
		}

		err := os.MkdirAll(filepath.Dir(writeToPath), os.ModePerm)
		if err != nil {
			log.Printf("failed to create directory: %s\n", err)
			return false, fmt.Errorf("failed to create directory: %s\n", err)

		}

		err = os.WriteFile(writeToPath, []byte(content), 0644)
		if err != nil {
			log.Printf("failed to write plan file '%s': %v\n", writeToPath, err)
			return false, fmt.Errorf("failed to write plan file '%s': %v\n", writeToPath, err)
		}

		err = writeTokenCounts(&chunk, numStreamedTokensByPath)
		if err != nil {
			return false, fmt.Errorf("failed to write token counts: %s\n", err)
		}

		finishedByPath[chunk.Path] = true

		return true, nil

	} else {
		return false, nil
	}

}

func writeTokenCounts(chunk *shared.PlanChunk, numTokensByPath map[string]int) error {
	currentPlanTokensByPath := make(map[string]int)
	currrentPlanTokensPath := filepath.Join(CurrentPlanRootDir, "tokens.json")

	// Read the existing token counts if the file exists.
	if _, err := os.Stat(currrentPlanTokensPath); !os.IsNotExist(err) {
		fileBytes, err := os.ReadFile(currrentPlanTokensPath)
		if err != nil {
			return fmt.Errorf("failed to read current plan token count file: %s", err)
		}
		err = json.Unmarshal(fileBytes, &currentPlanTokensByPath)
		if err != nil {
			return fmt.Errorf("failed to parse current plan token count json: %s", err)
		}
	}

	// Update token count for this path.
	currentPlanTokensByPath[chunk.Path] = numTokensByPath[chunk.Path]

	// Write the updated token counts.
	tokensFileBytes, err := json.Marshal(currentPlanTokensByPath)
	if err != nil {
		return fmt.Errorf("failed to marshal current plan token count json: %s", err)
	}

	// Open the file for writing with truncation.
	tokensFile, err := os.OpenFile(currrentPlanTokensPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open current plan token count file for writing: %s", err)
	}
	defer tokensFile.Close()

	_, err = tokensFile.Write(tokensFileBytes)
	if err != nil {
		return fmt.Errorf("failed to write current plan token count file: %s", err)
	}

	return nil
}

func loadCurrentPlanTokensByFilePath() (map[string]int, error) {
	currentPlanTokensByPath := make(map[string]int)
	currrentPlanTokensPath := filepath.Join(CurrentPlanRootDir, "tokens.json")
	if _, err := os.Stat(currrentPlanTokensPath); os.IsNotExist(err) {
		// do nothing
	} else {
		fileBytes, err := os.ReadFile(currrentPlanTokensPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open current plan token count file: %s\n", err)
		}
		err = json.Unmarshal(fileBytes, &currentPlanTokensByPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse current plan token count json: %s\n", err)
		}
	}

	currentPlanTokensByFilePath := make(map[string]int)

	for path, numTokens := range currentPlanTokensByPath {
		filePath, _, err := shared.SplitSectionPath(path)
		if err != nil {
			return nil, fmt.Errorf("error parsing section number: %v", err)
		}
		currentPlanTokensByFilePath[filePath] += numTokens
	}

	return currentPlanTokensByFilePath, nil
}
