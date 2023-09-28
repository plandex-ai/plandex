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

func worker(filesDir string, srcPath string, info os.FileInfo, err error, errCh chan<- error) {
	if err != nil {
		errCh <- err
		return
	}
	if info.IsDir() {
		return
	}

	// Compute relative path
	relPath, err := filepath.Rel(filesDir, srcPath)
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
	filesDir := filepath.Join(CurrentPlanRootDir, "plan", "files")

	// Check if filesDir exists
	_, err := os.Stat(filesDir)
	exists := !os.IsNotExist(err)

	if exists {
		errCh := make(chan error)
		// Enumerate all paths in [planDir]/files
		err = filepath.Walk(filesDir, func(srcPath string, info os.FileInfo, err error) error {
			go worker(filesDir, srcPath, info, err, errCh)
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

	var execContent string
	var execExists bool
	execPath := filepath.Join(CurrentPlanRootDir, "exec.sh")
	_, err = os.Stat(execPath)
	execExists = !os.IsNotExist(err)

	if execExists {
		// exec.sh exists
		// Read file content and set it to planFiles["exec"]
		content, err := os.ReadFile(execPath)
		if err != nil {
			return shared.CurrentPlanFiles{}, err
		}
		execContent = string(content)
	}

	return shared.CurrentPlanFiles{Files: planFiles.data, Exec: execContent}, nil
}
