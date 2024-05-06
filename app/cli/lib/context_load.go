package lib

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"plandex/api"
	"plandex/fs"
	"plandex/term"
	"plandex/types"
	"plandex/url"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/plandex/plandex/shared"
)

func MustLoadContext(resources []string, params *types.LoadContextParams) {
	term.StartSpinner("📥 Loading context...")

	onErr := func(err error) {
		term.StopSpinner()
		term.OutputErrorAndExit("Failed to load context: %v", err)
	}

	var loadContextReq shared.LoadContextRequest

	if params.Note != "" {
		loadContextReq = append(loadContextReq, &shared.LoadContextParams{
			ContextType: shared.ContextNoteType,
			Body:        params.Note,
		})
	}
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		onErr(fmt.Errorf("failed to stat stdin: %v", err))
	}
	if fileInfo.Mode()&os.ModeNamedPipe != 0 {
		reader := bufio.NewReader(os.Stdin)
		pipedData, err := io.ReadAll(reader)
		if err != nil {
			onErr(fmt.Errorf("failed to read piped data: %v", err))
		}

		if len(pipedData) > 0 {
			apiKeys := MustVerifyApiKeysSilent()
			openAIBase := os.Getenv("OPENAI_API_BASE")
			if openAIBase == "" {
				openAIBase = os.Getenv("OPENAI_ENDPOINT")
			}

			loadContextReq = append(loadContextReq, &shared.LoadContextParams{
				ContextType: shared.ContextPipedDataType,
				Body:        string(pipedData),
				ApiKeys:     apiKeys,
				OpenAIBase:  openAIBase,
				OpenAIOrgId: os.Getenv("OPENAI_ORG_ID"),
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
				inputFilePaths = append(inputFilePaths, resource)
			}
		}
	}

	var contextMu sync.Mutex

	errCh := make(chan error)
	ignoredPaths := make(map[string]string)

	numRoutines := 0

	// filter out already loaded contexts
	alreadyLoadedByComposite := make(map[string]string)
	existingContexts, apiErr := api.Client.ListContext(CurrentPlanId, CurrentBranch)
	if apiErr != nil {
		onErr(fmt.Errorf("failed to list contexts: %v", apiErr.Msg))
	}

	existsByComposite := make(map[string]bool)
	for _, context := range existingContexts {
		switch context.ContextType {
		case shared.ContextFileType, shared.ContextDirectoryTreeType:
			existsByComposite[strings.Join([]string{string(context.ContextType), context.FilePath}, "|")] = true
		case shared.ContextURLType:
			existsByComposite[strings.Join([]string{string(context.ContextType), context.Url}, "|")] = true
		}
	}

	if len(inputFilePaths) > 0 {
		baseDir := fs.GetBaseDirForFilePaths(inputFilePaths)

		paths, err := fs.GetProjectPaths(baseDir)
		if err != nil {
			onErr(fmt.Errorf("failed to get project paths: %v", err))
		}

		// log.Println(spew.Sdump(paths))

		// fmt.Println("active paths", len(paths.ActivePaths))
		// fmt.Println("all paths", len(paths.AllPaths))
		// fmt.Println("ignored paths", len(paths.IgnoredPaths))

		// spew.Dump(paths.IgnoredPaths)
		// spew.Dump(paths.ActivePaths)

		if !params.ForceSkipIgnore {
			var filteredPaths []string
			for _, inputFilePath := range inputFilePaths {
				// log.Println("inputFilePath", inputFilePath)

				if _, ok := paths.ActivePaths[inputFilePath]; !ok {
					// log.Println("not active", inputFilePath)

					if _, ok := paths.IgnoredPaths[inputFilePath]; ok {
						// log.Println("ignored", inputFilePath)

						ignoredPaths[inputFilePath] = paths.IgnoredPaths[inputFilePath]
					}
				} else {
					// log.Println("active", inputFilePath)

					filteredPaths = append(filteredPaths, inputFilePath)
				}
			}
			inputFilePaths = filteredPaths
		}

		if params.NamesOnly {
			for _, inputFilePath := range inputFilePaths {
				composite := strings.Join([]string{string(shared.ContextDirectoryTreeType), inputFilePath}, "|")
				if existsByComposite[composite] {
					alreadyLoadedByComposite[composite] = inputFilePath
					continue
				}

				numRoutines++
				go func(inputFilePath string) {
					flattenedPaths, err := ParseInputPaths([]string{inputFilePath}, params)
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
								if _, ok := paths.IgnoredPaths[path]; ok {
									ignoredPaths[path] = paths.IgnoredPaths[path]
								}
							}
						}
						flattenedPaths = filteredPaths
					}

					body := strings.Join(flattenedPaths, "\n")

					name := inputFilePath
					if name == "." {
						name = "cwd"
					}
					if name == ".." {
						name = "parent"
					}

					contextMu.Lock()
					defer contextMu.Unlock()
					loadContextReq = append(loadContextReq, &shared.LoadContextParams{
						ContextType:     shared.ContextDirectoryTreeType,
						Name:            name,
						Body:            body,
						FilePath:        inputFilePath,
						ForceSkipIgnore: params.ForceSkipIgnore,
					})

					errCh <- nil
				}(inputFilePath)
			}

		} else {
			flattenedPaths, err := ParseInputPaths(inputFilePaths, params)
			if err != nil {
				onErr(fmt.Errorf("failed to parse input paths: %v", err))
			}

			if !params.ForceSkipIgnore {
				var filteredPaths []string
				for _, path := range flattenedPaths {
					if _, ok := paths.ActivePaths[path]; ok {
						filteredPaths = append(filteredPaths, path)
					} else {
						if _, ok := paths.IgnoredPaths[path]; ok {
							ignoredPaths[path] = paths.IgnoredPaths[path]
						}
					}
				}
				flattenedPaths = filteredPaths
			}

			inputFilePaths = flattenedPaths

			for _, path := range flattenedPaths {
				composite := strings.Join([]string{string(shared.ContextFileType), path}, "|")

				if existsByComposite[composite] {
					alreadyLoadedByComposite[composite] = path
					continue
				}

				numRoutines++
				go func(path string) {
					fileContent, err := os.ReadFile(path)
					if err != nil {
						errCh <- fmt.Errorf("failed to read the file %s: %v", path, err)
						return
					}
					body := string(fileContent)

					contextMu.Lock()
					defer contextMu.Unlock()

					loadContextReq = append(loadContextReq, &shared.LoadContextParams{
						ContextType: shared.ContextFileType,
						Name:        path,
						Body:        body,
						FilePath:    path,
					})

					errCh <- nil
				}(path)
			}
		}
	}

	if len(inputUrls) > 0 {
		for _, u := range inputUrls {
			composite := strings.Join([]string{string(shared.ContextURLType), u}, "|")
			if existsByComposite[composite] {
				alreadyLoadedByComposite[composite] = u
				continue
			}

			numRoutines++
			go func(u string) {
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

				contextMu.Lock()
				defer contextMu.Unlock()

				loadContextReq = append(loadContextReq, &shared.LoadContextParams{
					ContextType: shared.ContextURLType,
					Name:        name,
					Body:        body,
					Url:         u,
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

	if len(loadContextReq) == 0 {
		term.StopSpinner()
		fmt.Println("🤷‍♂️ No context loaded")
		if len(alreadyLoadedByComposite) > 0 {
			printAlreadyLoadedMsg(alreadyLoadedByComposite)
		}
		if len(ignoredPaths) > 0 {
			printIgnoredMsg()
		}
		os.Exit(0)
	}

	res, apiErr := api.Client.LoadContext(CurrentPlanId, CurrentBranch, loadContextReq)

	if apiErr != nil {
		onErr(fmt.Errorf("failed to load context: %v", apiErr.Msg))
	}

	term.StopSpinner()

	if res.MaxTokensExceeded {
		overage := res.TotalTokens - res.MaxTokens
		term.OutputErrorAndExit("Update would add %d 🪙 and exceed token limit (%d) by %d 🪙\n", res.TokensAdded, res.MaxTokens, overage)
	}

	if hasConflicts {
		term.StartSpinner("🏗️  Starting build...")
		_, err := buildPlanInlineFn(nil)

		if err != nil {
			onErr(fmt.Errorf("failed to build plan: %v", err))
		}

		fmt.Println()
	}

	fmt.Println("✅ " + res.Msg)

	if len(alreadyLoadedByComposite) > 0 {
		printAlreadyLoadedMsg(alreadyLoadedByComposite)
	}

	if len(ignoredPaths) > 0 {
		printIgnoredMsg()
	}
}

func printAlreadyLoadedMsg(alreadyLoadedByComposite map[string]string) {
	fmt.Println()
	pronoun := "they're"
	if len(alreadyLoadedByComposite) == 1 {
		pronoun = "it's"
	}
	fmt.Printf("🙅‍♂️ Skipped because %s already in context:\n", pronoun)
	for composite, name := range alreadyLoadedByComposite {
		parts := strings.Split(composite, "|")
		contextType := parts[0]

		_, icon := GetContextLabelAndIcon(shared.ContextType(contextType))

		fmt.Printf("  • %s %s\n", icon, name)
	}
}

func printIgnoredMsg() {
	fmt.Println()
	fmt.Println("ℹ️  " + color.New(color.FgWhite).Sprint("Due to .gitignore or .plandexignore, some paths weren't loaded.\nUse --force / -f to load ignored paths."))
}
