package lib

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/plandex/plandex/shared"
)

type receiveFileChunkParams struct {
	Content                 string
	JsonBuffers             map[string]string
	NumStreamedTokensByPath map[string]int
	FinishedByPath          map[string]bool
}

func receiveFileChunk(params *receiveFileChunkParams) (bool, error) {
	content := params.Content
	jsonBuffers := params.JsonBuffers
	numStreamedTokensByPath := params.NumStreamedTokensByPath
	finishedByPath := params.FinishedByPath

	var chunk shared.PlanChunk
	err := json.Unmarshal([]byte(content), &chunk)
	if err != nil {
		return false, fmt.Errorf("error parsing plan chunk: %v", err)
	}

	_, sectionNum, err := shared.SplitSectionPath(chunk.Path)
	if err != nil {
		return false, fmt.Errorf("error parsing section number: %v", err)
	}
	isSection := sectionNum >= 0

	// fmt.Println("path: " + chunk.Path)
	// fmt.Println("isSection: " + fmt.Sprintf("%t", isSection))
	// fmt.Println("sectionNum: " + fmt.Sprintf("%d", sectionNum))
	// fmt.Println("filePath: " + filePath)

	buffer, isFirstWriteToPath := jsonBuffers[chunk.Path]
	buffer += chunk.Content
	jsonBuffers[chunk.Path] = buffer

	numTokens := int(shared.GetNumTokens(chunk.Content))
	if isFirstWriteToPath {
		numStreamedTokensByPath[chunk.Path] = numTokens
	} else {
		numStreamedTokensByPath[chunk.Path] += numTokens
	}

	var streamed shared.StreamedFile
	err = json.Unmarshal([]byte(jsonBuffers[chunk.Path]), &streamed)

	if err == nil {
		var writeToPath string

		if isSection {
			// log.Printf("Section. Writing to section path '%s'\n", chunk.Path)
			writeToPath = filepath.Join(PlanSectionsDir, chunk.Path)
		} else {
			// log.Printf("Note section. Writing to file path '%s'\n", chunk.Path)
			writeToPath = filepath.Join(PlanFilesDir, chunk.Path)
		}

		// log.Printf("Writing to file or section path '%s'\n", writeToPath)
		err := os.MkdirAll(filepath.Dir(writeToPath), os.ModePerm)
		if err != nil {
			log.Printf("failed to create directory: %s\n", err)
			return false, fmt.Errorf("failed to create directory: %s\n", err)

		}
		file, err := os.OpenFile(writeToPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY,
			0644)
		if err != nil {
			log.Printf("failed to open plan file '%s': %v\n", writeToPath, err)
			return false, fmt.Errorf("failed to open plan file '%s': %v\n", writeToPath,
				err)
		}
		defer file.Close()
		_, err = file.WriteString(streamed.Content)

		if err != nil {
			log.Printf("failed to write plan file '%s': %v\n", writeToPath, err)
			return false, fmt.Errorf("failed to write plan file '%s': %v\n", writeToPath, err)
		}

		err = writeTokenCounts(&chunk, numStreamedTokensByPath)
		if err != nil {
			return false, fmt.Errorf("failed to write token counts: %s\n", err)
		}

		// fmt.Println("Wrote to file " + filePath)

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

func writeFilesFromSections(apiReq *shared.PromptRequest, finishedByPath map[string]bool) error {
	log.Println("Starting writeFilesFromSections...")

	var filePaths []string

	for path := range finishedByPath {
		filePath, sectionNum, err := shared.SplitSectionPath(path)
		if err != nil {
			return fmt.Errorf("error parsing section number: %v", err)
		}
		if sectionNum >= 0 {
			filePaths = append(filePaths, filePath)
		}
	}

	log.Printf("File paths to process: %v\n", filePaths)

	errCh := make(chan error)
	doneCh := make(chan bool)
	numWritten := 0

	for _, filePath := range filePaths {
		go func(filePath string) {
			log.Printf("Processing file: %s\n", filePath)

			origPath := filepath.Join(".", filePath)
			planFilesPath := filepath.Join(PlanFilesDir, filePath)

			// read the original file
			origBytes, err := os.ReadFile(origPath)
			if err != nil {
				log.Printf("failed to read original file '%s': %v\n", origPath, err)
				errCh <- fmt.Errorf("failed to read original file '%s': %v\n", origPath, err)
				return
			}
			origContent := string(origBytes)

			// get the full sections from the original file
			var contextPart *shared.ModelContextPart
			for _, part := range apiReq.ModelContext {
				if part.FilePath == filePath {
					contextPart = &part
					break
				}
			}
			if contextPart == nil {
				log.Printf("failed to find context part for file '%s'\n", filePath)
				errCh <- fmt.Errorf("failed to find context part for file '%s'\n", filePath)
				return
			}
			origSections := shared.GetFullSections(origContent, contextPart.SectionEnds)
			fileContent := origContent

			// list existing section files for this file path
			sectionsDir := filepath.Dir(filepath.Join(PlanSectionsDir, filePath))
			sectionFiles, err := os.ReadDir(sectionsDir)
			if err != nil {
				log.Printf("failed to read section files for '%s': %v\n", filePath, err)
				errCh <- fmt.Errorf("failed to read section files for '%s': %v\n", filePath, err)
				return
			}
			for _, sectionFile := range sectionFiles {
				name := sectionFile.Name()
				sectionPath := filepath.Join(filepath.Dir(filePath), name)

				_, sectionNum, err := shared.SplitSectionPath(sectionPath)
				if err != nil || sectionNum < 0 {
					log.Printf("failed to parse section number for '%s': %v\n", sectionPath, err)
					errCh <- fmt.Errorf("failed to parse section number for '%s': %v\n", sectionPath, err)
					return
				}

				if sectionNum >= len(origSections) {
					log.Printf("failed to find section %d in file '%s'\n", sectionNum, filePath)
					errCh <- fmt.Errorf("failed to find section %d in file '%s'\n", sectionNum, filePath)
					return
				}

				origSection := origSections[sectionNum]

				// read the section file
				sectionBytes, err := os.ReadFile(filepath.Join(sectionsDir, name))
				if err != nil {
					log.Printf("failed to read section file '%s': %v\n", sectionPath, err)
					errCh <- fmt.Errorf("failed to read section file '%s': %v\n", sectionPath, err)
					return
				}

				sectionContent := string(sectionBytes)

				// replace the section in the original file
				fileContent = strings.Replace(fileContent, origSection, sectionContent, 1)
			}

			// ensure directory exists
			err = os.MkdirAll(filepath.Dir(planFilesPath), os.ModePerm)
			if err != nil {
				log.Printf("failed to create directory: %s\n", err)
				errCh <- fmt.Errorf("failed to create directory: %s\n", err)
				return
			}

			// write the file
			err = os.WriteFile(planFilesPath, []byte(fileContent), 0644)
			if err != nil {
				log.Printf("failed to write file '%s': %v\n", planFilesPath, err)
				errCh <- fmt.Errorf("failed to write file '%s': %v\n", planFilesPath, err)
				return
			}

			// log.Printf("File written successfully: %s\n", planFilesPath)
			doneCh <- true
		}(filePath)
	}

	for numWritten < len(filePaths) {
		select {
		case err := <-errCh:
			return err
		case <-doneCh:
			numWritten++
			// log.Printf("Number of files written: %d\n", numWritten)
		}
	}

	// log.Println("Completed writeFilesFromSections.")

	return nil
}
