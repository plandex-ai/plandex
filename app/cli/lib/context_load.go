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

	"github.com/plandex/plandex/shared"
	ignore "github.com/sabhiram/go-gitignore"
)

func MustLoadContext(resources []string, params *types.LoadContextParams) {
	term.StartSpinner("📥 Loading context...")

	onErr := func(err error) {
		term.StopSpinner()
		fmt.Fprintf(os.Stderr, "Failed to load context: %v\n", err)
		os.Exit(1)
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
			loadContextReq = append(loadContextReq, &shared.LoadContextParams{
				ContextType: shared.ContextPipedDataType,
				Body:        string(pipedData),
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

	contextCh := make(chan *shared.LoadContextParams)
	errCh := make(chan error)

	ignoredPaths := make(map[string]string)

	if len(inputFilePaths) > 0 {

		projectPaths := make(map[string]bool)
		var plandexIgnored *ignore.GitIgnore

		if !params.ForceSkipIgnore {
			projectPaths, plandexIgnored, err = fs.GetProjectPaths()
			if err != nil {
				onErr(fmt.Errorf("failed to get project paths: %v", err))
			}
			var filteredPaths []string
			for _, inputFilePath := range inputFilePaths {
				if _, ok := projectPaths[inputFilePath]; !ok {
					if plandexIgnored.MatchesPath(inputFilePath) {
						ignoredPaths[inputFilePath] = "plandex"
					} else {
						ignoredPaths[inputFilePath] = "git"
					}
				} else {
					filteredPaths = append(filteredPaths, inputFilePath)
				}
			}
			inputFilePaths = filteredPaths
		}

		if params.NamesOnly {
			for _, inputFilePath := range inputFilePaths {

				go func(inputFilePath string) {
					flattenedPaths, err := ParseInputPaths([]string{inputFilePath}, params)
					if err != nil {
						errCh <- fmt.Errorf("failed to parse input paths: %v", err)
						return
					}

					if !params.ForceSkipIgnore {
						var filteredPaths []string
						for _, path := range flattenedPaths {
							if _, ok := projectPaths[path]; ok {
								filteredPaths = append(filteredPaths, path)
							} else if plandexIgnored != nil && plandexIgnored.MatchesPath(path) {
								ignoredPaths[path] = "plandex"
							} else {
								ignoredPaths[path] = "git"
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

					contextCh <- &shared.LoadContextParams{
						ContextType: shared.ContextDirectoryTreeType,
						Name:        name,
						Body:        body,
						FilePath:    inputFilePath,
					}
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
					if _, ok := projectPaths[path]; ok {
						filteredPaths = append(filteredPaths, path)
					} else if plandexIgnored != nil && plandexIgnored.MatchesPath(path) {
						ignoredPaths[path] = "plandex"
					} else {
						ignoredPaths[path] = "git"
					}
				}
				flattenedPaths = filteredPaths
			}

			inputFilePaths = flattenedPaths

			for _, path := range flattenedPaths {

				go func(path string) {
					fileContent, err := os.ReadFile(path)
					if err != nil {
						errCh <- fmt.Errorf("failed to read the file %s: %v", path, err)
						return
					}
					body := string(fileContent)

					contextCh <- &shared.LoadContextParams{
						ContextType: shared.ContextFileType,
						Name:        path,
						Body:        body,
						FilePath:    path,
					}
				}(path)
			}
		}
	}

	if len(inputUrls) > 0 {
		for _, u := range inputUrls {
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

				contextCh <- &shared.LoadContextParams{
					ContextType: shared.ContextURLType,
					Name:        name,
					Body:        body,
					Url:         u,
				}
			}(u)
		}
	}

	for i := 0; i < len(inputFilePaths)+len(inputUrls); i++ {
		select {
		case err := <-errCh:
			onErr(err)
		case context := <-contextCh:
			loadContextReq = append(loadContextReq, context)
		}
	}

	if len(loadContextReq) == 0 {
		term.StopSpinner()
		fmt.Println("🤷‍♂️ No context loaded")
		os.Exit(0)
	}

	res, apiErr := api.Client.LoadContext(CurrentPlanId, CurrentBranch, loadContextReq)

	if apiErr != nil {
		onErr(fmt.Errorf("failed to load context: %v", apiErr.Msg))
	}

	term.StopSpinner()

	if res.MaxTokensExceeded {
		overage := res.TotalTokens - shared.MaxContextTokens
		fmt.Printf("🚨 Update would add %d 🪙 and exceed token limit (%d) by %d 🪙\n", res.TokensAdded, shared.MaxContextTokens, overage)
		os.Exit(1)
	}

	fmt.Println("✅ " + res.Msg)

	if len(ignoredPaths) > 0 {
		fmt.Println()
		fmt.Println("ℹ️ Due to .gitignore or .plandexignore, some paths weren't loaded. Use --force / -f to load ignored paths.")

	}
}
