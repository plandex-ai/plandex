package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/plandex/plandex/shared"
)

func WriteInitialContextState(contextDir string) error {
	contextState := shared.ModelContextState{
		NumTokens:    0,
		ActiveTokens: 0,
		ChatFlexPct:  25,
		PlanFlexPct:  50,
	}
	contextStateFilePath := filepath.Join(contextDir, "context.json")
	contextStateFile, err := os.OpenFile(contextStateFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer contextStateFile.Close()

	contextStateFileContents, err := json.Marshal(contextState)
	if err != nil {
		return err
	}

	_, err = contextStateFile.Write(contextStateFileContents)

	return err
}

// createContextFileName constructs a filename based on the given name and sha.
func createContextFileName(name, sha string) string {
	// Extract the first 8 characters of the sha
	shaSubstring := sha[:8]
	return fmt.Sprintf("%s.%s", name, shaSubstring)
}

// writeContextPartToFile writes a single ModelContextPart to a file.
func writeContextPartToFile(part shared.ModelContextPart) error {
	metaFilename := createContextFileName(part.Name, part.Sha) + ".meta"
	metaPath := filepath.Join(ContextSubdir, metaFilename)

	bodyFilename := createContextFileName(part.Name, part.Sha) + ".body"
	bodyPath := filepath.Join(ContextSubdir, bodyFilename)
	body := []byte(part.Body)
	part.Body = ""

	// Convert the ModelContextPart to JSON
	data, err := json.MarshalIndent(part, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context part: %v", err)
	}

	// Open or create a bodyFile for writing
	bodyFile, err := os.OpenFile(bodyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s for writing: %v", bodyPath, err)
	}
	defer bodyFile.Close()

	// Write the body to the file
	if _, err = bodyFile.Write(body); err != nil {
		return fmt.Errorf("failed to write data to file %s: %v", bodyPath, err)
	}

	// Open or create a metaFile for writing
	metaFile, err := os.OpenFile(metaPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s for writing: %v", metaPath, err)
	}
	defer metaFile.Close()

	// Write the JSON data to the file
	if _, err = metaFile.Write(data); err != nil {
		return fmt.Errorf("failed to write data to file %s: %v", metaPath, err)
	}

	return nil
}

// Write each context part in parallel
func writeContextParts(contextParts []shared.ModelContextPart) {
	var wg sync.WaitGroup
	for _, part := range contextParts {
		wg.Add(1)
		go func(p shared.ModelContextPart) {
			defer wg.Done()
			if err := writeContextPartToFile(p); err != nil {
				// Handling the error in the goroutine by logging. Depending on your application,
				// you might want a different strategy (e.g., collect errors and handle them after waiting).
				fmt.Fprintf(os.Stderr, "Error writing context part to file: %v", err)
			}
		}(part)
	}
	wg.Wait() // Wait for all goroutines to finish writing
}
