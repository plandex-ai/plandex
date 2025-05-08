package lib

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"plandex-cli/api"
	"plandex-cli/types"
	shared "plandex-shared"
	"sync"

	"github.com/sashabaranov/go-openai"
)

func AutoLoadContextFiles(ctx context.Context, files []string) (string, error) {
	contexts, err := api.Client.ListContext(CurrentPlanId, CurrentBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get contexts: %v", err)
	}

	var totalSize int64
	totalContexts := len(contexts)

	for _, context := range contexts {
		totalSize += context.BodySize
	}

	loadContextReqsByIndex := make(map[int]*shared.LoadContextParams)
	filesSkippedTooLarge := []filePathWithSize{}
	filesSkippedAfterSizeLimit := []string{}

	var mu sync.Mutex
	errCh := make(chan error, len(files))

	for i, path := range files {
		totalContexts++
		if totalContexts > shared.MaxContextCount {
			log.Println("Skipping file", path, "because it would exceed the max context count", totalContexts)
			filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, path)
			errCh <- nil
			continue
		}

		go func(index int, path string) {
			fileInfo, err := os.Stat(path)
			if err != nil {
				errCh <- fmt.Errorf("failed to get file info for %s: %v", path, err)
				return
			}

			if fileInfo.IsDir() {
				log.Println("Skipping directory", path)
				errCh <- nil // skip directories
				return
			}

			size := fileInfo.Size()

			mu.Lock()
			if size > shared.MaxContextBodySize {
				log.Println("Skipping file", path, "because it's too large", size)
				filesSkippedTooLarge = append(filesSkippedTooLarge, filePathWithSize{Path: path, Size: size})
				mu.Unlock()
				errCh <- nil
				return
			}
			if totalSize+size > shared.MaxTotalContextSize {
				log.Println("Skipping file", path, "because it would exceed the max context body size", totalSize+size)
				filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, path)
				mu.Unlock()
				errCh <- nil
				return
			}
			totalSize += size
			mu.Unlock()

			b, err := os.ReadFile(path)
			if err != nil {
				errCh <- fmt.Errorf("failed to read file %s: %v", path, err)
				return
			}

			var contextType shared.ContextType
			isImage := shared.IsImageFile(path)
			if isImage {
				contextType = shared.ContextImageType
			} else {
				contextType = shared.ContextFileType
			}

			var imageDetail openai.ImageURLDetail
			if isImage {
				imageDetail = openai.ImageURLDetailHigh
			}

			var body string
			if isImage {
				body = base64.StdEncoding.EncodeToString(b)
			} else {
				body = string(shared.NormalizeEOL(b))
			}

			mu.Lock()
			loadContextReqsByIndex[index] = &shared.LoadContextParams{
				ContextType: contextType,
				FilePath:    path,
				Name:        path,
				Body:        body,
				AutoLoaded:  true,
				ImageDetail: imageDetail,
			}
			mu.Unlock()
			errCh <- nil
		}(i, path)
	}

	for range files {
		if e := <-errCh; e != nil {
			return "", fmt.Errorf("failed to load context: %v", e)
		}
	}

	// Convert map back to ordered slice
	loadContextReqs := make(shared.LoadContextRequest, 0, len(loadContextReqsByIndex))
	for i := 0; i < len(files); i++ {
		if req := loadContextReqsByIndex[i]; req != nil {
			loadContextReqs = append(loadContextReqs, req)
		}
	}

	// even if there are no files to load, we still need to hit the API endpoint because the stream is waiting on a channel for the autoload to finish
	res, apiErr := api.Client.AutoLoadContext(ctx, CurrentPlanId, CurrentBranch, loadContextReqs)
	if apiErr != nil {
		return "", fmt.Errorf("failed to load context: %v", apiErr.Msg)
	}

	if res.MaxTokensExceeded {
		overage := res.TotalTokens - res.MaxTokens
		return "", fmt.Errorf("update would add %d 🪙 and exceed token limit (%d) by %d 🪙", res.TokensAdded, res.MaxTokens, overage)
	}

	msg := res.Msg

	// Print skip info if any
	if len(filesSkippedTooLarge) > 0 || len(filesSkippedAfterSizeLimit) > 0 {
		msg += "\n\n" + getSkippedFilesMsg(filesSkippedTooLarge, filesSkippedAfterSizeLimit, nil, nil)
	}

	return msg, nil
}

func MustLoadAutoContextMap() {
	MustLoadContext([]string{"."}, &types.LoadContextParams{
		DefsOnly:          true,
		SkipIgnoreWarning: true,
		AutoLoaded:        true,
	})
}
