package syntax

import (
	"context"
	"fmt"
	"sync"

	"github.com/plandex/plandex/shared"
)

// handles concurrent processing of multiple files for mapping
func ProcessMapFiles(ctx context.Context, inputs map[string]string) (shared.FileMapBodies, error) {
	bodies := make(shared.FileMapBodies, len(inputs))
	var mu sync.Mutex
	errCh := make(chan error, len(inputs))

	for path, content := range inputs {
		go func(path, content string) {
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
