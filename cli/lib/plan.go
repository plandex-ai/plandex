package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/plandex/plandex/shared"
)

type safePlanFiles struct {
	mux  sync.Mutex
	data map[string]string
}

func (s *safePlanFiles) Add(key string, value string) {
	s.mux.Lock()
	s.data[key] = value
	s.mux.Unlock()
}

var planFiles safePlanFiles

func worker(PlanFilesDir string, srcPath string, info os.FileInfo, err error, errCh chan<- error) {
	if err != nil {
		errCh <- err
		return
	}
	if info.IsDir() {
		return
	}

	// Compute relative path
	relPath, err := filepath.Rel(PlanFilesDir, srcPath)
	if err != nil {
		errCh <- err
		return
	}

	// Read file content
	content, err := os.ReadFile(srcPath)
	if err != nil {
		errCh <- err
		return
	}

	// Add file content to planFiles with relative path as key
	planFiles.Add(relPath, string(content))
}

func getCurrentPlanFiles() (shared.CurrentPlanFiles, error) {
	planFiles.data = make(map[string]string)

	// Check if filesDir exists
	_, err := os.Stat(PlanFilesDir)
	exists := !os.IsNotExist(err)

	if exists {
		errCh := make(chan error)
		// Enumerate all paths in [planDir]/files
		err = filepath.Walk(PlanFilesDir, func(srcPath string, info os.FileInfo, err error) error {
			go worker(PlanFilesDir, srcPath, info, err, errCh)
			return nil
		})

		close(errCh)
		for err := range errCh {
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error processing files: %v", err)
				return shared.CurrentPlanFiles{}, err
			}
		}
	}

	return shared.CurrentPlanFiles{Files: planFiles.data}, nil
}

func getCurrentPlanFilePaths() ([]string, error) {
	// Check if filesDir exists
	_, err := os.Stat(PlanFilesDir)
	exists := !os.IsNotExist(err)

	filePaths := make([]string, 0)

	if exists {
		// Enumerate all paths in [planDir]/files
		err = filepath.Walk(PlanFilesDir, func(srcPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			filePaths = append(filePaths, srcPath)
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing files: %v", err)
			return []string{}, err
		}
	}

	return filePaths, nil
}

func isFilePathInPlan(filePath string) bool {
	filePaths, err := getCurrentPlanFilePaths()
	if err != nil {
		return false
	}

	for _, path := range filePaths {
		if path == filePath {
			return true
		}
	}

	return false
}
