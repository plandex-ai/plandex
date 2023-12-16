package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"plandex/api"
	"plandex/term"
	"plandex/types"
	"plandex/url"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/plandex/plandex/shared"
)

func MustUpdateContextWithOuput() {
	timeStart := time.Now()

	s := spinner.New(spinner.CharSets[33], 100*time.Millisecond)
	s.Prefix = "🔄 Updating context... "
	s.Start()

	stopFn := func() {
		elapsed := time.Since(timeStart)
		if elapsed < 700*time.Millisecond {
			time.Sleep(700*time.Millisecond - elapsed)
		}
		s.Stop()
		term.ClearCurrentLine()
	}

	updateRes, err := UpdateContext()

	if err != nil {
		stopFn()
		fmt.Fprintln(os.Stderr, "Error updating context:", err)
		os.Exit(1)
	}

	stopFn()

	fmt.Println("✅ " + updateRes.Msg)

}

func UpdateContext() (*types.ContextOutdatedResult, error) {
	return checkOutdatedAndMaybeUpdateContext(true)
}

func CheckOutdatedContext() (*types.ContextOutdatedResult, error) {
	return checkOutdatedAndMaybeUpdateContext(false)
}

func checkOutdatedAndMaybeUpdateContext(doUpdate bool) (*types.ContextOutdatedResult, error) {

	contexts, err := api.Client.ListContext(CurrentPlanId)
	if err != nil {
		return nil, fmt.Errorf("error retrieving context: %w", err)
	}

	var errs []error

	req := shared.UpdateContextRequest{}
	var updatedContexts []*shared.Context
	var numFiles int
	var numUrls int
	var numTrees int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, context := range contexts {

		if context.ContextType == shared.ContextFileType {
			wg.Add(1)
			go func(context *shared.Context) {
				defer wg.Done()
				fileContent, err := os.ReadFile(context.FilePath)

				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to read the file %s: %v", context.FilePath, err))
					return
				}

				hash := sha256.Sum256(fileContent)
				sha := hex.EncodeToString(hash[:])

				if sha != context.Sha {
					numFiles++
					updatedContexts = append(updatedContexts, context)

					body := string(fileContent)

					req[context.Id] = &shared.UpdateContextParams{
						Body: body,
					}
				}
			}(context)

		} else if context.ContextType == shared.ContextDirectoryTreeType {
			wg.Add(1)
			go func(context *shared.Context) {
				defer wg.Done()
				flattenedPaths, err := ParseInputPaths([]string{context.FilePath}, &types.LoadContextParams{
					NamesOnly: true,
				})

				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					errs = append(errs, fmt.Errorf("failed to get the directory tree %s: %v", context.FilePath, err))
					return
				}
				body := strings.Join(flattenedPaths, "\n")
				bytes := []byte(body)

				hash := sha256.Sum256(bytes)
				sha := hex.EncodeToString(hash[:])

				if sha != context.Sha {
					numTrees++
					updatedContexts = append(updatedContexts, context)
					req[context.Id] = &shared.UpdateContextParams{
						Body: body,
					}
				}
			}(context)

		} else if context.ContextType == shared.ContextURLType {
			wg.Add(1)
			go func(part *shared.Context) {
				defer wg.Done()
				body, err := url.FetchURLContent(part.Url)

				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					errs = append(errs, fmt.Errorf("failed to fetch the URL %s: %v", part.Url, err))
					return
				}

				hash := sha256.Sum256([]byte(body))
				sha := hex.EncodeToString(hash[:])

				if sha != part.Sha {
					numUrls++
					updatedContexts = append(updatedContexts, part)
					req[part.Id] = &shared.UpdateContextParams{
						Body: body,
					}
				}

			}(context)
		}
	}

	wg.Wait()

	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to check context outdated: %v", errs)
	}

	var msg string

	if len(req) == 0 {
		return &types.ContextOutdatedResult{
			Msg: "Context is up to date",
		}, nil
	} else if doUpdate {
		res, err := api.Client.UpdateContext(CurrentPlanId, req)
		if err != nil {
			return nil, fmt.Errorf("failed to update context: %w", err)
		}
		msg = res.Msg
	}

	return &types.ContextOutdatedResult{
		Msg:             msg,
		UpdatedContexts: updatedContexts,
		NumFiles:        numFiles,
		NumUrls:         numUrls,
		NumTrees:        numTrees,
	}, nil
}
