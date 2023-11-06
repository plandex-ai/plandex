package lib

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"plandex/format"
	"plandex/types"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/plandex/plandex/shared"
)

func MustLoadContext(resources []string, params *types.LoadContextParams) (int, int) {
	timeStart := time.Now()

	s := spinner.New(spinner.CharSets[33], 100*time.Millisecond)
	s.Prefix = "📥 Loading context... "
	s.Start()

	maxTokens := shared.MaxContextTokens

	planState, err := GetPlanState()
	if err != nil {
		s.Stop()
		ClearCurrentLine()
		log.Fatalf("Failed to get plan state: %v", err)
	}

	tokensAdded := 0
	totalTokens := planState.ContextTokens
	totalUpdatableTokens := planState.ContextUpdatableTokens
	var totalTokensMutex sync.Mutex

	var contextParts []*shared.ModelContextPart
	var contextPartsMutex sync.Mutex

	wg := sync.WaitGroup{}

	if params.Note != "" {
		wg.Add(1)

		go func() {
			defer wg.Done()

			body := params.Note
			numTokens := shared.GetNumTokens(body)

			totalTokensMutex.Lock()
			func() {
				defer totalTokensMutex.Unlock()

				totalTokens += numTokens
				tokensAdded += numTokens

				if totalTokens > maxTokens {
					s.Stop()
					ClearCurrentLine()
					log.Fatalf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, maxTokens)
				}
			}()

			hash := sha256.Sum256([]byte(body))
			sha := hex.EncodeToString(hash[:])

			fileNameResp, err := Api.FileName(body)
			if err != nil {
				s.Stop()
				ClearCurrentLine()
				log.Fatalf("Failed to get a file name for the text: %v", err)
			}

			fileName := format.GetFileNameWithoutExt(fileNameResp.FileName)

			ts := shared.StringTs()
			contextPart := &shared.ModelContextPart{
				Type:      shared.ContextNoteType,
				Name:      fileName,
				Body:      body,
				Sha:       sha,
				NumTokens: numTokens,
				AddedAt:   ts,
				UpdatedAt: ts,
			}

			contextPartsMutex.Lock()
			contextParts = append(contextParts, contextPart)
			contextPartsMutex.Unlock()

		}()

	}

	hasPipeData := false
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		s.Stop()
		ClearCurrentLine()
		log.Fatalf("Failed to stat stdin: %v", err)
	}
	if fileInfo.Mode()&os.ModeNamedPipe != 0 {
		reader := bufio.NewReader(os.Stdin)
		pipedData, err := io.ReadAll(reader)
		if err != nil {
			s.Stop()
			ClearCurrentLine()
			log.Fatalf("Failed to read piped data: %v", err)
		}

		if len(pipedData) > 0 {
			wg.Add(1)

			hasPipeData = true

			go func() {
				defer wg.Done()

				body := string(pipedData)
				numTokens := shared.GetNumTokens(body)

				totalTokensMutex.Lock()
				func() {
					defer totalTokensMutex.Unlock()

					totalTokens += numTokens
					tokensAdded += numTokens
					if totalTokens > maxTokens {
						s.Stop()
						ClearCurrentLine()
						log.Fatalf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, maxTokens)
					}
				}()

				hash := sha256.Sum256([]byte(body))
				sha := hex.EncodeToString(hash[:])

				fileNameResp, err := Api.FileName(body)
				if err != nil {
					s.Stop()
					ClearCurrentLine()
					log.Fatalf("Failed to get a file name for piped data: %v", err)
				}

				fileName := format.GetFileNameWithoutExt(fileNameResp.FileName)

				ts := shared.StringTs()
				contextPart := &shared.ModelContextPart{
					Type:      shared.ContextPipedDataType,
					Name:      fileName,
					Body:      body,
					Sha:       sha,
					NumTokens: numTokens,
					AddedAt:   ts,
					UpdatedAt: ts,
				}

				contextPartsMutex.Lock()
				contextParts = append(contextParts, contextPart)
				contextPartsMutex.Unlock()

			}()
		}
	}

	var inputUrls []string
	var inputFilePaths []string

	if len(resources) > 0 {
		for _, resource := range resources {
			// so far resources are either files or urls
			if IsValidURL(resource) {
				inputUrls = append(inputUrls, resource)
			} else {
				inputFilePaths = append(inputFilePaths, resource)
			}
		}
	}

	if len(inputFilePaths) > 0 {
		if params.NamesOnly {
			for _, inputFilePath := range inputFilePaths {
				wg.Add(1)

				go func(inputFilePath string) {
					defer wg.Done()

					flattenedPaths, err := ParseInputPaths([]string{inputFilePath}, params)
					if err != nil {
						s.Stop()
						ClearCurrentLine()
						log.Fatalf("Failed to parse input paths: %v", err)
					}

					body := strings.Join(flattenedPaths, "\n")
					bytes := []byte(body)

					hash := sha256.Sum256(bytes)
					sha := hex.EncodeToString(hash[:])
					numTokens := shared.GetNumTokens(body)

					totalTokensMutex.Lock()
					func() {
						defer totalTokensMutex.Unlock()
						totalTokens += numTokens
						totalUpdatableTokens += numTokens
						tokensAdded += numTokens
						if totalTokens > maxTokens {
							s.Stop()
							ClearCurrentLine()
							log.Fatalf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, maxTokens)
						}

					}()

					ts := shared.StringTs()

					// get last portion of directory path
					name := filepath.Base(inputFilePath)
					if name == "." {
						name = "cwd"
					}
					if name == ".." {
						name = "parent"
					}

					contextPart := &shared.ModelContextPart{
						Type:      shared.ContextDirectoryTreeType,
						Name:      name,
						FilePath:  inputFilePath,
						Body:      body,
						Sha:       sha,
						NumTokens: numTokens,
						AddedAt:   ts,
						UpdatedAt: ts,
					}

					contextPartsMutex.Lock()
					contextParts = append(contextParts, contextPart)
					contextPartsMutex.Unlock()

				}(inputFilePath)
			}

		} else {
			flattenedPaths, err := ParseInputPaths(inputFilePaths, params)
			if err != nil {
				s.Stop()
				ClearCurrentLine()
				log.Fatalf("Failed to parse input paths: %v", err)
			}

			for _, path := range flattenedPaths {
				wg.Add(1)

				go func(path string) {
					defer wg.Done()

					fileContent, err := os.ReadFile(path)
					if err != nil {
						s.Stop()
						ClearCurrentLine()
						log.Fatalf("Failed to read the file %s: %v", path, err)
					}

					body := string(fileContent)
					hash := sha256.Sum256(fileContent)
					sha := hex.EncodeToString(hash[:])
					numTokens := shared.GetNumTokens(body)

					totalTokensMutex.Lock()
					func() {
						defer totalTokensMutex.Unlock()
						totalTokens += numTokens
						tokensAdded += numTokens
						totalUpdatableTokens += numTokens
						if totalTokens > maxTokens {
							s.Stop()
							ClearCurrentLine()
							log.Fatalf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, maxTokens)
						}

					}()

					_, fileName := filepath.Split(path)

					ts := shared.StringTs()

					contextPart := &shared.ModelContextPart{
						Type:      shared.ContextFileType,
						Name:      fileName,
						Body:      body,
						FilePath:  path,
						Sha:       sha,
						NumTokens: numTokens,
						AddedAt:   ts,
						UpdatedAt: ts,
					}

					contextPartsMutex.Lock()
					contextParts = append(contextParts, contextPart)
					contextPartsMutex.Unlock()

				}(path)

			}
		}

	}

	if len(inputUrls) > 0 {
		for _, url := range inputUrls {
			wg.Add(1)

			go func(url string) {
				defer wg.Done()

				body, err := FetchURLContent(url)
				if err != nil {
					s.Stop()
					ClearCurrentLine()
					log.Fatalf("Failed to fetch content from URL %s: %v", url, err)
				}

				numTokens := shared.GetNumTokens(body)

				totalTokensMutex.Lock()
				func() {
					defer totalTokensMutex.Unlock()
					totalTokens += numTokens
					tokensAdded += numTokens
					totalUpdatableTokens += numTokens
					if totalTokens > maxTokens {
						s.Stop()
						ClearCurrentLine()
						log.Fatalf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, maxTokens)
					}
				}()

				hash := sha256.Sum256([]byte(body))
				sha := hex.EncodeToString(hash[:])

				ts := shared.StringTs()
				contextPart := &shared.ModelContextPart{
					Type:      shared.ContextURLType,
					Name:      SanitizeAndClipURL(url, 70),
					Url:       url,
					Body:      body,
					Sha:       sha,
					NumTokens: numTokens,
					AddedAt:   ts,
					UpdatedAt: ts,
				}

				contextPartsMutex.Lock()
				contextParts = append(contextParts, contextPart)
				contextPartsMutex.Unlock()
			}(url)
		}
	}

	wg.Wait()

	if len(contextParts) == 0 {
		fmt.Println("🤷‍♂️ No context loaded")
		os.Exit(1)
	}

	errCh := make(chan error, 2)
	go func() {
		errCh <- writeContextParts(contextParts)
	}()

	go func() {
		planState.ContextTokens = totalTokens
		planState.ContextUpdatableTokens = totalUpdatableTokens
		errCh <- SetPlanState(planState, shared.StringTs())
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			log.Fatal(err)
		}
	}

	var added []string
	if params.Note != "" {
		added = append(added, "a note")
	}
	if hasPipeData {
		added = append(added, "piped data")
	}
	if len(inputFilePaths) > 0 {
		var label string
		if params.NamesOnly {
			label = "directory tree"
			if len(inputFilePaths) > 1 {
				label = "directory trees"
			}
		} else {
			label = "file"
			if len(inputFilePaths) > 1 {
				label = "files"
			}
		}

		added = append(added, fmt.Sprintf("%d %s", len(inputFilePaths), label))
	}
	if len(inputUrls) > 0 {
		label := "url"
		if len(inputUrls) > 1 {
			label = "urls"
		}
		added = append(added, fmt.Sprintf("%d %s", len(inputUrls), label))
	}

	msg := "Loaded "
	if len(added) <= 2 {
		msg += strings.Join(added, " and ")
	} else {
		for i, add := range added {
			if i == len(added)-1 {
				msg += ", and " + add
			} else {
				msg += ", " + add
			}
		}
	}
	msg += fmt.Sprintf(" into context | added → %d 🪙 |  total → %d 🪙", tokensAdded, totalTokens)

	if err != nil {
		s.Stop()
		ClearCurrentLine()
		log.Fatalf("Failed to get total tokens: %v", err)
	}

	err = GitCommitContextUpdate(msg)
	if err != nil {
		s.Stop()
		ClearCurrentLine()
		log.Fatalf("Failed to commit context update to git: %v", err)
	}

	elapsed := time.Since(timeStart)
	if elapsed < 700*time.Millisecond {
		time.Sleep(700*time.Millisecond - elapsed)
	}

	s.Stop()
	ClearCurrentLine()
	fmt.Println("✅ " + msg)

	return tokensAdded, totalTokens
}
