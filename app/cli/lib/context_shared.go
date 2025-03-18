package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"plandex-cli/api"
	"strings"
	"sync"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

const ContextMapMaxClientConcurrency = 250

type filePathWithSize struct {
	Path string
	Size int64
}

type mapFileDetails struct {
	size                          int64
	tokens                        int
	shaVal                        string
	mapFilesSkippedAfterSizeLimit []string
	mapFilesTruncatedTooLarge     []filePathWithSize
	mapContent                    string
}

func getMapFileDetails(path string, size, mapSize int64) (mapFileDetails, error) {
	var isImage bool
	var totalMapSizeExceeded bool

	res := mapFileDetails{
		size:                          size,
		mapFilesSkippedAfterSizeLimit: []string{},
		mapFilesTruncatedTooLarge:     []filePathWithSize{},
	}

	if !shared.HasFileMapSupport(path) {
		if shared.IsImageFile(path) {
			isImage = true

			var err error
			res.tokens, err = readImageTokensForDefsOnly(path, size, openai.ImageURLDetailHigh, 8*1024)
			if err != nil {
				return mapFileDetails{}, fmt.Errorf("failed to read image tokens for %s: %v", path, err)
			}
		} else {
			res.tokens = shared.GetBytesToTokensEstimate(size)
		}
	} else {
		var truncated bool
		if size > shared.MaxContextMapSingleInputSize {
			size = shared.MaxContextMapSingleInputSize
			truncated = true
			res.tokens = shared.GetBytesToTokensEstimate(size)
		}

		// should go in either skip list *or* truncated list, not both
		if mapSize+size > shared.MaxContextMapTotalInputSize {
			totalMapSizeExceeded = true
			res.mapFilesSkippedAfterSizeLimit = append(res.mapFilesSkippedAfterSizeLimit, path)
			res.tokens = shared.GetBytesToTokensEstimate(size)
		} else if truncated {
			res.mapFilesTruncatedTooLarge = append(res.mapFilesTruncatedTooLarge, filePathWithSize{Path: path, Size: size})
		}
	}

	if totalMapSizeExceeded || !shared.HasFileMapSupport(path) || isImage {
		shaVal := sha256.Sum256([]byte(fmt.Sprintf("%d", res.tokens)))
		res.shaVal = hex.EncodeToString(shaVal[:])

		res.mapContent = ""
		res.size = 0
	} else {
		// partial read for the map
		contentRes, err := getMapFileContent(path)
		if err != nil {
			return mapFileDetails{}, fmt.Errorf("failed to read file %s: %v", path, err)
		}

		res.mapContent = contentRes.content
		res.shaVal = contentRes.shaVal

		if contentRes.truncated {
			res.mapFilesTruncatedTooLarge = append(res.mapFilesTruncatedTooLarge, filePathWithSize{Path: path, Size: shared.MaxContextMapSingleInputSize})
			res.size = shared.MaxContextMapSingleInputSize
			res.tokens = shared.GetBytesToTokensEstimate(shared.MaxContextMapSingleInputSize)
		} else {
			// do the actual token count if we didn't truncate
			res.tokens = shared.GetNumTokensEstimate(res.mapContent)
		}
	}

	return res, nil
}

type mapFileContent struct {
	mapData   []byte
	content   string
	shaVal    string
	truncated bool
}

func getMapFileContent(path string) (mapFileContent, error) {
	f, err := os.Open(path)
	if err != nil {
		return mapFileContent{}, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return mapFileContent{}, err
	}
	size := info.Size()

	limit := int64(shared.MaxContextMapSingleInputSize)
	truncated := size > limit

	limitReader := io.LimitReader(f, limit)
	bytes, err := io.ReadAll(limitReader)
	if err != nil {
		return mapFileContent{}, err
	}

	sum := sha256.Sum256(bytes)
	shaVal := hex.EncodeToString(sum[:])

	return mapFileContent{mapData: bytes, content: string(bytes), shaVal: shaVal, truncated: truncated}, nil
}

