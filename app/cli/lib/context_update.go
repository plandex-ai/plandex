package lib

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"plandex-cli/api"
	"plandex-cli/fs"
	"plandex-cli/term"
	"plandex-cli/types"
	"plandex-cli/url"
	"strconv"
	"strings"
	"sync"

	shared "plandex-shared"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

func CheckOutdatedContextWithOutput(quiet, autoConfirm bool, maybeContexts []*shared.Context, projectPaths *types.ProjectPaths) (contextOutdated, updated bool, err error) {
	if !quiet {
		term.StartSpinner("🔬 Checking context...")
	}

	var contexts []*shared.Context

	if maybeContexts != nil {
		contexts = maybeContexts
	} else {
		res, err := api.Client.ListContext(CurrentPlanId, CurrentBranch)
		if err != nil {
			term.StopSpinner()
			return false, false, fmt.Errorf("failed to list context: %s", err)
		}
		contexts = res
	}

	outdatedRes, err := CheckOutdatedContext(contexts, projectPaths)
	if err != nil {
		term.StopSpinner()
		return false, false, fmt.Errorf("failed to check outdated context: %s", err)
	}

	if !quiet {
		term.StopSpinner()
	}

	if len(outdatedRes.UpdatedContexts) == 0 && len(outdatedRes.RemovedContexts) == 0 {
		if !quiet {
			fmt.Println("✅ Context is up to date")
		}
		return false, false, nil
	}
	if len(outdatedRes.UpdatedContexts) > 0 {
		types := []string{}
		if outdatedRes.NumFiles > 0 {
			lbl := "file"
			if outdatedRes.NumFiles > 1 {
				lbl = "files"
			}
			lbl = strconv.Itoa(outdatedRes.NumFiles) + " " + lbl
			types = append(types, lbl)
		}
		if outdatedRes.NumUrls > 0 {
			lbl := "url"
			if outdatedRes.NumUrls > 1 {
				lbl = "urls"
			}
			lbl = strconv.Itoa(outdatedRes.NumUrls) + " " + lbl
			types = append(types, lbl)
		}
		if outdatedRes.NumTrees > 0 {
			lbl := "directory tree"
			if outdatedRes.NumTrees > 1 {
				lbl = "directory trees"
			}
			lbl = strconv.Itoa(outdatedRes.NumTrees) + " " + lbl
			types = append(types, lbl)
		}
		if outdatedRes.NumMaps > 0 {
			lbl := "map"
			if outdatedRes.NumMaps > 1 {
				lbl = "maps"
			}
			lbl = strconv.Itoa(outdatedRes.NumMaps) + " " + lbl
			types = append(types, lbl)
		}

		var msg string
		if len(types) <= 2 {
			msg += strings.Join(types, " and ")
		} else {
			for i, add := range types {
				if i == len(types)-1 {
					msg += ", and " + add
				} else {
					msg += ", " + add
				}
			}
		}

		phrase := "have been"
		if len(outdatedRes.UpdatedContexts) == 1 {
			phrase = "has been"
		}

		if !quiet {
			term.StopSpinner()

			color.New(term.ColorHiCyan, color.Bold).Printf("%s in context %s modified 👇\n\n", msg, phrase)

			tableString := tableForContextOutdated(outdatedRes.UpdatedContexts, outdatedRes.TokenDiffsById)
			fmt.Println(tableString)
		}
	}

	if len(outdatedRes.RemovedContexts) > 0 {
		types := []string{}
		if outdatedRes.NumFilesRemoved > 0 {
			lbl := "file"
			if outdatedRes.NumFilesRemoved > 1 {
				lbl = "files"
			}
			lbl = strconv.Itoa(outdatedRes.NumFilesRemoved) + " " + lbl
			types = append(types, lbl)
		}
		if outdatedRes.NumTreesRemoved > 0 {
			lbl := "directory tree"
			if outdatedRes.NumTreesRemoved > 1 {
				lbl = "directory trees"
			}
			lbl = strconv.Itoa(outdatedRes.NumTreesRemoved) + " " + lbl
			types = append(types, lbl)
		}

		var msg string
		if len(types) <= 2 {
			msg += strings.Join(types, " and ")
		} else {
			for i, add := range types {
				if i == len(types)-1 {
					msg += ", and " + add
				} else {
					msg += ", " + add
				}
			}
		}

		phrase := "have been"
		if len(outdatedRes.RemovedContexts) == 1 {
			phrase = "has been"
		}

		if !quiet {
			term.StopSpinner()

			color.New(term.ColorHiCyan, color.Bold).Printf("%s in context %s removed 👇\n\n", msg, phrase)

			tableString := tableForContextOutdated(outdatedRes.RemovedContexts, outdatedRes.TokenDiffsById)
			fmt.Println(tableString)
		}
	}

	confirmed := autoConfirm

	if !autoConfirm {
		confirmed, err = term.ConfirmYesNo("Update context now?")

		if err != nil {
			term.OutputErrorAndExit("failed to get user input: %s", err)
		}
	}

	if confirmed {
		reqFn := outdatedRes.ReqFn
		if reqFn == nil {
			return false, false, fmt.Errorf("no update request function provided")
		}
		_, err = UpdateContextWithOutput(UpdateContextParams{
			Contexts:    contexts,
			OutdatedRes: *outdatedRes,
			ReqFn:       reqFn,
		})
		if err != nil {
			return false, false, fmt.Errorf("error updating context: %v", err)
		}
		return true, true, nil
	} else {
		return true, false, nil
	}

}

