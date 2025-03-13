package main

// import (
// 	"context"
// 	"fmt"
// 	"os"
// 	"plandex-server/syntax/file_map"
// 	"plandex/lib"
// 	"plandex/types"
// 	"sync"
// 	"time"

// 	shared "plandex-shared"
// )

// func main() {
// 	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
// 	defer cancel()

// 	args := os.Args[1:]

// 	if len(args) < 1 {
// 		fmt.Println("usage: mapper [files-or-dirs...]")
// 		os.Exit(1)
// 	}

// 	var parserTree bool = false

// 	for i, arg := range args {
// 		if arg == "--trees" {
// 			parserTree = true
// 			args = append(args[:i], args[i+1:]...)
// 			break
// 		}
// 	}

// 	flattenedPaths, err := lib.ParseInputPaths(args, &types.LoadContextParams{
// 		DefsOnly: true,
// 	})

// 	if err != nil {
// 		fmt.Printf("error parsing input paths: %v\n", err)
// 		os.Exit(1)
// 	}

// 	var filteredPaths []string
// 	for _, path := range flattenedPaths {
// 		if shared.HasFileMapSupport(path) {
// 			filteredPaths = append(filteredPaths, path)
// 		}
// 	}

// 	errCh := make(chan error, len(filteredPaths))
// 	fileInputs := map[string]string{}
// 	var mu sync.Mutex

// 	for _, path := range filteredPaths {
// 		go func(path string) {
// 			content, err := os.ReadFile(path)
// 			if err != nil {
// 				errCh <- fmt.Errorf("error reading file: %v", err)
// 				return
// 			}
// 			mu.Lock()
// 			fileInputs[path] = string(content)
// 			mu.Unlock()
// 			errCh <- nil
// 		}(path)
// 	}

// 	for i := 0; i < len(filteredPaths); i++ {
// 		err := <-errCh
// 		if err != nil {
// 			fmt.Printf("error reading file: %v\n", err)
// 			os.Exit(1)
// 		}
// 	}

// 	if parserTree {
// 		trees, err := file_map.ProcessMapTrees(ctx, fileInputs)
// 		if err != nil {
// 			fmt.Printf("error processing map files: %v\n", err)
// 			os.Exit(1)
// 		}
// 		fmt.Println(trees.CombinedTrees())
// 	} else {
// 		mapBodies, err := file_map.ProcessMapFiles(ctx, fileInputs)
// 		if err != nil {
// 			fmt.Printf("error processing map files: %v\n", err)
// 			os.Exit(1)
// 		}

// 		fmt.Println(mapBodies.CombinedMap())
// 	}
// }
