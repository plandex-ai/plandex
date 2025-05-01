package lib

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/fs"
	"plandex-cli/term"
	"plandex-cli/types"
	"plandex-cli/url"
	"strings"
	"sync"

	shared "plandex-shared"

	"github.com/fatih/color"
)

const maxSkippedFileList = 20

func MustLoadContext(resources []string, params *types.LoadContextParams) {
	if params.DefsOnly {
		// while caching is set up to work with multiple map paths, it can end up in a partially loaded state if token limits are exceeded, so better to just load one at a time
		if len(resources) > 1 {
			term.OutputErrorAndExit("Please load a single map directory at a time")
		}

		term.LongSpinnerWithWarning("🗺️  Building project map...", "🗺️  This can take a while in larger projects...")
	} else if params.NamesOnly {
		term.LongSpinnerWithWarning("🌳 Loading directory tree...", "🌳 This can take a while in larger projects...")
	} else {
		term.StartSpinner("📥 Loading context...")
	}

	onErr := func(err error) {
		term.StopSpinner()
		term.OutputErrorAndExit("Failed to load context: %v", err)
	}

	var loadContextReq shared.LoadContextRequest

	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		onErr(fmt.Errorf("failed to stat stdin: %v", err))
	}

	var apiKeys map[string]string
	var openAIBase string

	if !auth.Current.IntegratedModelsMode {
		if params.Note != "" || fileInfo.Mode()&os.ModeNamedPipe != 0 {
			apiKeys = MustVerifyApiKeysSilent()
			openAIBase = os.Getenv("OPENAI_API_BASE")
			if openAIBase == "" {
				openAIBase = os.Getenv("OPENAI_ENDPOINT")
			}
		}
	}

	if params.Note != "" {
		loadContextReq = append(loadContextReq, &shared.LoadContextParams{
			ContextType: shared.ContextNoteType,
			Body:        params.Note,
			ApiKeys:     apiKeys,
			OpenAIBase:  openAIBase,
			OpenAIOrgId: os.Getenv("OPENAI_ORG_ID"),
			SessionId:   params.SessionId,
			AutoLoaded:  params.AutoLoaded,
		})
	}

	if fileInfo.Mode()&os.ModeNamedPipe != 0 {
		reader := bufio.NewReader(os.Stdin)
		pipedData, err := io.ReadAll(reader)
		if err != nil {
			onErr(fmt.Errorf("failed to read piped data: %v", err))
		}

		if len(pipedData) > 0 {
			loadContextReq = append(loadContextReq, &shared.LoadContextParams{
				ContextType: shared.ContextPipedDataType,
				Body:        string(pipedData),
				ApiKeys:     apiKeys,
				OpenAIBase:  openAIBase,
				OpenAIOrgId: os.Getenv("OPENAI_ORG_ID"),
				SessionId:   params.SessionId,
				AutoLoaded:  params.AutoLoaded,
			})
		}
	}

	var inputUrls []string
	var inputFilePaths []string

	if len(resources) > 0 {
		for _, resource := range resources {
			// so far resources are either files or urls
			if url.IsValidURL(resource) {
				inputUrls = append(inputUrls, resource)
			} else {
				if strings.HasPrefix(resource, "."+string(os.PathSeparator)) {
					resource = resource[2:]
				}

				inputFilePaths = append(inputFilePaths, resource)
			}
		}
	}

	var contextMu sync.Mutex

	errCh := make(chan error)
	ignoredPaths := make(map[string]string)

	mapFilesTruncatedTooLarge := []filePathWithSize{}
	mapFilesSkippedAfterSizeLimit := []string{}

	// We'll reuse these for all skipping, including directory-tree partial skipping and URLs
	filesSkippedTooLarge := []filePathWithSize{}
	filesSkippedAfterSizeLimit := []string{}

	var totalSize int64

	numRoutines := 0

	// filter out already loaded contexts
	alreadyLoadedByComposite := make(map[string]*shared.Context)
	existingContexts, apiErr := api.Client.ListContext(CurrentPlanId, CurrentBranch)
	if apiErr != nil {
		onErr(fmt.Errorf("failed to list contexts: %v", apiErr.Msg))
	}

	existsByComposite := make(map[string]*shared.Context)
	for _, context := range existingContexts {
		switch context.ContextType {
		case shared.ContextFileType, shared.ContextDirectoryTreeType, shared.ContextMapType, shared.ContextImageType:
			existsByComposite[strings.Join([]string{string(context.ContextType), context.FilePath}, "|")] = context
		case shared.ContextURLType:
			existsByComposite[strings.Join([]string{string(context.ContextType), context.Url}, "|")] = context
		}
	}

	var cachedMapPaths map[string]bool
	var cachedMapLoadRes *shared.LoadContextResponse

	mapInputShas := map[string]string{}
	mapInputTokens := map[string]int{}
	mapInputSizes := map[string]int64{}

	toLoadMapPaths := []string{}
	mapInputPathsForPaths := map[string]string{}

	currentMapInputBatch := shared.FileMapInputs{}
	mapInputBatches := []shared.FileMapInputs{currentMapInputBatch}

	sem := make(chan struct{}, ContextMapMaxClientConcurrency)

	if len(inputFilePaths) > 0 {

		var mapSize int64

		if params.DefsOnly {
			for _, inputFilePath := range inputFilePaths {
				composite := strings.Join([]string{string(shared.ContextMapType), inputFilePath}, "|")
				if existsByComposite[composite] != nil {
					alreadyLoadedByComposite[composite] = existsByComposite[composite]
					continue
				}

				toLoadMapPaths = append(toLoadMapPaths, inputFilePath)
			}

			var uncachedMapPaths []string

			res, err := api.Client.LoadCachedFileMap(CurrentPlanId, CurrentBranch, shared.LoadCachedFileMapRequest{
				FilePaths: toLoadMapPaths,
			})

			if err != nil {
				onErr(fmt.Errorf("error checking cached file map: %v", err))
			}

			if res.LoadRes != nil {
				if res.LoadRes.MaxTokensExceeded {
					term.StopSpinner()
					overage := res.LoadRes.TotalTokens - res.LoadRes.MaxTokens

					term.OutputErrorAndExit("Update would add %d 🪙 and exceed token limit (%d) by %d 🪙\n", res.LoadRes.TokensAdded, res.LoadRes.MaxTokens, overage)
				}

				cachedMapLoadRes = res.LoadRes
				cachedMapPaths = res.CachedByPath

				for _, path := range toLoadMapPaths {
					if !cachedMapPaths[path] {
						uncachedMapPaths = append(uncachedMapPaths, path)
					}
				}
			} else {
				uncachedMapPaths = toLoadMapPaths
			}

			toLoadMapPaths = uncachedMapPaths
			inputFilePaths = toLoadMapPaths
		}

		if len(inputFilePaths) > 0 {
			baseDir := fs.GetBaseDirForFilePaths(inputFilePaths)

			paths, err := fs.GetProjectPaths(baseDir)
			if err != nil {
				onErr(fmt.Errorf("failed to get project paths: %v", err))
			}

			if !params.ForceSkipIgnore {
				var filteredPaths []string
				for _, inputFilePath := range inputFilePaths {
					if _, ok := paths.ActivePaths[inputFilePath]; !ok {
						ignored, reason, err := fs.IsIgnored(paths, inputFilePath, baseDir)
						if err != nil {
							onErr(fmt.Errorf("failed to check if %s is ignored: %v", inputFilePath, err))
						}
						if ignored {
							ignoredPaths[inputFilePath] = reason
						}
					} else {
						filteredPaths = append(filteredPaths, inputFilePath)
					}
				}
				inputFilePaths = filteredPaths

			}

			if params.NamesOnly {
				// "params.NamesOnly" => we create directory-tree contexts (ContextDirectoryTreeType)
				// Partial skipping of subpaths
				for _, inputFilePath := range inputFilePaths {
					composite := strings.Join([]string{string(shared.ContextDirectoryTreeType), inputFilePath}, "|")
					if existsByComposite[composite] != nil {
						alreadyLoadedByComposite[composite] = existsByComposite[composite]
						continue
					}

					numRoutines++
					go func(inputFilePath string) {
						sem <- struct{}{}
						defer func() { <-sem }()

						flattenedPaths, err := ParseInputPaths(ParseInputPathsParams{
							FileOrDirPaths: []string{inputFilePath},
							BaseDir:        baseDir,
							ProjectPaths:   paths,
							LoadParams:     params,
						})
						if err != nil {
							errCh <- fmt.Errorf("failed to parse input paths: %v", err)
							return
						}

						if !params.ForceSkipIgnore {
							var filteredPaths []string
							for _, path := range flattenedPaths {
								if _, ok := paths.ActivePaths[path]; ok {
									filteredPaths = append(filteredPaths, path)
								} else {
									ignored, reason, err := fs.IsIgnored(paths, path, baseDir)
									if err != nil {
										errCh <- fmt.Errorf("failed to check if %s is ignored: %v", path, err)
										return
									}
									if ignored {
										ignoredPaths[path] = reason
									}
								}
							}
							flattenedPaths = filteredPaths
						}

						// PARTIAL skipping of subpaths
						var keptPaths []string
						for _, p := range flattenedPaths {
							lineSize := int64(len(p))

							contextMu.Lock()
							if lineSize > shared.MaxContextBodySize {
								filesSkippedTooLarge = append(filesSkippedTooLarge, filePathWithSize{Path: p, Size: lineSize})
								contextMu.Unlock()
								continue
							}

							if totalSize+lineSize > shared.MaxContextBodySize {
								filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, p)
								contextMu.Unlock()
								continue
							}

							totalSize += lineSize
							contextMu.Unlock()

							keptPaths = append(keptPaths, p)
						}

						body := strings.Join(keptPaths, "\n")

						name := inputFilePath
						if name == "." {
							name = "cwd"
						}
						if name == ".." {
							name = "parent"
						}

						contextMu.Lock()
						loadContextReq = append(loadContextReq, &shared.LoadContextParams{
							ContextType:     shared.ContextDirectoryTreeType,
							Name:            name,
							Body:            body,
							FilePath:        inputFilePath,
							ForceSkipIgnore: params.ForceSkipIgnore,
							AutoLoaded:      params.AutoLoaded,
						})
						contextMu.Unlock()

						errCh <- nil
					}(inputFilePath)
				}

			} else {
				flattenedPaths, err := ParseInputPaths(ParseInputPathsParams{
					FileOrDirPaths: inputFilePaths,
					BaseDir:        baseDir,
					ProjectPaths:   paths,
					LoadParams:     params,
				})
				if err != nil {
					onErr(fmt.Errorf("failed to parse input paths: %v", err))
				}

				if !params.ForceSkipIgnore {
					var filteredPaths []string
					for _, path := range flattenedPaths {
						if _, ok := paths.ActivePaths[path]; ok {
							filteredPaths = append(filteredPaths, path)
						} else {
							ignored, reason, err := fs.IsIgnored(paths, path, baseDir)
							if err != nil {
								onErr(fmt.Errorf("failed to check if %s is ignored: %v", path, err))
							}
							if ignored {
								ignoredPaths[path] = reason
							}
						}
					}
					flattenedPaths = filteredPaths

				}

				var numPaths int
				if params.DefsOnly {
					filtered := []string{}
					for _, path := range flattenedPaths {
						if shared.HasFileMapSupport(path) {
							numPaths++

							if numPaths > shared.MaxContextMapPaths {
								mapFilesSkippedAfterSizeLimit = append(mapFilesSkippedAfterSizeLimit, path)
								continue
							}

							filtered = append(filtered, path)
						}
					}
					flattenedPaths = filtered
				} else if params.NamesOnly {
					filtered := []string{}
					for _, path := range flattenedPaths {
						numPaths++

						if numPaths > shared.MaxContextMapPaths {
							filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, path)
							continue
						}

						filtered = append(filtered, path)
					}
					flattenedPaths = filtered
				} else {
					filtered := []string{}
					for _, path := range flattenedPaths {
						numPaths++

						if numPaths > shared.MaxContextCount {
							filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, path)
							continue
						}

						filtered = append(filtered, path)
					}
					flattenedPaths = filtered
				}

				inputFilePaths = flattenedPaths

				for _, path := range flattenedPaths {
					var mapInputPath string
					if params.DefsOnly {
						for _, inputPath := range toLoadMapPaths {
							absPath, err := filepath.Abs(path)
							if err != nil {
								continue
							}
							absInputPath, err := filepath.Abs(inputPath)
							if err != nil {
								continue
							}
							if absPath == absInputPath ||
								strings.HasPrefix(absPath+string(os.PathSeparator), absInputPath+string(os.PathSeparator)) {
								mapInputPath = inputPath
								break
							}
						}

						if mapInputPath == "" {
							continue
						}

						mapInputPathsForPaths[path] = mapInputPath
					}

					var contextType shared.ContextType
					isImage := shared.IsImageFile(path)
					if isImage {
						contextType = shared.ContextImageType
					} else if params.DefsOnly {
						contextType = shared.ContextMapType
					} else {
						contextType = shared.ContextFileType
					}

					if !params.DefsOnly {
						composite := strings.Join([]string{string(contextType), path}, "|")
						if existsByComposite[composite] != nil {
							alreadyLoadedByComposite[composite] = existsByComposite[composite]
							continue
						}
					}

					numRoutines++

					go func(path string) {
						sem <- struct{}{}
						defer func() { <-sem }()

						var size int64

						fileInfo, err := os.Stat(path)
						if err != nil {
							errCh <- fmt.Errorf("failed to get file info for %s: %v", path, err)
							return
						}
						size = fileInfo.Size()

						if !params.DefsOnly && size > shared.MaxContextBodySize {
							contextMu.Lock()
							filesSkippedTooLarge = append(filesSkippedTooLarge, filePathWithSize{Path: path, Size: size})
							contextMu.Unlock()
							errCh <- nil
							return
						}

						if !params.DefsOnly {
							contextMu.Lock()
							if totalSize+size > shared.MaxContextBodySize {
								filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, path)
								contextMu.Unlock()
								errCh <- nil
								return
							}
							totalSize += size
							contextMu.Unlock()
						}

						if params.DefsOnly {
							res, err := getMapFileDetails(path, size, totalSize)
							if err != nil {
								errCh <- fmt.Errorf("failed to get map file details for %s: %v", path, err)
								return
							}

							contextMu.Lock()
							defer contextMu.Unlock()

							if currentMapInputBatch.NumFiles()+1 > shared.ContextMapMaxBatchSize || currentMapInputBatch.TotalSize()+size > shared.ContextMapMaxBatchBytes {
								currentMapInputBatch = shared.FileMapInputs{}
								mapInputBatches = append(mapInputBatches, currentMapInputBatch)
							}

							currentMapInputBatch[path] = res.mapContent
							mapSize += res.size
							mapInputShas[path] = res.shaVal
							mapInputTokens[path] = res.tokens
							mapInputSizes[path] = res.size

							if len(res.mapFilesTruncatedTooLarge) > 0 {
								mapFilesTruncatedTooLarge = append(mapFilesTruncatedTooLarge, res.mapFilesTruncatedTooLarge...)
							}

							if len(res.mapFilesSkippedAfterSizeLimit) > 0 {
								mapFilesSkippedAfterSizeLimit = append(mapFilesSkippedAfterSizeLimit, res.mapFilesSkippedAfterSizeLimit...)
							}

						} else if isImage {
							fileContent, err := os.ReadFile(path)
							if err != nil {
								errCh <- fmt.Errorf("failed to read the file %s: %v", path, err)
								return
							}

							contextMu.Lock()
							defer contextMu.Unlock()

							loadContextReq = append(loadContextReq, &shared.LoadContextParams{
								ContextType: shared.ContextImageType,
								Name:        path,
								Body:        base64.StdEncoding.EncodeToString(fileContent),
								FilePath:    path,
								ImageDetail: params.ImageDetail,
								AutoLoaded:  params.AutoLoaded,
							})
						} else {
							fileContent, err := os.ReadFile(path)
							if err != nil {
								errCh <- fmt.Errorf("failed to read the file %s: %v", path, err)
								return
							}
							fileContent = shared.NormalizeEOL(fileContent)

							contextMu.Lock()
							defer contextMu.Unlock()

							loadContextReq = append(loadContextReq, &shared.LoadContextParams{
								ContextType: shared.ContextFileType,
								Name:        path,
								Body:        string(fileContent),
								FilePath:    path,
								AutoLoaded:  params.AutoLoaded,
							})
						}

						errCh <- nil
					}(path)
				}
			}
		}
	}

	if len(inputUrls) > 0 {
		for _, u := range inputUrls {
			composite := strings.Join([]string{string(shared.ContextURLType), u}, "|")
			if existsByComposite[composite] != nil {
				alreadyLoadedByComposite[composite] = existsByComposite[composite]
				continue
			}

			numRoutines++
			go func(u string) {
				sem <- struct{}{}
				defer func() { <-sem }()

				body, err := url.FetchURLContent(u)
				if err != nil {
					errCh <- fmt.Errorf("failed to fetch content from URL %s: %v", u, err)
					return
				}

				name := url.SanitizeURL(u)
				// show the first 20 characters, then ellipsis then the last 20 characters of 'name'
				if len(name) > 40 {
					name = name[:20] + "⋯" + name[len(name)-20:]
				}

				// Check the size of the URL body, just like a file:
				size := int64(len(body))

				contextMu.Lock()
				defer contextMu.Unlock()

				if size > shared.MaxContextBodySize {
					filesSkippedTooLarge = append(filesSkippedTooLarge, filePathWithSize{Path: u, Size: size})
					errCh <- nil
					return
				}
				if totalSize+size > shared.MaxContextBodySize {
					filesSkippedAfterSizeLimit = append(filesSkippedAfterSizeLimit, u)
					errCh <- nil
					return
				}
				totalSize += size

				loadContextReq = append(loadContextReq, &shared.LoadContextParams{
					ContextType: shared.ContextURLType,
					Name:        name,
					Body:        body,
					Url:         u,
					AutoLoaded:  params.AutoLoaded,
				})

				errCh <- nil
			}(u)
		}
	}

	for i := 0; i < numRoutines; i++ {
		err := <-errCh
		if err != nil {
			onErr(err)
		}
	}

	if params.DefsOnly {
		allMapBodies, err := processMapBatches(mapInputBatches)
		if err != nil {
			onErr(fmt.Errorf("failed to process map batches: %v", err))
		}

		for _, inputPath := range toLoadMapPaths {
			var name string
			if inputPath == "." {
				name = "cwd"
			} else if inputPath == ".." {
				name = "parent"
			} else {
				name = inputPath
			}

			pathBodies := shared.FileMapBodies{}
			pathShas := map[string]string{}
			pathTokens := map[string]int{}
			pathSizes := map[string]int64{}
			for path, body := range allMapBodies {
				mapInputPath := mapInputPathsForPaths[path]
				if mapInputPath == inputPath {
					pathBodies[path] = body
					pathShas[path] = mapInputShas[path]
					pathTokens[path] = mapInputTokens[path]
					pathSizes[path] = mapInputSizes[path]
				}
			}

			// load the map even if it's empty (no paths)
			// it needs to exist so it can be updated later
			loadContextReq = append(loadContextReq, &shared.LoadContextParams{
				ContextType: shared.ContextMapType,
				Name:        name,
				MapBodies:   pathBodies,
				InputShas:   pathShas,
				InputTokens: pathTokens,
				InputSizes:  pathSizes,
				FilePath:    inputPath,
				AutoLoaded:  params.AutoLoaded,
			})

		}
	}

	filesToLoad := map[string]string{}
	for _, context := range loadContextReq {
		if context.ContextType == shared.ContextFileType {
			filesToLoad[context.FilePath] = context.Body
		}
	}

	hasConflicts, err := checkContextConflicts(filesToLoad)

	if err != nil {
		onErr(fmt.Errorf("failed to check context conflicts: %v", err))
	}

	if len(loadContextReq)+len(cachedMapPaths) == 0 {
		term.StopSpinner()
		fmt.Println("🤷‍♂️ No context loaded")

		didOutputReason := false
		if len(alreadyLoadedByComposite) > 0 {
			printAlreadyLoadedMsg(alreadyLoadedByComposite)
			didOutputReason = true
		}
		if len(ignoredPaths) > 0 && !params.SkipIgnoreWarning {
			printIgnoredMsg()
			didOutputReason = true
		}

		if !didOutputReason {
			fmt.Println()
			fmt.Printf("Use %s to load a file or URL:", color.New(color.BgCyan, color.FgHiWhite).Sprint(" plandex load [file-path|url] "))
			fmt.Println()
			fmt.Println("plandex load file.c file.h")
			fmt.Println("plandex load https://github.com/some-org/some-repo/README.md")

			fmt.Println()
			fmt.Printf("%s with the --recursive/-r flag:\n", color.New(color.Bold, term.ColorHiCyan).Sprint("Load a whole directory"))
			fmt.Println("plandex load app/src -r")

			fmt.Println()
			fmt.Printf("%s with the --tree flag:\n", color.New(color.Bold, term.ColorHiCyan).Sprint("Load a directory layout (file names only)"))

			fmt.Println()
			fmt.Printf("%s file paths are relative to the current directory\n", color.New(color.Bold, term.ColorHiYellow).Sprint("Note:"))

			fmt.Println()
			fmt.Printf("%s with the -n flag:\n", color.New(color.Bold, term.ColorHiCyan).Sprint("Load a note"))
			fmt.Println("plandex load -n 'Some note here'")

			fmt.Println()
			fmt.Printf("%s from any command:\n", color.New(color.Bold, term.ColorHiCyan).Sprint("Pipe data in"))
			fmt.Println("npm test | plandex load")
		}

		os.Exit(0)
	}

	var res *shared.LoadContextResponse
	if cachedMapLoadRes != nil {
		res = cachedMapLoadRes
	} else {
		res, apiErr = api.Client.LoadContext(CurrentPlanId, CurrentBranch, loadContextReq)
		if apiErr != nil {
			onErr(fmt.Errorf("failed to load context: %v", apiErr.Msg))
		}
	}

	term.StopSpinner()

	if hasConflicts {
		term.StartSpinner("🏗️  Starting build...")
		_, err := buildPlanInlineFn(false, nil)
		if err != nil {
			onErr(fmt.Errorf("failed to build plan: %v", err))
		}
		fmt.Println()
	}

	fmt.Println("✅ " + res.Msg)

	if len(alreadyLoadedByComposite) > 0 {
		printAlreadyLoadedMsg(alreadyLoadedByComposite)
	}

	if len(ignoredPaths) > 0 && !params.SkipIgnoreWarning {
		printIgnoredMsg()
	}

	if len(filesSkippedTooLarge) > 0 || len(filesSkippedAfterSizeLimit) > 0 ||
		len(mapFilesTruncatedTooLarge) > 0 || len(mapFilesSkippedAfterSizeLimit) > 0 {
		printSkippedFilesMsg(filesSkippedTooLarge, filesSkippedAfterSizeLimit,
			mapFilesTruncatedTooLarge, mapFilesSkippedAfterSizeLimit)
	}
}

func printAlreadyLoadedMsg(alreadyLoadedByComposite map[string]*shared.Context) {
	fmt.Println()
	pronoun := "they're"
	if len(alreadyLoadedByComposite) == 1 {
		pronoun = "it's"
	}
	fmt.Printf("🙅‍♂️ Skipped because %s already in context:\n", pronoun)
	for _, context := range alreadyLoadedByComposite {
		_, icon := context.TypeAndIcon()

		fmt.Printf("  • %s %s\n", icon, context.Name)
	}
}

func printIgnoredMsg() {
	fmt.Println()
	fmt.Println("ℹ️  " + color.New(color.FgWhite).Sprint("Due to .gitignore or .plandexignore, some paths weren't loaded.\nUse --force / -f to load ignored paths."))
}
