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
		_, err := UpdateContextWithOutput(UpdateContextParams{
			Contexts:    contexts,
			OutdatedRes: *outdatedRes,
			Req:         outdatedRes.Req,
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
	Req         map[string]*shared.UpdateContextParams
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
	req := params.Req

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

	var err error
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

func CheckOutdatedContext(maybeContexts []*shared.Context, projectPaths *types.ProjectPaths) (*types.ContextOutdatedResult, error) {
	return checkOutdatedAndMaybeUpdateContext(false, maybeContexts, projectPaths)
}

func checkOutdatedAndMaybeUpdateContext(doUpdate bool, maybeContexts []*shared.Context, projectPaths *types.ProjectPaths) (*types.ContextOutdatedResult, error) {
	var contexts []*shared.Context

	if maybeContexts == nil {
		var apiErr *shared.ApiError
		contexts, apiErr = api.Client.ListContext(CurrentPlanId, CurrentBranch)
		if apiErr != nil {
			return nil, fmt.Errorf("error retrieving context: %v", apiErr)
		}
	} else {
		contexts = maybeContexts
	}

	totalTokens := 0
	for _, c := range contexts {
		totalTokens += c.NumTokens
	}

	var errs []error

	req := shared.UpdateContextRequest{}
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
	var filesSkippedTooLarge []struct {
		Path string
		Size int64
	}
	var filesSkippedAfterSizeLimit []string
	var mapFilesSkippedTooLarge []struct {
		Path string
		Size int64
	}
	var mapFilesSkippedAfterSizeLimit []string

	// Partial skipping needs to track cumulative sizes
	var totalSize int64
	var totalMapSize int64
	var totalBodySize int64
	var totalContextCount int

	for _, c := range contexts {
		contextsById[c.Id] = c
	}

	for _, context := range contexts {
		switch context.ContextType {
		case shared.ContextFileType:
			wg.Add(1)
			go func(ctx *shared.Context) {
				defer wg.Done()
				mu.Lock()
				defer mu.Unlock()

				if _, e := os.Stat(ctx.FilePath); os.IsNotExist(e) {
					deleteIds[ctx.Id] = true
					numFilesRemoved++
					tokenDiffsById[ctx.Id] = -ctx.NumTokens
					return
				}

				fileContent, e := os.ReadFile(ctx.FilePath)
				if e != nil {
					errs = append(errs, fmt.Errorf("failed to read the file %s: %v", ctx.FilePath, e))
					return
				}
				fileInfo, e := os.Stat(ctx.FilePath)
				if e != nil {
					errs = append(errs, fmt.Errorf("failed to get file info for %s: %v", ctx.FilePath, e))
					return
				}
				size := fileInfo.Size()

				// Individual skip checks
				if size > shared.MaxContextBodySize {
					filesSkippedTooLarge = append(filesSkippedTooLarge, struct {
						Path string
						Size int64
					}{Path: ctx.FilePath, Size: size})
					return
				}
				if totalSize+size > shared.MaxContextBodySize {
					filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.FilePath)
					return
				}

				// Compare new sha
				hash := sha256.Sum256(fileContent)
				sha := hex.EncodeToString(hash[:])
				if sha != ctx.Sha {
					// Check context count & overall body size
					if totalContextCount >= shared.MaxContextCount {
						filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.FilePath)
						return
					}

					oldBodySize := int64(len(ctx.Body))
					newBodySize := int64(len(fileContent))
					if totalBodySize+(newBodySize-oldBodySize) > shared.MaxContextBodySize {
						filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.FilePath)
						return
					}

					// Accept
					totalSize += size
					totalContextCount++
					totalBodySize += (newBodySize - oldBodySize)

					var numTokens int
					if shared.IsImageFile(ctx.FilePath) {
						tokens, err := shared.GetImageTokens(base64.StdEncoding.EncodeToString(fileContent), ctx.ImageDetail)
						if err != nil {
							errs = append(errs, fmt.Errorf("failed to get image tokens for %s: %v", ctx.FilePath, err))
							return
						}
						numTokens = tokens
					} else {
						numTokens = shared.GetNumTokensEstimate(string(fileContent))
					}

					tokenDiffsById[ctx.Id] = numTokens - ctx.NumTokens
					numFiles++
					updatedContexts = append(updatedContexts, ctx)
					req[ctx.Id] = &shared.UpdateContextParams{
						Body: string(fileContent),
					}
				}
			}(context)

		case shared.ContextDirectoryTreeType:
			wg.Add(1)
			go func(ctx *shared.Context) {
				defer wg.Done()

				if _, e := os.Stat(ctx.FilePath); os.IsNotExist(e) {
					mu.Lock()
					deleteIds[ctx.Id] = true
					numTreesRemoved++
					tokenDiffsById[ctx.Id] = -ctx.NumTokens
					mu.Unlock()
					return
				}

				baseDir := fs.GetBaseDirForFilePaths([]string{ctx.FilePath})
				flattenedPaths, e := ParseInputPaths(ParseInputPathsParams{
					FileOrDirPaths: []string{ctx.FilePath},
					BaseDir:        baseDir,
					ProjectPaths:   paths,
					LoadParams: &types.LoadContextParams{
						NamesOnly:       true,
						ForceSkipIgnore: ctx.ForceSkipIgnore,
					},
				})
				mu.Lock()
				defer mu.Unlock()

				if e != nil {
					errs = append(errs, fmt.Errorf("failed to get directory tree %s: %v", ctx.FilePath, e))
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
				for _, p := range flattenedPaths {
					lineSize := int64(len(p))
					// If line is individually too large, skip
					if lineSize > shared.MaxContextBodySize {
						filesSkippedTooLarge = append(filesSkippedTooLarge, struct {
							Path string
							Size int64
						}{Path: p, Size: lineSize})
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
				body := strings.Join(kept, "\n")
				newHash := sha256.Sum256([]byte(body))
				newSha := hex.EncodeToString(newHash[:])

				if newSha != ctx.Sha {
					if totalContextCount >= shared.MaxContextCount {
						filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.FilePath)
						return
					}

					oldBodySize := int64(len(ctx.Body))
					newBodySize := int64(len(body))
					if totalBodySize+(newBodySize-oldBodySize) > shared.MaxContextBodySize {
						filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.FilePath)
						return
					}

					totalContextCount++
					totalBodySize += (newBodySize - oldBodySize)

					numTokens := shared.GetNumTokensEstimate(body)
					tokenDiffsById[ctx.Id] = numTokens - ctx.NumTokens
					numTrees++
					updatedContexts = append(updatedContexts, ctx)
					req[ctx.Id] = &shared.UpdateContextParams{
						Body: body,
					}
				}
			}(context)

		case shared.ContextMapType:
			wg.Add(1)
			go func(ctx *shared.Context) {
				defer wg.Done()
				mu.Lock()
				defer mu.Unlock()

				var removedMapPaths []string
				var updatedInputs = make(shared.FileMapInputs)
				var updatedInputShas = map[string]string{}

				// Check existing files
				for path, currentSha := range ctx.MapShas {
					bytes, e := os.ReadFile(path)
					if e != nil {
						if os.IsNotExist(e) {
							removedMapPaths = append(removedMapPaths, path)
							continue
						}
						errs = append(errs, fmt.Errorf("failed to read map file %s: %v", path, e))
						return
					}

					fileInfo, e := os.Stat(path)
					if e != nil {
						errs = append(errs, fmt.Errorf("failed to stat map file %s: %v", path, e))
						return
					}
					size := fileInfo.Size()

					var effectiveContent string
					if !shared.HasFileMapSupport(path) {
						effectiveContent = ""
					} else if size > shared.MaxContextBodySize {
						effectiveContent = ""
						mapFilesSkippedTooLarge = append(mapFilesSkippedTooLarge, struct {
							Path string
							Size int64
						}{Path: path, Size: size})
					} else if totalMapSize+size > shared.MaxContextMapInputSize {
						effectiveContent = ""
						mapFilesSkippedAfterSizeLimit = append(mapFilesSkippedAfterSizeLimit, path)
					} else {
						effectiveContent = string(bytes)
						totalMapSize += size
					}
					hash := sha256.Sum256([]byte(effectiveContent))
					newSha := hex.EncodeToString(hash[:])
					if newSha != currentSha {
						updatedInputs[path] = effectiveContent
						updatedInputShas[path] = newSha
					}
				}

				// Check newly added files
				baseDir := fs.GetBaseDirForFilePaths([]string{ctx.FilePath})
				flattenedPaths, e := ParseInputPaths(ParseInputPathsParams{
					FileOrDirPaths: []string{ctx.FilePath},
					BaseDir:        baseDir,
					ProjectPaths:   paths,
					LoadParams:     &types.LoadContextParams{Recursive: true},
				})
				if e != nil {
					errs = append(errs, fmt.Errorf("failed to get the directory tree %s: %v", ctx.FilePath, e))
					return
				}

				var filtered []string
				if paths != nil {
					for _, p := range flattenedPaths {
						if _, ok := paths.ActivePaths[p]; ok {
							filtered = append(filtered, p)
						}
					}
					flattenedPaths = filtered
				}

				for _, p := range flattenedPaths {
					if !shared.HasFileMapSupport(p) {
						continue
					}
					// If p not in ctx.MapShas => new file
					if _, ok := ctx.MapShas[p]; !ok {
						bytes, e := os.ReadFile(p)
						if e != nil {
							errs = append(errs, fmt.Errorf("failed to read map file %s: %v", p, e))
							return
						}
						fileInfo, e := os.Stat(p)
						if e != nil {
							errs = append(errs, fmt.Errorf("failed to stat map file %s: %v", p, e))
							return
						}
						size := fileInfo.Size()
						if size > shared.MaxContextBodySize {
							mapFilesSkippedTooLarge = append(mapFilesSkippedTooLarge, struct {
								Path string
								Size int64
							}{Path: p, Size: size})
							continue
						}
						if totalMapSize+size > shared.MaxContextMapInputSize {
							mapFilesSkippedAfterSizeLimit = append(mapFilesSkippedAfterSizeLimit, p)
							continue
						}
						totalMapSize += size
						hash := sha256.Sum256(bytes)
						sha := hex.EncodeToString(hash[:])
						updatedInputs[p] = string(bytes)
						updatedInputShas[p] = sha
					}
				}

				if len(updatedInputs) > 0 || len(removedMapPaths) > 0 {
					// Check if we can update or must skip
					if totalContextCount >= shared.MaxContextCount {
						for p := range updatedInputs {
							mapFilesSkippedAfterSizeLimit = append(mapFilesSkippedAfterSizeLimit, p)
						}
						return
					}
					// Estimate new size
					var newSize int64
					for _, v := range updatedInputs {
						newSize += int64(len(v))
					}
					oldBodySize := int64(len(ctx.Body))
					if totalBodySize+(newSize-oldBodySize) > shared.MaxContextBodySize {
						for p := range updatedInputs {
							mapFilesSkippedAfterSizeLimit = append(mapFilesSkippedAfterSizeLimit, p)
						}
						return
					}

					totalContextCount++
					totalBodySize += (newSize - oldBodySize)

					numMaps++
					updatedContexts = append(updatedContexts, ctx)

					// Use GetFileMap to get final map bodies
					var updatedMapBodies shared.FileMapBodies
					if len(updatedInputs) > 0 {
						mapRes, apiErr := api.Client.GetFileMap(shared.GetFileMapRequest{
							MapInputs: updatedInputs,
						})
						if apiErr != nil {
							errs = append(errs, fmt.Errorf("failed to get file map: %v", apiErr))
							return
						}
						updatedMapBodies = mapRes.MapBodies

						// Update tokens
						for path, body := range mapRes.MapBodies {
							prev := ctx.MapTokens[path]
							numTokens := mapRes.MapBodies.TokenEstimateForPath(path)
							tokenDiffsById[ctx.Id] += numTokens - prev
							_ = body
						}
					}

					req[ctx.Id] = &shared.UpdateContextParams{
						MapBodies:       updatedMapBodies,
						InputShas:       updatedInputShas,
						RemovedMapPaths: removedMapPaths,
					}
				}
			}(context)

		case shared.ContextURLType:
			wg.Add(1)
			go func(ctx *shared.Context) {
				defer wg.Done()
				body, e := url.FetchURLContent(ctx.Url)

				mu.Lock()
				defer mu.Unlock()

				if e != nil {
					errs = append(errs, fmt.Errorf("failed to fetch the URL %s: %v", ctx.Url, e))
					return
				}

				size := int64(len(body))
				if size > shared.MaxContextBodySize {
					filesSkippedTooLarge = append(filesSkippedTooLarge, struct {
						Path string
						Size int64
					}{Path: ctx.Url, Size: size})
					return
				}
				if totalSize+size > shared.MaxContextBodySize {
					filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.Url)
					return
				}

				hash := sha256.Sum256([]byte(body))
				newSha := hex.EncodeToString(hash[:])
				if newSha != ctx.Sha {
					if totalContextCount >= shared.MaxContextCount {
						filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.Url)
						return
					}
					oldBodySize := int64(len(ctx.Body))
					newBodySize := size
					if totalBodySize+(newBodySize-oldBodySize) > shared.MaxContextBodySize {
						filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, ctx.Url)
						return
					}

					totalSize += size
					totalContextCount++
					totalBodySize += (newBodySize - oldBodySize)

					numTokens := shared.GetNumTokensEstimate(body)
					tokenDiffsById[ctx.Id] = numTokens - ctx.NumTokens

					numUrls++
					updatedContexts = append(updatedContexts, ctx)
					req[ctx.Id] = &shared.UpdateContextParams{
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

	// Identify contexts to remove
	var removedContexts []*shared.Context
	for id := range deleteIds {
		removedContexts = append(removedContexts, contextsById[id])
	}

	// If nothing changed
	if len(req) == 0 && len(removedContexts) == 0 {
		return &types.ContextOutdatedResult{
			Msg: "Context is up to date",
		}, nil
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
		Req:             req,
	}

	var hasConflicts bool
	var msg string
	if doUpdate {
		res, e := UpdateContext(UpdateContextParams{
			Contexts:    contexts,
			OutdatedRes: outdatedRes,
			Req:         req,
		})
		if e != nil {
			return nil, fmt.Errorf("failed to update context: %v", e)
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
		_, e := buildPlanInlineFn(false, nil)
		term.StopSpinner()
		fmt.Println()
		if e != nil {
			return nil, fmt.Errorf("failed to build plan: %v", e)
		}
	}

	// Warn about any items skipped
	if len(filesSkippedTooLarge) > 0 || len(filesSkippedAfterSizeLimit) > 0 ||
		len(mapFilesSkippedTooLarge) > 0 || len(mapFilesSkippedAfterSizeLimit) > 0 {
		printSkippedFilesMsg(filesSkippedTooLarge, filesSkippedAfterSizeLimit,
			mapFilesSkippedTooLarge, mapFilesSkippedAfterSizeLimit)
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
