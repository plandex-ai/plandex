package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

func GetPlanContexts(orgId, planId string, includeBody bool) ([]*Context, error) {
	var contexts []*Context
	contextDir := getPlanContextDir(orgId, planId)

	// get all context files
	files, err := os.ReadDir(contextDir)
	if err != nil {
		if os.IsNotExist(err) {
			return contexts, nil
		}

		return nil, fmt.Errorf("error reading context dir: %v", err)
	}

	errCh := make(chan error, len(files)/2)
	contextCh := make(chan *Context, len(files)/2)

	// read each context file
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".meta") {
			go func(file os.DirEntry) {
				context, err := GetContext(orgId, planId, strings.TrimSuffix(file.Name(), ".meta"), includeBody)

				if err != nil {
					errCh <- fmt.Errorf("error reading context file: %v", err)
					return
				}

				contextCh <- context
			}(file)
		}
	}

	for i := 0; i < len(files)/2; i++ {
		select {
		case err := <-errCh:
			return nil, fmt.Errorf("error reading context files: %v", err)
		case context := <-contextCh:
			contexts = append(contexts, context)
		}
	}

	// sort contexts by CreatedAt
	sort.Slice(contexts, func(i, j int) bool {
		return contexts[i].CreatedAt.Before(contexts[j].CreatedAt)
	})

	return contexts, nil
}

func GetContext(orgId, planId, contextId string, includeBody bool) (*Context, error) {
	contextDir := getPlanContextDir(orgId, planId)

	// read the meta file
	metaPath := filepath.Join(contextDir, contextId+".meta")

	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("error reading context meta file: %v", err)
	}

	var context Context
	err = json.Unmarshal(metaBytes, &context)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling context meta file: %v", err)
	}

	if includeBody {
		// read the body file
		bodyPath := filepath.Join(contextDir, strings.TrimSuffix(contextId, ".meta")+".body")
		bodyBytes, err := os.ReadFile(bodyPath)

		if err != nil {
			return nil, fmt.Errorf("error reading context body file: %v", err)
		}

		context.Body = string(bodyBytes)
	}

	return &context, nil
}

func ContextRemove(contexts []*Context) error {
	// remove files
	numFiles := len(contexts) * 2

	errCh := make(chan error, numFiles)
	for _, context := range contexts {
		contextDir := getPlanContextDir(context.OrgId, context.PlanId)
		for _, ext := range []string{".meta", ".body"} {
			go func(context *Context, dir, ext string) {
				errCh <- os.Remove(filepath.Join(dir, context.Id+ext))
			}(context, contextDir, ext)
		}
	}

	for i := 0; i < numFiles; i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error removing context file: %v", err)
		}
	}

	return nil
}

func StoreContext(context *Context) error {
	contextDir := getPlanContextDir(context.OrgId, context.PlanId)

	err := os.MkdirAll(contextDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating context dir: %v", err)
	}

	ts := time.Now().UTC()
	if context.Id == "" {
		context.Id = uuid.New().String()
		context.CreatedAt = ts
	}
	context.UpdatedAt = ts

	metaFilename := context.Id + ".meta"
	metaPath := filepath.Join(contextDir, metaFilename)

	originalBody := context.Body

	bodyFilename := context.Id + ".body"
	bodyPath := filepath.Join(contextDir, bodyFilename)
	body := []byte(context.Body)
	context.Body = ""

	// Convert the ModelContextPart to JSON
	data, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context context: %v", err)
	}

	// Write the body to the file
	if err = os.WriteFile(bodyPath, body, 0644); err != nil {
		return fmt.Errorf("failed to write context body to file %s: %v", bodyPath, err)
	}

	// Write the meta data to the file
	if err = os.WriteFile(metaPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write context meta to file %s: %v", metaPath, err)
	}

	context.Body = originalBody

	return nil
}
