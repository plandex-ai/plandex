package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"plandex/types"
	"strings"
	"sync"

	"github.com/plandex/plandex/shared"
)

func UpdateContext() ([]string, int, bool, error) {
	return checkOutdatedAndMaybeUpdateContext(true)
}

func CheckOutdatedContext() ([]string, int, bool, error) {
	return checkOutdatedAndMaybeUpdateContext(false)
}

type contextUpdate struct {
	Path string
	Part *shared.ModelContextPart
}

func checkOutdatedAndMaybeUpdateContext(doUpdate bool) ([]string, int, bool, error) {
	maxTokens := shared.MaxContextTokens
	planState, err := GetPlanState()
	if err != nil {
		return nil, 0, false, fmt.Errorf("error retrieving plan state: %w", err)
	}

	tokensDiff := 0
	totalTokens := planState.ContextTokens
	var mu sync.Mutex

	context, err := GetAllContext(true)
	if err != nil {
		return nil, 0, false, fmt.Errorf("error retrieving context: %w", err)
	}

	ts := shared.StringTs()

	updateCh := make(chan *contextUpdate, len(context))
	errCh := make(chan error, len(context))

	var wg sync.WaitGroup

	for _, part := range context {
		contextPath := CreateContextFileName(part.Name, part.Sha)

		if part.Type == shared.ContextFileType {
			wg.Add(1)
			go func(part *shared.ModelContextPart) {
				defer wg.Done()
				fileContent, err := os.ReadFile(part.FilePath)
				if err != nil {
					errCh <- fmt.Errorf("failed to read the file %s: %v", part.FilePath, err)
					return
				}

				hash := sha256.Sum256(fileContent)
				sha := hex.EncodeToString(hash[:])

				if sha == part.Sha {
					updateCh <- nil
				} else {
					body := string(fileContent)
					numTokens := shared.GetNumTokens(body)

					mu.Lock()
					tokensDiff += numTokens - part.NumTokens
					mu.Unlock()

					updateCh <- &contextUpdate{Path: contextPath, Part: &shared.ModelContextPart{
						Type:      shared.ContextFileType,
						Name:      part.Name,
						FilePath:  part.FilePath,
						Body:      body,
						Sha:       sha,
						NumTokens: numTokens,
						AddedAt:   part.AddedAt,
						UpdatedAt: ts,
					}}
				}
			}(part)

		} else if part.Type == shared.ContextDirectoryTreeType {
			wg.Add(1)
			go func(part *shared.ModelContextPart) {
				defer wg.Done()
				flattenedPaths, err := ParseInputPaths([]string{part.FilePath}, &types.LoadContextParams{
					NamesOnly: true,
				})
				if err != nil {
					errCh <- fmt.Errorf("failed to get the directory tree %s: %v", part.FilePath, err)
					return
				}
				body := strings.Join(flattenedPaths, "\n")
				bytes := []byte(body)

				hash := sha256.Sum256(bytes)
				sha := hex.EncodeToString(hash[:])

				if sha == part.Sha {
					updateCh <- nil
				} else {
					numTokens := shared.GetNumTokens(body)
					mu.Lock()
					tokensDiff += numTokens - part.NumTokens
					mu.Unlock()

					updateCh <- &contextUpdate{Path: contextPath, Part: &shared.ModelContextPart{
						Type:      shared.ContextDirectoryTreeType,
						Name:      part.Name,
						FilePath:  part.FilePath,
						Body:      body,
						Sha:       sha,
						NumTokens: numTokens,
						AddedAt:   part.AddedAt,
						UpdatedAt: ts,
					}}
				}
			}(part)

		} else if part.Type == shared.ContextURLType {
			wg.Add(1)
			go func(part *shared.ModelContextPart) {
				defer wg.Done()
				body, err := FetchURLContent(part.Url)
				if err != nil {
					errCh <- fmt.Errorf("failed to fetch the URL %s: %v", part.Url, err)
					return
				}

				hash := sha256.Sum256([]byte(body))
				sha := hex.EncodeToString(hash[:])

				if sha == part.Sha {
					updateCh <- nil
				} else {
					numTokens := shared.GetNumTokens(body)
					mu.Lock()
					tokensDiff += numTokens - part.NumTokens
					mu.Unlock()

					updateCh <- &contextUpdate{Path: contextPath, Part: &shared.ModelContextPart{
						Type:      shared.ContextURLType,
						Name:      part.Name,
						Url:       part.Url,
						Body:      body,
						Sha:       sha,
						NumTokens: numTokens,
						AddedAt:   part.AddedAt,
						UpdatedAt: ts,
					}}
				}

			}(part)
		} else {
			updateCh <- nil
		}
	}

	wg.Wait()

	updatedPaths := []string{}
	updatedParts := []*shared.ModelContextPart{}

	for i := 0; i < len(context); i++ {
		select {
		case err := <-errCh:
			return nil, 0, false, err
		case update := <-updateCh:
			if update != nil {
				updatedPaths = append(updatedPaths, update.Path)
				updatedParts = append(updatedParts, update.Part)
			}
		}
	}

	totalTokens += tokensDiff

	if doUpdate {
		if totalTokens > maxTokens {
			return nil, 0, true, fmt.Errorf("context update would exceed the token limit of %d", maxTokens)
		}

		err = ContextRm(updatedPaths)
		if err != nil {
			return nil, 0, false, fmt.Errorf("error removing updated context paths: %w", err)
		}

		err = writeContextParts(updatedParts)
		if err != nil {
			return nil, 0, false, fmt.Errorf("error writing updated context parts: %w", err)
		}

		planState.ContextTokens = totalTokens
		err = SetPlanState(planState, ts)

		if err != nil {
			return nil, 0, false, fmt.Errorf("error writing plan state: %w", err)
		}
	}

	return updatedPaths, tokensDiff, totalTokens > maxTokens, nil
}
