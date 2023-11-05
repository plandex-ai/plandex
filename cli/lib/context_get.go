package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/plandex/plandex/shared"
)

func GetAllContext(metaOnly bool) ([]shared.ModelContextPart, error) {
	files, err := os.ReadDir(ContextSubdir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read context directory: %v", err)
		return nil, err
	}

	var context []shared.ModelContextPart
	for _, file := range files {
		filename := file.Name()

		if filename == ".git" || filename == "context.json" {
			continue
		}

		// Only process .meta files and then look for their corresponding .body files
		if strings.HasSuffix(filename, ".meta") {
			// fmt.Fprintf(os.Stderr, "Reading meta context file %s\n", filename)

			metaContent, err := os.ReadFile(filepath.Join(ContextSubdir, filename))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read meta file %s: %v", filename, err)
				return nil, err
			}

			var contextPart shared.ModelContextPart
			if err := json.Unmarshal(metaContent, &contextPart); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to unmarshal JSON from file %s: %v", filename, err)
				return nil, err
			}

			if !metaOnly {
				// get the body filename by replacing the .meta suffix with .body
				bodyFilename := strings.TrimSuffix(filename, ".meta") + ".body"
				bodyPath := filepath.Join(ContextSubdir, bodyFilename)

				// fmt.Fprintf(os.Stderr, "Reading body context file %s\n", bodyPath)

				bodyContent, err := os.ReadFile(bodyPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to read body file %s: %v", bodyPath, err)
					return nil, err
				}

				contextPart.Body = string(bodyContent)
			}

			context = append(context, contextPart)
		}
	}

	// sort by timestamp ascending
	sort.Slice(context, func(i, j int) bool {
		return context[i].AddedAt < context[j].AddedAt
	})

	return context, nil
}
