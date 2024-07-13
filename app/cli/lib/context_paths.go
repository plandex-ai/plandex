package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"plandex/types"
	"strings"
	"sync"
)

func ParseInputPaths(fileOrDirPaths []string, params *types.LoadContextParams) ([]string, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	resPaths := []string{}

	for _, path := range fileOrDirPaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			err := filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				mu.Lock()
				defer mu.Unlock()
				if firstErr != nil {
					return firstErr // If an error was encountered, stop walking
				}

				if info.IsDir() {
					if info.Name() == ".git" || strings.Index(info.Name(), ".plandex") == 0 {
						return filepath.SkipDir
					}

					if !(params.Recursive || params.NamesOnly) {
						// log.Println("path", path, "info.Name()", info.Name())

						return fmt.Errorf("cannot process directory %s: --recursive or --tree flag not set", path)
					}

					// calculate directory depth from base
					// depth := strings.Count(path[len(p):], string(filepath.Separator))
					// if params.MaxDepth != -1 && depth > params.MaxDepth {
					// 	return filepath.SkipDir
					// }

					if params.NamesOnly {
						// add directory name to results
						resPaths = append(resPaths, path)
					}
				} else {
					// add file path to results
					resPaths = append(resPaths, path)
				}

				return nil
			})

			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(path)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return resPaths, nil
}
