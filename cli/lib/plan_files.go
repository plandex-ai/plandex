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

func worker(dir string, srcPath string, info os.FileInfo, err error, doneCh chan<- error) {
	// Compute relative path
	relPath, err := filepath.Rel(dir, srcPath)
	if err != nil {
		doneCh <- err
		return
	}

	// Read file content
	content, err := os.ReadFile(srcPath)
	if err != nil {
		doneCh <- err
		return
	}

	// Add file content to planFiles with relative path as key
	planFiles.Add(relPath, string(content))
	doneCh <- nil
}

func getCurrentPlanFiles() (shared.CurrentPlanFiles, error) {
	planFiles.data = make(map[string]string)

	_, err := os.Stat(PlanFilesDir)
	filesDirExists := !os.IsNotExist(err)

	_, err = os.Stat(PlanSectionsDir)
	sectionsDirExists := !os.IsNotExist(err)

	numFiles := 0

	if filesDirExists || sectionsDirExists {
		doneCh := make(chan error)

		if filesDirExists {
			// Enumerate all paths in [planDir]/files
			err = filepath.Walk(PlanFilesDir, func(srcPath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				numFiles++
				go worker(PlanFilesDir, srcPath, info, err, doneCh)
				return nil
			})
		}

		if sectionsDirExists {
			// Enumerate all paths in [planDir]/sections
			err = filepath.Walk(PlanSectionsDir, func(srcPath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				numFiles++
				go worker(PlanSectionsDir, srcPath, info, err, doneCh)
				return nil
			})
		}

		numFinished := 0
		for numFinished < numFiles {
			select {
			case err := <-doneCh:
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error processing files: %v", err)
					return shared.CurrentPlanFiles{}, err
				}
				numFinished++
			}
		}
	}

	return shared.CurrentPlanFiles{Files: planFiles.data}, nil
}

func getCurrentPlanFilePaths() ([]string, error) {
	_, err := os.Stat(PlanFilesDir)
	filesDirExists := !os.IsNotExist(err)

	_, err = os.Stat(PlanSectionsDir)
	sectionsDirExists := !os.IsNotExist(err)

	filePaths := make([]string, 0)

	if filesDirExists {
		// Enumerate all paths in [planDir]/files
		err = filepath.Walk(PlanFilesDir, func(srcPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			filePaths = append(filePaths, srcPath)
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing files: %v", err)
			return []string{}, err
		}
	}

	if sectionsDirExists {
		// Enumerate all paths in [planDir]/sections
		err = filepath.Walk(PlanSectionsDir, func(srcPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
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
