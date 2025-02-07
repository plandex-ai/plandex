package file_map

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"plandex-server/syntax"
	"runtime"
	"sort"
	"strings"
	"sync"

	shared "plandex-shared"

	tree_sitter "github.com/smacker/go-tree-sitter"
)

// handles concurrent processing of multiple files for mapping
func ProcessMapFiles(ctx context.Context, inputs map[string]string) (shared.FileMapBodies, error) {
	log.Println("ProcessMapFiles")
	bodies := make(shared.FileMapBodies, len(inputs))
	var mu sync.Mutex
	errCh := make(chan error, len(inputs))

	// Use half of available CPUs
	cpus := runtime.NumCPU()
	log.Printf("ProcessMapFiles: Available CPUs: %d", cpus)
	maxWorkers := cpus / 2
	if maxWorkers < 1 {
		maxWorkers = 1 // Ensure at least one worker
	}
	log.Printf("ProcessMapFiles: Max workers: %d", maxWorkers)
	sem := make(chan struct{}, maxWorkers)

	sortedPaths := make([]string, 0, len(inputs))
	for path := range inputs {
		sortedPaths = append(sortedPaths, path)
	}
	// sort paths by content length, longest first for more efficient cpu utilization
	sort.Slice(sortedPaths, func(i, j int) bool {
		return len(inputs[sortedPaths[i]]) > len(inputs[sortedPaths[j]])
	})

	for _, path := range sortedPaths {
		content := inputs[path]
		sem <- struct{}{}
		go func(path, content string) {
			defer func() { <-sem }()
			fileMap, err := MapFile(ctx, path, []byte(content))
			if err != nil {
				errCh <- fmt.Errorf("error mapping file %s: %v", path, err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			bodies[path] = fileMap.String()
			errCh <- nil
		}(path, content)
	}

	for i := 0; i < len(inputs); i++ {
		err := <-errCh
		if err != nil {
			return nil, err
		}
	}

	return bodies, nil
}

type MapTrees map[string]string

func ProcessMapTrees(ctx context.Context, inputs map[string]string) (MapTrees, error) {
	trees := make(MapTrees, len(inputs))
	var mu sync.Mutex
	errCh := make(chan error, len(inputs))

	for path, content := range inputs {
		go func(path, content string) {
			// Get appropriate parser
			var parser *tree_sitter.Parser
			file := filepath.Base(path)
			if strings.Contains(strings.ToLower(file), "dockerfile") {
				parser = syntax.GetParserForLanguage(shared.LanguageDockerfile)
			} else {
				parser, _, _, _ = syntax.GetParserForPath(path)

				if parser == nil {
					errCh <- fmt.Errorf("unsupported file type: %s", path)
					return
				}
			}

			contentBytes := []byte(content)

			// Parse file
			tree, err := parser.ParseCtx(ctx, nil, contentBytes)
			if err != nil {
				errCh <- fmt.Errorf("failed to parse file: %v", err)
				return
			}
			defer tree.Close()

			mu.Lock()
			defer mu.Unlock()
			trees[path] = string(tree.RootNode().String())
			errCh <- nil
		}(path, content)
	}

	for i := 0; i < len(inputs); i++ {
		err := <-errCh
		if err != nil {
			return nil, err
		}
	}

	return trees, nil
}

func (m MapTrees) CombinedTrees() string {
	var combinedMap strings.Builder
	paths := make([]string, 0, len(m))
	for path := range m {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		body := m[path]
		body = strings.TrimSpace(body)
		if body == "" {
			continue
		}
		fileHeading := fmt.Sprintf("\n### %s\n", path)
		combinedMap.WriteString(fileHeading)
		combinedMap.WriteString(body)
		combinedMap.WriteString("\n")
	}
	return combinedMap.String()
}