type UpdateContextParams struct {
	Contexts    []*shared.Context
	OutdatedRes types.ContextOutdatedResult
	ReqFn       func() (map[string]*shared.UpdateContextParams, error)
}

type UpdateContextResult struct {
	HasConflicts bool
	Msg          string
}

func UpdateContextWithOutput(params UpdateContextParams) (UpdateContextResult, error) {
	term.StartSpinner("🔄 Updating context...")

	updateRes, err := UpdateContext(params)
	if err != nil {
		return UpdateContextResult{}, err
	}

	term.StopSpinner()

	fmt.Println("✅ " + updateRes.Msg)

	return updateRes, nil
}

func UpdateContext(params UpdateContextParams) (UpdateContextResult, error) {
	var err error
	reqFn := params.ReqFn
	if reqFn == nil {
		return UpdateContextResult{}, fmt.Errorf("no update request function provided")
	}
	req, err := reqFn()
	if err != nil {
		return UpdateContextResult{}, fmt.Errorf("error getting update request: %v", err)
	}
	var hasConflicts bool
	var msg string

	contextsById := map[string]*shared.Context{}
	for _, context := range params.Contexts {
		contextsById[context.Id] = context
	}
	deleteIds := map[string]bool{}
	for _, context := range params.OutdatedRes.RemovedContexts {
		deleteIds[context.Id] = true
	}

	filesToLoad := map[string]string{}
	for id := range req {
		context := contextsById[id]
		if context.ContextType == shared.ContextFileType {
			filesToLoad[context.FilePath] = context.Body
		}
	}
	for id := range deleteIds {
		context := contextsById[id]
		if context.ContextType == shared.ContextFileType {
			filesToLoad[context.FilePath] = ""
		}
	}

	hasConflicts, err = checkContextConflicts(filesToLoad)
	if err != nil {
		return UpdateContextResult{}, fmt.Errorf("failed to check context conflicts: %v", err)
	}

	if len(req) > 0 {
		res, apiErr := api.Client.UpdateContext(CurrentPlanId, CurrentBranch, req)
		if apiErr != nil {
			return UpdateContextResult{}, fmt.Errorf("failed to update context: %v", apiErr)
		}
		msg = res.Msg
	}

	if len(deleteIds) > 0 {
		res, apiErr := api.Client.DeleteContext(CurrentPlanId, CurrentBranch, shared.DeleteContextRequest{
			Ids: deleteIds,
		})
		if apiErr != nil {
			return UpdateContextResult{}, fmt.Errorf("failed to delete contexts: %v", apiErr)
		}
		msg += " " + res.Msg
	}

	return UpdateContextResult{
		HasConflicts: hasConflicts,
		Msg:          strings.TrimSpace(msg),
	}, nil
}

