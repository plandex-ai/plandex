package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plandex-server/syntax"
	"plandex/lib"
	"plandex/types"
	"sync"
	"time"

	"github.com/plandex/plandex/shared"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if len(os.Args) < 2 {
		fmt.Println("usage: mapper [files-or-dirs...]")
		os.Exit(1)
	}

	inputPaths := os.Args[1:]

	flattenedPaths, err := lib.ParseInputPaths(inputPaths, &types.LoadContextParams{
		DefsOnly: true,
	})

	if err != nil {
		fmt.Printf("error parsing input paths: %v\n", err)
		os.Exit(1)
	}

	var filteredPaths []string
	for _, path := range flattenedPaths {
		if shared.IsTreeSitterExtension(filepath.Ext(path)) {
			filteredPaths = append(filteredPaths, path)
		}
	}

	errCh := make(chan error, len(filteredPaths))
	fileInputs := map[string]string{}
	var mu sync.Mutex

	for _, path := range filteredPaths {
		go func(path string) {
			content, err := os.ReadFile(path)
			if err != nil {
				errCh <- fmt.Errorf("error reading file: %v", err)
				return
			}
			mu.Lock()
			fileInputs[path] = string(content)
			mu.Unlock()
			errCh <- nil
		}(path)
	}

	for i := 0; i < len(filteredPaths); i++ {
		err := <-errCh
		if err != nil {
			fmt.Printf("error reading file: %v\n", err)
			os.Exit(1)
		}
	}

	mapBodies, err := syntax.ProcessMapFiles(ctx, fileInputs)
	if err != nil {
		fmt.Printf("error processing map files: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(mapBodies.CombinedMap())
}
