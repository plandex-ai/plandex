package lib

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"plandex/types"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/briandowns/spinner"
	"github.com/plandex/plandex/shared"
)

func LoadContextOrDie(params *types.LoadContextParams) (int, int) {
	timeStart := time.Now()

	s := spinner.New(spinner.CharSets[33], 100*time.Millisecond)
	s.Prefix = "📥 Loading context... "
	s.Start()
	var contextState shared.ModelContextState
	contextStateFilePath := filepath.Join(ContextSubdir, "context.json")

	// fmt.Fprintf(os.Stderr, "Loading context from %s\n", contextStateFilePath)

	var maxTokens int
	if params.MaxTokens == -1 {
		maxTokens = shared.MaxTokens
	} else {
		maxTokens = min(params.MaxTokens, shared.MaxTokens)
	}

	func() {
		contextStateFile, err := os.Open(contextStateFilePath)
		if err != nil {
			log.Fatalf("Error opening file: %v", err)
			return
		}

		defer contextStateFile.Close()

		data, err := io.ReadAll(contextStateFile)
		if err != nil {
			log.Fatalf("Error reading file: %v", err)
			return
		}

		err = json.Unmarshal(data, &contextState)
		if err != nil {
			log.Fatalf("Error unmarshalling JSON: %v", err)
			return
		}
	}()

	tokensAdded := 0
	totalTokens := contextState.NumTokens
	var totalTokensMutex sync.Mutex

	counter := contextState.Counter

	var contextParts []shared.ModelContextPart
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
					if params.Truncate {
						log.Printf("The total number of tokens (%d) exceeds the maximum allowed (%d). Truncating input text.", totalTokens, maxTokens)
						numTokens = maxTokens - (totalTokens - numTokens)

						// If the number of tokens is less than or equal to 0, then we can stop processing files
						if numTokens <= 0 {
							return
						}

						body = body[:numTokens]
						totalTokens = maxTokens

					} else {
						log.Fatalf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, maxTokens)
					}

				}
			}()

			hash := sha256.Sum256([]byte(body))
			sha := hex.EncodeToString(hash[:])

			fileNameResp, err := Api.FileName(body)
			if err != nil {
				log.Fatalf("Failed to summarize the text: %v", err)
			}

			fileName := GetFileNameWithoutExt(fileNameResp.FileName)

			contextPart := shared.ModelContextPart{
				Name:      fmt.Sprintf("%d.%s", atomic.LoadUint32(&counter), fileName),
				Body:      body,
				Sha:       sha,
				NumTokens: numTokens,
				UpdatedAt: StringTs(),
			}

			contextPartsMutex.Lock()
			contextParts = append(contextParts, contextPart)
			contextPartsMutex.Unlock()

			atomic.AddUint32(&counter, 1)
		}()

	}

	hasPipeData := false
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		log.Fatalf("Failed to stat stdin: %v", err)
	}
	if fileInfo.Mode()&os.ModeNamedPipe != 0 {
		reader := bufio.NewReader(os.Stdin)
		pipedData, err := io.ReadAll(reader)
		if err != nil {
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
						if params.Truncate {
							log.Printf("The total number of tokens (%d) exceeds the maximum allowed (%d). Truncating piped data.", totalTokens, maxTokens)
							numTokens = maxTokens - (totalTokens - numTokens)

							if numTokens <= 0 {
								return
							}

							body = body[:numTokens]
							totalTokens = maxTokens

						} else {
							log.Fatalf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, maxTokens)
						}
					}
				}()

				hash := sha256.Sum256([]byte(body))
				sha := hex.EncodeToString(hash[:])

				fileNameResp, err := Api.FileName(body)
				if err != nil {
					log.Fatalf("Failed to summarize piped data: %v", err)
				}

				fileName := GetFileNameWithoutExt(fileNameResp.FileName)

				contextPart := shared.ModelContextPart{
					Name:      fmt.Sprintf("%d.%s", atomic.LoadUint32(&counter), fileName),
					Body:      body,
					Sha:       sha,
					NumTokens: numTokens,
					UpdatedAt: StringTs(),
				}

				contextPartsMutex.Lock()
				contextParts = append(contextParts, contextPart)
				contextPartsMutex.Unlock()

				atomic.AddUint32(&counter, 1)
			}()
		}
	}

	var inputUrls []string
	var inputFilePaths []string

	if len(params.Resources) > 0 {
		for _, resource := range params.Resources {
			// so far resources are either files or urls
			if IsValidURL(resource) {
				inputUrls = append(inputUrls, resource)
			} else {
				inputFilePaths = append(inputFilePaths, resource)
			}
		}
	}

	if len(inputFilePaths) > 0 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			flattenedPaths := FlattenPaths(inputFilePaths, params)

			if params.NamesOnly {
				body := strings.Join(flattenedPaths, "\n")
				bytes := []byte(body)

				fmt.Println(body)

				hash := sha256.Sum256(bytes)
				sha := hex.EncodeToString(hash[:])
				numTokens := shared.GetNumTokens(body)

				totalTokensMutex.Lock()
				func() {
					defer totalTokensMutex.Unlock()
					totalTokens += numTokens
					tokensAdded += numTokens
					if totalTokens > maxTokens {

						if params.Truncate {
							log.Printf("The total number of tokens (%d) exceeds the maximum allowed (%d). Truncating filenames.", totalTokens, maxTokens)
							numTokens = maxTokens - (totalTokens - numTokens)

							// If the number of tokens is less than or equal to 0, then we can stop processing files
							if numTokens <= 0 {
								return
							}

							body = body[:numTokens]
							totalTokens = maxTokens

						} else {
							log.Fatalf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, maxTokens)
						}

					}

				}()

				contextPart := shared.ModelContextPart{
					Name:      fmt.Sprintf("%d-directory-layout", atomic.LoadUint32(&counter)),
					Body:      body,
					Sha:       sha,
					NumTokens: numTokens,
					UpdatedAt: StringTs(),
				}

				contextPartsMutex.Lock()
				contextParts = append(contextParts, contextPart)
				contextPartsMutex.Unlock()

				atomic.AddUint32(&counter, 1)

			} else {
				for _, path := range flattenedPaths {
					wg.Add(1)

					go func(path string) {
						defer wg.Done()

						fileContent, err := os.ReadFile(path)
						if err != nil {
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
							if totalTokens > maxTokens {

								if params.Truncate {
									log.Printf("The total number of tokens (%d) exceeds the maximum allowed (%d). Truncating the file %s.", totalTokens, maxTokens, path)
									numTokens = maxTokens - (totalTokens - numTokens)

									// If the number of tokens is less than or equal to 0, then we can stop processing files
									if numTokens <= 0 {
										return
									}

									body = body[:numTokens]
									totalTokens = maxTokens

								} else {
									log.Fatalf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, maxTokens)
								}

							}

						}()

						_, fileName := filepath.Split(path)

						contextPart := shared.ModelContextPart{
							Name:      fmt.Sprintf("%d.%s", atomic.LoadUint32(&counter), fileName),
							Body:      body,
							FilePath:  path,
							Sha:       sha,
							NumTokens: numTokens,
							UpdatedAt: StringTs(),
						}

						contextPartsMutex.Lock()
						contextParts = append(contextParts, contextPart)
						contextPartsMutex.Unlock()

						atomic.AddUint32(&counter, 1)
					}(path)

				}
			}
		}()

	}

	if len(inputUrls) > 0 {
		for _, url := range inputUrls {
			wg.Add(1)

			go func(url string) {
				defer wg.Done()

				body, err := FetchURLContent(url)
				if err != nil {
					log.Fatalf("Failed to fetch content from URL %s: %v", url, err)
				}

				numTokens := shared.GetNumTokens(body)

				totalTokensMutex.Lock()
				func() {
					defer totalTokensMutex.Unlock()
					totalTokens += numTokens
					tokensAdded += numTokens
					if totalTokens > maxTokens {
						if params.Truncate {
							log.Printf("The total number of tokens (%d) exceeds the maximum allowed (%d). Truncating content from URL %s.", totalTokens, maxTokens, url)
							numTokens = maxTokens - (totalTokens - numTokens)

							// If the number of tokens is less than or equal to 0, then we can stop processing content
							if numTokens <= 0 {
								return
							}

							body = body[:numTokens]
							totalTokens = maxTokens

						} else {
							log.Fatalf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, maxTokens)
						}
					}
				}()

				hash := sha256.Sum256([]byte(body))
				sha := hex.EncodeToString(hash[:])

				// fileNameResp, err := Api.FileName(body)
				// if err != nil {
				// 	log.Fatalf("Failed to summarize content from URL %s: %v", url, err)
				// }

				contextPart := shared.ModelContextPart{
					Name:      fmt.Sprintf("%d.%s", atomic.LoadUint32(&counter), SanitizeAndClipURL(url, 70)),
					Body:      body,
					Sha:       sha,
					NumTokens: numTokens,
					UpdatedAt: StringTs(),
				}

				contextPartsMutex.Lock()
				contextParts = append(contextParts, contextPart)
				contextPartsMutex.Unlock()

				atomic.AddUint32(&counter, 1)
			}(url)
		}
	}

	wg.Wait()

	if len(contextParts) == 0 {
		log.Fatalln("No context loaded")
	}

	wg = sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		writeContextParts(contextParts)
	}()

	wg.Add(1)

	go func() {
		defer wg.Done()
		contextState.NumTokens = totalTokens
		contextState.Counter = counter

		data, err := json.MarshalIndent(contextState, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal context state: %v", err)
		}
		// write file
		err = os.WriteFile(contextStateFilePath, data, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write context state to file: %v", err)
		}
	}()

	wg.Wait()

	var added []string
	if params.Note != "" {
		added = append(added, "a note")
	}
	if hasPipeData {
		added = append(added, "piped data")
	}
	if len(inputFilePaths) > 0 {
		label := "file"
		if len(inputFilePaths) > 1 {
			label = "files"
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
	msg += fmt.Sprintf(" into context | added → %d ./🪙 |  total → %d 🪙", tokensAdded, totalTokens)

	if err != nil {
		log.Fatalf("Failed to get total tokens: %v", err)
	}

	// msg += fmt.Sprintf("\n\nTotal tokens in context: %d\n", totalTokens)

	err = GitAddAndCommit(ContextSubdir, msg)

	if err != nil {
		log.Fatalf("Failed to commit context: %v", err)
	}

	err = GitAdd(CurrentPlanRootDir, ContextSubdir, true)
	if err != nil {
		log.Fatalf("failed to stage submodule changes in context dir: %s\n", err)
	}

	// Commit these staged submodule changes in the root repo
	err = GitCommit(CurrentPlanRootDir, msg, true)
	if err != nil {
		log.Fatalf("failed to commit submodule updates in root dir: %s\n", err)
	}

	elapsed := time.Since(timeStart)

	if elapsed < 700*time.Millisecond {
		time.Sleep(700*time.Millisecond - elapsed)
	}

	s.Stop()
	clearCurrentLine()
	fmt.Fprintln(os.Stderr, "✅ "+msg)

	return tokensAdded, totalTokens
}