// CheckOutdatedContext is where we replicate your partial-read logic for map files
// so that large map files or newly added map files do not read more than MaxContextMapSingleInputSize
func CheckOutdatedContext(maybeContexts []*shared.Context, projectPaths *types.ProjectPaths) (*types.ContextOutdatedResult, error) {
	return checkOutdatedAndMaybeUpdateContext(false, maybeContexts, projectPaths)
}

type mapState struct {
	removedMapPaths      []string
	mapInputShas         map[string]string
	mapInputTokens       map[string]int
	mapInputSizes        map[string]int64
	totalMapSize         int64
	currentMapInputBatch shared.FileMapInputs
	mapInputBatches      []shared.FileMapInputs
}

func checkOutdatedAndMaybeUpdateContext(doUpdate bool, maybeContexts []*shared.Context, projectPaths *types.ProjectPaths) (*types.ContextOutdatedResult, error) {
	var contexts []*shared.Context

	if maybeContexts == nil {
		contextsRes, apiErr := api.Client.ListContext(CurrentPlanId, CurrentBranch)
		if apiErr != nil {
			return nil, fmt.Errorf("error retrieving context: %v", apiErr)
		}
		contexts = contextsRes
	} else {
		contexts = maybeContexts
	}

	totalTokens := 0
	for _, c := range contexts {
		totalTokens += c.NumTokens
	}

	var errs []error

	reqFns := map[string]func() (*shared.UpdateContextParams, error){}

	var updatedContexts []*shared.Context
	var tokenDiffsById = map[string]int{}
	var numFiles int
	var numUrls int
	var numTrees int
	var numMaps int
	var numFilesRemoved int
	var numTreesRemoved int
	var mu sync.Mutex
	var wg sync.WaitGroup
	contextsById := make(map[string]*shared.Context)
	deleteIds := make(map[string]bool)

	paths := projectPaths

	// We track skipped items for final warnings
	var filesSkippedTooLarge []filePathWithSize
	var filesSkippedAfterSizeLimit []string

	var mapFilesTruncatedTooLarge []filePathWithSize
	var mapFilesSkippedAfterSizeLimit []string

	mapFilesTruncatedSet := map[string]bool{}
	mapFilesSkippedAfterSizeLimitSet := map[string]bool{}

	mapFileInfoByPath := map[string]os.FileInfo{}
	mapFileRemovedByPath := map[string]bool{}

	var totalSize int64
	var totalBodySize int64
	var totalContextCount int

	sem := make(chan struct{}, ContextMapMaxClientConcurrency)

	for _, c := range contexts {
		contextsById[c.Id] = c
	}

	for _, context := range contexts {
		switch context.ContextType {
		case shared.ContextFileType:
			wg.Add(1)
			go func(ctx *shared.Context) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				if _, err := os.Stat(ctx.FilePath); os.IsNotExist(err) {
					mu.Lock()
					defer mu.Unlock()

					deleteIds[ctx.Id] = true
					numFilesRemoved++
					tokenDiffsById[ctx.Id] = -ctx.NumTokens
					return
				}

				fileContent, err := os.ReadFile(ctx.FilePath)
				if err != nil {
					mu.Lock()
					defer mu.Unlock()
					errs = append(errs, fmt.Errorf("failed to read the file %s: %v", ctx.FilePath, err))
					return
				}
				fileContent = shared.NormalizeEOL(fileContent)

				fileInfo, err := os.Stat(ctx.FilePath)
				if err != nil {
					mu.Lock()
					defer mu.Unlock()
					errs = append(errs, fmt.Errorf("failed to get file info for %s: %v", ctx.FilePath, err))
					return
				}
				size := fileInfo.Size()

				// Individual skip checks
				if size > shared.MaxContextBodySize {
					mu.Lock()
					defer mu.Unlock()

					filesSkippedTooLarge = append(filesSkippedTooLarge, filePathWithSize{Path: ctx.FilePath, Size: size})
					return
				}
				if totalSize+size > shared.MaxContextBodySize {
					mu.Lock()
					defer mu.Unlock()

					filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.FilePath)
					return
				}

				// Compare new sha
				hash := sha256.Sum256(fileContent)
				sha := hex.EncodeToString(hash[:])

				if sha != ctx.Sha {
					if totalContextCount >= shared.MaxContextCount {
						mu.Lock()
						defer mu.Unlock()

						filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.FilePath)
						return
					}

					oldBodySize := int64(len(ctx.Body))
					newBodySize := int64(len(fileContent))
					if totalBodySize+(newBodySize-oldBodySize) > shared.MaxContextBodySize {
						mu.Lock()
						defer mu.Unlock()

						filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.FilePath)
						return
					}

					// Accept
					mu.Lock()
					totalSize += size
					totalContextCount++
					totalBodySize += (newBodySize - oldBodySize)
					mu.Unlock()

					var numTokens int
					if shared.IsImageFile(ctx.FilePath) {
						tokens, err := shared.GetImageTokens(base64.StdEncoding.EncodeToString(fileContent), ctx.ImageDetail)
						if err != nil {
							mu.Lock()
							defer mu.Unlock()
							errs = append(errs, fmt.Errorf("failed to get image tokens for %s: %v", ctx.FilePath, err))
							return
						}
						numTokens = tokens
					} else {
						numTokens = shared.GetNumTokensEstimate(string(fileContent))
					}

					mu.Lock()
					defer mu.Unlock()

					tokenDiffsById[ctx.Id] = numTokens - ctx.NumTokens
					numFiles++
					updatedContexts = append(updatedContexts, ctx)

					reqFns[ctx.Id] = func() (*shared.UpdateContextParams, error) {
						return &shared.UpdateContextParams{
							Body: string(fileContent),
						}, nil
					}
				}
			}(context)

		case shared.ContextDirectoryTreeType:
			wg.Add(1)
			go func(ctx *shared.Context) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				if _, err := os.Stat(ctx.FilePath); os.IsNotExist(err) {
					mu.Lock()
					deleteIds[ctx.Id] = true
					numTreesRemoved++
					tokenDiffsById[ctx.Id] = -ctx.NumTokens
					mu.Unlock()
					return
				}

				baseDir := fs.GetBaseDirForFilePaths([]string{ctx.FilePath})
				flattenedPaths, err := ParseInputPaths(ParseInputPathsParams{
					FileOrDirPaths: []string{ctx.FilePath},
					BaseDir:        baseDir,
					ProjectPaths:   paths,
					LoadParams: &types.LoadContextParams{
						NamesOnly:       true,
						ForceSkipIgnore: ctx.ForceSkipIgnore,
					},
				})

				if err != nil {
					mu.Lock()
					defer mu.Unlock()

					errs = append(errs, fmt.Errorf("failed to get directory tree %s: %v", ctx.FilePath, err))
					return
				}

				if !ctx.ForceSkipIgnore && paths != nil {
					var filtered []string
					for _, p := range flattenedPaths {
						if _, ok := paths.ActivePaths[p]; ok {
							filtered = append(filtered, p)
						}
					}
					flattenedPaths = filtered
				}

				// Partial skipping for sub-paths
				var kept []string
				mu.Lock()
				for _, p := range flattenedPaths {
					lineSize := int64(len(p))
					// If line is individually too large, skip
					if lineSize > shared.MaxContextBodySize {
						filesSkippedTooLarge = append(filesSkippedTooLarge, filePathWithSize{Path: p, Size: lineSize})
						continue
					}
					if totalSize+lineSize > shared.MaxContextBodySize {
						filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, p)
						continue
					}
					// Accept
					totalSize += lineSize
					kept = append(kept, p)
				}
				mu.Unlock()

				body := strings.Join(kept, "\n")
				newHash := sha256.Sum256([]byte(body))
				newSha := hex.EncodeToString(newHash[:])

				if newSha != ctx.Sha {
					if totalContextCount >= shared.MaxContextCount {
						mu.Lock()
						defer mu.Unlock()

						filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.FilePath)
						return
					}

					oldBodySize := int64(len(ctx.Body))
					newBodySize := int64(len(body))
					if totalBodySize+(newBodySize-oldBodySize) > shared.MaxContextBodySize {
						mu.Lock()
						defer mu.Unlock()

						filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.FilePath)
						return
					}

					numTokens := shared.GetNumTokensEstimate(body)

					mu.Lock()
					defer mu.Unlock()

					totalContextCount++
					totalBodySize += (newBodySize - oldBodySize)
					tokenDiffsById[ctx.Id] = numTokens - ctx.NumTokens
					numTrees++
					updatedContexts = append(updatedContexts, ctx)
					reqFns[ctx.Id] = func() (*shared.UpdateContextParams, error) {
						return &shared.UpdateContextParams{
							Body: body,
						}, nil
					}
				}
			}(context)

		case shared.ContextMapType:
			// Instead of reading all files in the same goroutine,
			// we now spawn one goroutine per map-file to mirror the loading logic concurrency.
			wg.Add(1)
			go func(ctx *shared.Context) {
				defer wg.Done()

				// We collect paths from the existing map
				var mapPaths []string
				for path := range ctx.MapShas {
					mapPaths = append(mapPaths, path)
				}

				// Next, see if there are newly added files
				baseDir := fs.GetBaseDirForFilePaths([]string{ctx.FilePath})
				flattenedPaths, err := ParseInputPaths(ParseInputPathsParams{
					FileOrDirPaths: []string{ctx.FilePath},
					BaseDir:        baseDir,
					ProjectPaths:   projectPaths,
					LoadParams:     &types.LoadContextParams{Recursive: true},
				})
				if err != nil {
					mu.Lock()
					defer mu.Unlock()
					errs = append(errs, fmt.Errorf("failed to get the directory tree %s: %v", ctx.FilePath, err))
					return
				}

				var filtered []string
				if projectPaths != nil {
					for _, p := range flattenedPaths {
						if _, ok := projectPaths.ActivePaths[p]; ok {
							filtered = append(filtered, p)
						}
					}
					flattenedPaths = filtered
				}

				// If a path was not already in the map, it's newly added
				for _, p := range flattenedPaths {
					if _, ok := ctx.MapShas[p]; !ok {
						mapPaths = append(mapPaths, p)
					}
				}

				totalMapPaths := len(mapPaths)

				currentMapInputBatch := shared.FileMapInputs{}
				state := mapState{
					removedMapPaths:      []string{},
					mapInputShas:         map[string]string{},
					mapInputTokens:       map[string]int{},
					mapInputSizes:        map[string]int64{},
					totalMapSize:         0,
					currentMapInputBatch: currentMapInputBatch,
					mapInputBatches:      []shared.FileMapInputs{currentMapInputBatch},
				}

				innerExistenceErrCh := make(chan error, len(mapPaths))

				// Existence: check each path in its own goroutine:

				for _, path := range mapPaths {
					go func(path string) {
						sem <- struct{}{}
						defer func() { <-sem }()

						var removed bool

						var hasFileInfo bool
						mu.Lock()
						if _, ok := mapFileInfoByPath[path]; ok {
							hasFileInfo = true
						} else if _, ok := mapFileRemovedByPath[path]; ok {
							removed = true
						}
						mu.Unlock()

						if !(hasFileInfo || removed) {
							fileInfo, err := os.Stat(path)
							if err != nil {
								if os.IsNotExist(err) {
									removed = true

								} else {
									innerExistenceErrCh <- fmt.Errorf("failed to stat map file %s: %v", path, err)
									return
								}
							}

							mu.Lock()
							prevTokens := ctx.MapTokens[path]
							prevSize := ctx.MapSizes[path]

							if removed {
								mapFileRemovedByPath[path] = true
								totalMapPaths--
								if _, existed := ctx.MapShas[path]; existed {
									state.removedMapPaths = append(state.removedMapPaths, path)
									tokenDiffsById[ctx.Id] -= prevTokens
									state.totalMapSize -= prevSize
								}
							} else {
								mapFileInfoByPath[path] = fileInfo
							}
							mu.Unlock()
						}

						innerExistenceErrCh <- nil
					}(path)
				}

				for range mapPaths {
					err := <-innerExistenceErrCh
					if err != nil {
						mu.Lock()
						defer mu.Unlock()
						errs = append(errs, err)
						return
					}
				}

				// Updates: check each path in its own goroutine:
				innerUpdatesErrCh := make(chan error, len(mapPaths))

				for _, path := range mapPaths {
					go func(path string) {
						sem <- struct{}{}
						defer func() { <-sem }()

						var removed bool
						var fileInfo os.FileInfo

						var hasFileInfo bool
						mu.Lock()
						if _, ok := mapFileInfoByPath[path]; ok {
							fileInfo = mapFileInfoByPath[path]
							hasFileInfo = true
						} else if _, ok := mapFileRemovedByPath[path]; ok {
							removed = true
						}
						mu.Unlock()

						if removed {
							innerUpdatesErrCh <- nil
							return
						}

						if !hasFileInfo {
							innerUpdatesErrCh <- fmt.Errorf("failed to get map file info for %s - should already be set", path)
							return
						}

						size := fileInfo.Size()
						var totalMapSize int64
						mu.Lock()
						prevTokens := ctx.MapTokens[path]
						prevSize := ctx.MapSizes[path]
						prevSha := ctx.MapShas[path]
						totalMapSize = state.totalMapSize
						mu.Unlock()

						res, err := getMapFileDetails(path, size, totalMapSize)
						if err != nil {
							innerUpdatesErrCh <- fmt.Errorf("failed to get map file details for %s: %v", path, err)
							return
						}

						if res.shaVal == prevSha {
							// no change
							innerUpdatesErrCh <- nil
							return
						}

						mu.Lock()

						totalMapPaths++

						if totalMapPaths > shared.MaxContextMapPaths {
							if _, ok := mapFilesSkippedAfterSizeLimitSet[path]; !ok {
								mapFilesSkippedAfterSizeLimitSet[path] = true
								mapFilesSkippedAfterSizeLimit = append(mapFilesSkippedAfterSizeLimit, path)
							}
							mu.Unlock()
							innerUpdatesErrCh <- nil
							return
						}

						defer mu.Unlock()

						if state.currentMapInputBatch.NumFiles()+1 > shared.ContextMapMaxBatchSize || state.totalMapSize+res.size > shared.ContextMapMaxBatchBytes {
							state.currentMapInputBatch = shared.FileMapInputs{}
							state.mapInputBatches = append(state.mapInputBatches, state.currentMapInputBatch)
						}

						sizeChange := int64(res.size) - prevSize
						state.totalMapSize += sizeChange
						tokenDiffsById[ctx.Id] += (res.tokens - prevTokens)

						state.mapInputShas[path] = res.shaVal
						state.mapInputTokens[path] = res.tokens
						state.currentMapInputBatch[path] = res.mapContent
						state.mapInputSizes[path] = res.size

						if len(res.mapFilesTruncatedTooLarge) > 0 {
							for _, file := range res.mapFilesTruncatedTooLarge {
								if _, ok := mapFilesTruncatedSet[file.Path]; !ok {
									mapFilesTruncatedSet[file.Path] = true
									mapFilesTruncatedTooLarge = append(mapFilesTruncatedTooLarge, file)
								}
							}
						}

						if len(res.mapFilesSkippedAfterSizeLimit) > 0 {
							for _, file := range res.mapFilesSkippedAfterSizeLimit {
								if _, ok := mapFilesSkippedAfterSizeLimitSet[file]; !ok {
									mapFilesSkippedAfterSizeLimitSet[file] = true
									mapFilesSkippedAfterSizeLimit = append(mapFilesSkippedAfterSizeLimit, file)
								}
							}
						}

						innerUpdatesErrCh <- nil
					}(path)
				}

				for range mapPaths {
					err := <-innerUpdatesErrCh
					if err != nil {
						mu.Lock()
						defer mu.Unlock()
						errs = append(errs, err)
						return
					}
				}

				hasAnyUpdate := len(state.removedMapPaths) > 0 || len(state.mapInputShas) > 0

				if hasAnyUpdate {
					mu.Lock()
					defer mu.Unlock()

					updatedContexts = append(updatedContexts, ctx)

					numMaps++

					reqFns[ctx.Id] = func() (*shared.UpdateContextParams, error) {
						updatedMapBodies, err := processMapBatches(state.mapInputBatches)
						if err != nil {
							return nil, fmt.Errorf("failed to process map batches: %v", err)
						}

						return &shared.UpdateContextParams{
							MapBodies:       updatedMapBodies,
							InputShas:       state.mapInputShas,
							InputTokens:     state.mapInputTokens,
							InputSizes:      state.mapInputSizes,
							RemovedMapPaths: state.removedMapPaths,
						}, nil
					}
				}

			}(context)

		case shared.ContextURLType:
			wg.Add(1)
			go func(ctx *shared.Context) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				body, err := url.FetchURLContent(ctx.Url)

				if err != nil {
					mu.Lock()
					defer mu.Unlock()
					errs = append(errs, fmt.Errorf("failed to fetch the URL %s: %v", ctx.Url, err))
					return
				}

				size := int64(len(body))
				if size > shared.MaxContextBodySize {
					mu.Lock()
					defer mu.Unlock()
					filesSkippedTooLarge = append(filesSkippedTooLarge, filePathWithSize{Path: ctx.Url, Size: size})
					return
				}
				if totalSize+size > shared.MaxContextBodySize {
					mu.Lock()
					defer mu.Unlock()
					filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.Url)
					return
				}
				hash := sha256.Sum256([]byte(body))
				newSha := hex.EncodeToString(hash[:])
				if newSha != ctx.Sha {
					if totalContextCount >= shared.MaxContextCount {
						mu.Lock()
						defer mu.Unlock()
						filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.Url)
						return
					}
					oldBodySize := int64(len(ctx.Body))
					newBodySize := size
					if totalBodySize+(newBodySize-oldBodySize) > shared.MaxContextBodySize {
						mu.Lock()
						defer mu.Unlock()
						filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.Url)
						return
					}

					numTokens := shared.GetNumTokensEstimate(string(body))

					mu.Lock()
					defer mu.Unlock()

					totalSize += size
					totalContextCount++
					totalBodySize += (newBodySize - oldBodySize)

					tokenDiffsById[ctx.Id] = numTokens - ctx.NumTokens

					numUrls++
					updatedContexts = append(updatedContexts, ctx)
					reqFns[ctx.Id] = func() (*shared.UpdateContextParams, error) {
						return &shared.UpdateContextParams{
							Body: string(body),
						}, nil
					}
				}
			}(context)
		}
	}

	wg.Wait()

	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to check context outdated: %v", errs)
	}

	// Identify contexts to remove
	var removedContexts []*shared.Context
	for id := range deleteIds {
		removedContexts = append(removedContexts, contextsById[id])
	}

	// If nothing changed
	if len(reqFns) == 0 && len(removedContexts) == 0 {
		return &types.ContextOutdatedResult{
			Msg: "Context is up to date",
		}, nil
	}

	reqFn := func() (map[string]*shared.UpdateContextParams, error) {
		req := map[string]*shared.UpdateContextParams{}
		var mu sync.Mutex

		errCh := make(chan error, len(reqFns))
		for id, fn := range reqFns {
			go func(id string, fn func() (*shared.UpdateContextParams, error)) {
				res, err := fn()
				if err != nil {
					errCh <- err
					return
				}
				mu.Lock()
				req[id] = res
				mu.Unlock()
				errCh <- nil
			}(id, fn)
		}
		for i := 0; i < len(reqFns); i++ {
			err := <-errCh
			if err != nil {
				return nil, err
			}
		}
		return req, nil
	}

	// Build final result
	outdatedRes := types.ContextOutdatedResult{
		UpdatedContexts: updatedContexts,
		RemovedContexts: removedContexts,
		TokenDiffsById:  tokenDiffsById,
		NumFiles:        numFiles,
		NumUrls:         numUrls,
		NumTrees:        numTrees,
		NumMaps:         numMaps,
		NumFilesRemoved: numFilesRemoved,
		NumTreesRemoved: numTreesRemoved,
		ReqFn:           reqFn,
	}

	var hasConflicts bool
	var msg string
	if doUpdate {
		res, err := UpdateContext(UpdateContextParams{
			Contexts:    contexts,
			OutdatedRes: outdatedRes,
			ReqFn:       reqFn,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update context: %v", err)
		}
		hasConflicts = res.HasConflicts
		msg = res.Msg
		outdatedRes.Msg = msg
	} else {
		var tokensDiff int
		for _, diff := range tokenDiffsById {
			tokensDiff += diff
		}
		newTotal := totalTokens + tokensDiff
		outdatedRes.Msg = shared.SummaryForUpdateContext(shared.SummaryForUpdateContextParams{
			NumFiles:    numFiles,
			NumTrees:    numTrees,
			NumUrls:     numUrls,
			NumMaps:     numMaps,
			TokensDiff:  tokensDiff,
			TotalTokens: newTotal,
		})
	}

	if hasConflicts {
		term.StartSpinner("🏗️  Starting build...")
		_, err := buildPlanInlineFn(false, nil)
		term.StopSpinner()
		fmt.Println()
		if err != nil {
			return nil, fmt.Errorf("failed to build plan: %v", err)
		}
	}

	// Warn about any items skipped
	if len(filesSkippedTooLarge) > 0 || len(filesSkippedAfterSizeLimit) > 0 ||
		len(mapFilesTruncatedTooLarge) > 0 || len(mapFilesSkippedAfterSizeLimit) > 0 {
		printSkippedFilesMsg(filesSkippedTooLarge, filesSkippedAfterSizeLimit,
			mapFilesTruncatedTooLarge, mapFilesSkippedAfterSizeLimit)
	}

	return &outdatedRes, nil
}

func tableForContextOutdated(updatedContexts []*shared.Context, tokenDiffsById map[string]int) string {
	if len(updatedContexts) == 0 {
		return ""
	}

	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Name", "Type", "🪙"})
	table.SetAutoWrapText(false)

	for _, ctx := range updatedContexts {
		t, icon := ctx.TypeAndIcon()
		diff := tokenDiffsById[ctx.Id]
		diffStr := "+" + strconv.Itoa(diff)
		tableColor := tablewriter.FgHiGreenColor

		if diff < 0 {
			diffStr = strconv.Itoa(diff)
			tableColor = tablewriter.FgHiRedColor
		}

		row := []string{
			" " + icon + " " + ctx.Name,
			t,
			diffStr,
		}

		table.Rich(row, []tablewriter.Colors{
			{tableColor, tablewriter.Bold},
			{tableColor},
			{tableColor},
		})
	}

	table.Render()
	return tableString.String()
}