func processMapBatches(mapInputBatches []shared.FileMapInputs) (shared.FileMapBodies, error) {
	allMapBodies := shared.FileMapBodies{}

	var mapMu sync.Mutex
	errCh := make(chan error, len(mapInputBatches))

	for _, batch := range mapInputBatches {
		if len(batch) == 0 {
			errCh <- nil
			continue
		}

		go func(batch shared.FileMapInputs) {
			mapRes, apiErr := api.Client.GetFileMap(shared.GetFileMapRequest{
				MapInputs: batch,
			})
			if apiErr != nil {
				errCh <- fmt.Errorf("failed to get file map: %v", apiErr)
				return
			}
			mapMu.Lock()
			for path, bodies := range mapRes.MapBodies {
				allMapBodies[path] = bodies
			}
			mapMu.Unlock()
			errCh <- nil
		}(batch)
	}

	for i := 0; i < len(mapInputBatches); i++ {
		err := <-errCh
		if err != nil {
			return nil, err
		}
	}

	return allMapBodies, nil
}

func readImageTokensForDefsOnly(path string, size int64, detail openai.ImageURLDetail, headerBytes int64) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	tokens, err := shared.GetImageTokensFromHeader(file, detail, headerBytes)
	if err != nil {
		tokens = shared.GetImageTokensEstimateFromBytes(size)
	}
	return tokens, nil
}

func printSkippedFilesMsg(
	filesSkippedTooLarge []filePathWithSize,
	filesSkippedAfterSizeLimit []string,
	mapFilesTruncatedTooLarge []filePathWithSize,
	mapFilesSkippedAfterSizeLimit []string,
) {
	fmt.Println()
	fmt.Println(getSkippedFilesMsg(filesSkippedTooLarge, filesSkippedAfterSizeLimit, mapFilesTruncatedTooLarge, mapFilesSkippedAfterSizeLimit))
}

func getSkippedFilesMsg(
	filesSkippedTooLarge []filePathWithSize,
	filesSkippedAfterSizeLimit []string,
	mapFilesTruncatedTooLarge []filePathWithSize,
	mapFilesSkippedAfterSizeLimit []string,
) string {
	var builder strings.Builder

	if len(filesSkippedTooLarge) > 0 {
		fmt.Fprintf(&builder, "ℹ️  These files were skipped because they're too large:\n")
		for i, file := range filesSkippedTooLarge {
			if i >= maxSkippedFileList {
				fmt.Fprintf(&builder, "  • and %d more\n", len(filesSkippedTooLarge)-maxSkippedFileList)
				break
			}
			fmt.Fprintf(&builder, "  • %s - %d MB\n", file.Path, file.Size/1024/1024)
		}
	}
	if len(mapFilesTruncatedTooLarge) > 0 {
		fmt.Fprintf(&builder, "ℹ️  These files were truncated because they're too large to map fully:\n")
		for i, file := range mapFilesTruncatedTooLarge {
			if i >= maxSkippedFileList {
				fmt.Fprintf(&builder, "  • and %d more\n", len(mapFilesTruncatedTooLarge)-maxSkippedFileList)
				break
			}
			if file.Size > 1024*1024 {
				fmt.Fprintf(&builder, "  • %s - %d MB\n", file.Path, file.Size/1024/1024)
			} else {
				fmt.Fprintf(&builder, "  • %s - %d KB\n", file.Path, file.Size/1024)
			}
		}
		if len(mapFilesTruncatedTooLarge) > 0 {
			fmt.Fprintf(&builder, "They will still be included in the map, but only the first %d KB will be mapped.\n", shared.MaxContextMapSingleInputSize/1024)
		}
	}
	if len(filesSkippedAfterSizeLimit) > 0 {
		fmt.Fprintf(&builder, "ℹ️  These files were skipped because the total size limit was exceeded:\n")
		for i, file := range filesSkippedAfterSizeLimit {
			if i >= maxSkippedFileList {
				fmt.Fprintf(&builder, "  • and %d more\n", len(filesSkippedAfterSizeLimit)-maxSkippedFileList)
				break
			}
			fmt.Fprintf(&builder, "  • %s\n", file)
		}
	}
	if len(mapFilesSkippedAfterSizeLimit) > 0 {
		fmt.Fprintf(&builder, "ℹ️  These files were skipped because the total map size limit was exceeded:\n")
		for i, file := range mapFilesSkippedAfterSizeLimit {
			if i >= maxSkippedFileList {
				fmt.Fprintf(&builder, "  • and %d more\n", len(mapFilesSkippedAfterSizeLimit)-maxSkippedFileList)
				break
			}
			fmt.Fprintf(&builder, "  • %s\n", file)
		}
		if len(mapFilesSkippedAfterSizeLimit) > 0 {
			fmt.Fprintf(&builder, "They will still be included in the map as paths in the project, but no maps will be generated for them.\n")
		}
	}
	return builder.String()
}
