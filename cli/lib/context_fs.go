package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/plandex/plandex/shared"
)

// CreateContextFileName constructs a filename based on the given name and sha.
func CreateContextFileName(name, sha string) string {
	// Extract the first 8 characters of the sha
	shaSubstring := sha[:8]
	return fmt.Sprintf("%s.%s", name, shaSubstring)
}

func ContextRemoveFiles(paths []string) error {
	// remove files
	errCh := make(chan error, len(paths)*2)
	for _, path := range paths {
		for _, ext := range []string{".meta", ".body"} {
			go func(path, ext string) {
				errCh <- os.Remove(filepath.Join(ContextSubdir, path+ext))
			}(path, ext)
		}
	}

	for i := 0; i < len(paths)*2; i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error removing context file: %v", err)
		}
	}

	return nil
}

// writeContextPartToFile writes a single ModelContextPart to a file.
func writeContextPartToFile(part *shared.ModelContextPart) error {
	metaFilename := CreateContextFileName(part.Name, part.Sha) + ".meta"
	metaPath := filepath.Join(ContextSubdir, metaFilename)

	bodyFilename := CreateContextFileName(part.Name, part.Sha) + ".body"
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
func writeContextParts(contextParts []*shared.ModelContextPart) error {
	var errCh = make(chan error)
	for _, part := range contextParts {
		go func(p *shared.ModelContextPart) {
			if err := writeContextPartToFile(p); err != nil {
				// Handling the error in the goroutine by logging. Depending on your application,
				// you might want a different strategy (e.g., collect errors and handle them after waiting).
				err := fmt.Errorf("Error writing context part to file: %v", err)
				errCh <- err
			}
			errCh <- nil
		}(part)
	}

	// Wait for all goroutines to finish
	for range contextParts {
		if err := <-errCh; err != nil {
			return err
		}
	}

	return nil

}
