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

	"github.com/plandex/plandex/shared"
)

func WriteInitialContextState(contextDir string) error {
	fmt.Println("Creating initial context state...")
	contextState := shared.ModelContextState{
		NumTokens:    0,
		ActiveTokens: 0,
		ChatFlexPct:  25,
		PlanFlexPct:  50,
	}
	contextStateFilePath := filepath.Join(contextDir, "context.json")
	contextStateFile, err := os.OpenFile(contextStateFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer contextStateFile.Close()

	contextStateFileContents, err := json.Marshal(contextState)
	if err != nil {
		return err
	}

	_, err = contextStateFile.Write(contextStateFileContents)

	return err
}

func LoadContextOrDie(params *types.LoadContextParams) {
	var contextState shared.ModelContextState
	contextStateFilePath := filepath.Join(ContextSubdir, "context.json")

	fmt.Fprintf(os.Stderr, "Loading context from %s\n", contextStateFilePath)

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
			numTokens := GetNumTokens(body)

			totalTokensMutex.Lock()
			func() {
				defer totalTokensMutex.Unlock()

				totalTokens += numTokens
				if totalTokens > params.MaxTokens {
					if params.Truncate {
						log.Printf("The total number of tokens (%d) exceeds the maximum allowed (%d). Truncating input text.", totalTokens, params.MaxTokens)
						numTokens = params.MaxTokens - (totalTokens - numTokens)

						// If the number of tokens is less than or equal to 0, then we can stop processing files
						if numTokens <= 0 {
							return
						}

						body = body[:numTokens]
						totalTokens = params.MaxTokens

					} else {
						log.Fatalf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, params.MaxTokens)
					}

				}
			}()

			hash := sha256.Sum256([]byte(body))
			sha := hex.EncodeToString(hash[:])

			summaryResp, err := Api.Summarize(body)
			if err != nil {
				log.Fatalf("Failed to summarize the text: %v", err)
			}

			fileName := GetFileNameWithoutExt(summaryResp.FileName)

			contextPart := shared.ModelContextPart{
				Name:      fmt.Sprintf("%d.%s", atomic.LoadUint32(&counter), fileName),
				Summary:   summaryResp.Summary,
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
				numTokens := GetNumTokens(body)

				totalTokensMutex.Lock()
				func() {
					defer totalTokensMutex.Unlock()

					totalTokens += numTokens
					if totalTokens > params.MaxTokens {
						if params.Truncate {
							log.Printf("The total number of tokens (%d) exceeds the maximum allowed (%d). Truncating piped data.", totalTokens, params.MaxTokens)
							numTokens = params.MaxTokens - (totalTokens - numTokens)

							if numTokens <= 0 {
								return
							}

							body = body[:numTokens]
							totalTokens = params.MaxTokens

						} else {
							log.Fatalf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, params.MaxTokens)
						}
					}
				}()

				hash := sha256.Sum256([]byte(body))
				sha := hex.EncodeToString(hash[:])

				summaryResp, err := Api.Summarize(body)
				if err != nil {
					log.Fatalf("Failed to summarize piped data: %v", err)
				}

				fileName := GetFileNameWithoutExt(summaryResp.FileName)

				contextPart := shared.ModelContextPart{
					Name:      fmt.Sprintf("%d.%s", atomic.LoadUint32(&counter), fileName),
					Summary:   summaryResp.Summary,
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

			flattenedPaths := FlattenPaths(inputFilePaths, params, 0)

			if params.NamesOnly {
				body := strings.Join(flattenedPaths, "\n")
				bytes := []byte(body)
				hash := sha256.Sum256(bytes)
				sha := hex.EncodeToString(hash[:])
				numTokens := GetNumTokens(body)

				totalTokensMutex.Lock()
				func() {
					defer totalTokensMutex.Unlock()
					totalTokens += numTokens
					if totalTokens > params.MaxTokens {

						if params.Truncate {
							log.Printf("The total number of tokens (%d) exceeds the maximum allowed (%d). Truncating filenames.", totalTokens, params.MaxTokens)
							numTokens = params.MaxTokens - (totalTokens - numTokens)

							// If the number of tokens is less than or equal to 0, then we can stop processing files
							if numTokens <= 0 {
								return
							}

							body = body[:numTokens]
							totalTokens = params.MaxTokens

						} else {
							log.Fatalf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, params.MaxTokens)
						}

					}

				}()

				contextPart := shared.ModelContextPart{
					Name:      fmt.Sprintf("%d-filenames", atomic.LoadUint32(&counter)),
					Summary:   "filenames",
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
						numTokens := GetNumTokens(body)

						totalTokensMutex.Lock()
						func() {
							defer totalTokensMutex.Unlock()
							totalTokens += numTokens
							if totalTokens > params.MaxTokens {

								if params.Truncate {
									log.Printf("The total number of tokens (%d) exceeds the maximum allowed (%d). Truncating the file %s.", totalTokens, params.MaxTokens, path)
									numTokens = params.MaxTokens - (totalTokens - numTokens)

									// If the number of tokens is less than or equal to 0, then we can stop processing files
									if numTokens <= 0 {
										return
									}

									body = body[:numTokens]
									totalTokens = params.MaxTokens

								} else {
									log.Fatalf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, params.MaxTokens)
								}

							}

						}()

						summaryResp, err := Api.Summarize(body)
						if err != nil {
							log.Fatalf("Failed to summarize the file %s: %v", path, err)
						}

						// get just the filename
						_, fileName := filepath.Split(path)

						contextPart := shared.ModelContextPart{
							Name:      fmt.Sprintf("%d.%s", atomic.LoadUint32(&counter), fileName),
							Summary:   summaryResp.Summary,
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

				numTokens := GetNumTokens(body)

				totalTokensMutex.Lock()
				func() {
					defer totalTokensMutex.Unlock()
					totalTokens += numTokens
					if totalTokens > params.MaxTokens {
						if params.Truncate {
							log.Printf("The total number of tokens (%d) exceeds the maximum allowed (%d). Truncating content from URL %s.", totalTokens, params.MaxTokens, url)
							numTokens = params.MaxTokens - (totalTokens - numTokens)

							// If the number of tokens is less than or equal to 0, then we can stop processing content
							if numTokens <= 0 {
								return
							}

							body = body[:numTokens]
							totalTokens = params.MaxTokens

						} else {
							log.Fatalf("The total number of tokens (%d) exceeds the maximum allowed (%d)", totalTokens, params.MaxTokens)
						}
					}
				}()

				hash := sha256.Sum256([]byte(body))
				sha := hex.EncodeToString(hash[:])

				summaryResp, err := Api.Summarize(body)
				if err != nil {
					log.Fatalf("Failed to summarize content from URL %s: %v", url, err)
				}

				contextPart := shared.ModelContextPart{
					Name:      fmt.Sprintf("%d.%s", atomic.LoadUint32(&counter), SanitizeAndClipURL(url, 70)),
					Summary:   summaryResp.Summary,
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
		added = append(added, fmt.Sprintf("%d files", len(inputFilePaths)))
	}
	if len(inputUrls) > 0 {
		added = append(added, fmt.Sprintf("%d urls", len(inputUrls)))
	}

	msg := "Loaded"
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
	msg += " into context"

	err = GitAddAndCommit(ContextSubdir, msg)

	if err != nil {
		return
	}

	err = GitAdd(CurrentPlanRootDir, ContextSubdir, true)
	if err != nil {
		fmt.Printf("failed to stage submodule changes in context dir: %s\n", err)
		return
	}

	// Commit these staged submodule changes in the root repo
	err = GitCommit(CurrentPlanRootDir, msg, true)
	if err != nil {
		fmt.Printf("failed to commit submodule updates in root dir: %s\n", err)
	}

	fmt.Fprint(os.Stderr, msg)
}

func GetAllContext(metaOnly bool) ([]shared.ModelContextPart, error) {
	files, err := os.ReadDir(ContextSubdir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read context directory: %v", err)
		return nil, err
	}

	var contexts []shared.ModelContextPart
	for _, file := range files {
		filename := file.Name()

		if filename == ".git" || filename == "context.json" {
			continue
		}

		// Only process .meta files and then look for their corresponding .body files
		if strings.HasSuffix(filename, ".meta") {
			// fmt.Fprintf(os.Stderr, "Reading meta context file %s\n", filename)

			metaContent, err := os.ReadFile(filepath.Join(ContextSubdir, filename))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read meta file %s: %v", filename, err)
				return nil, err
			}

			var contextPart shared.ModelContextPart
			if err := json.Unmarshal(metaContent, &contextPart); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to unmarshal JSON from file %s: %v", filename, err)
				return nil, err
			}

			if !metaOnly {
				// get the body filename by replacing the .meta suffix with .body
				bodyFilename := strings.TrimSuffix(filename, ".meta") + ".body"
				bodyPath := filepath.Join(ContextSubdir, bodyFilename)

				// fmt.Fprintf(os.Stderr, "Reading body context file %s\n", bodyPath)

				bodyContent, err := os.ReadFile(bodyPath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to read body file %s: %v", bodyPath, err)
					return nil, err
				}

				contextPart.Body = string(bodyContent)
			}

			contexts = append(contexts, contextPart)
		}
	}

	return contexts, nil
}

// createContextFileName constructs a filename based on the given name and sha.
func createContextFileName(name, sha string) string {
	// Extract the first 8 characters of the sha
	shaSubstring := sha[:8]
	return fmt.Sprintf("%s.%s", name, shaSubstring)
}

// writeContextPartToFile writes a single ModelContextPart to a file.
func writeContextPartToFile(part shared.ModelContextPart) error {
	metaFilename := createContextFileName(part.Name, part.Sha) + ".meta"
	metaPath := filepath.Join(ContextSubdir, metaFilename)

	bodyFilename := createContextFileName(part.Name, part.Sha) + ".body"
	bodyPath := filepath.Join(ContextSubdir, bodyFilename)
	body := []byte(part.Body)
	part.Body = ""

	// Convert the ModelContextPart to JSON
	data, err := json.MarshalIndent(part, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context part: %v", err)
	}

	// Open or create a bodyFile for writing
	bodyFile, err := os.OpenFile(bodyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s for writing: %v", bodyPath, err)
	}
	defer bodyFile.Close()

	// Write the body to the file
	if _, err = bodyFile.Write(body); err != nil {
		return fmt.Errorf("failed to write data to file %s: %v", bodyPath, err)
	}

	// Open or create a metaFile for writing
	metaFile, err := os.OpenFile(metaPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s for writing: %v", metaPath, err)
	}
	defer metaFile.Close()

	// Write the JSON data to the file
	if _, err = metaFile.Write(data); err != nil {
		return fmt.Errorf("failed to write data to file %s: %v", metaPath, err)
	}

	return nil
}

// Write each context part in parallel
func writeContextParts(contextParts []shared.ModelContextPart) {
	var wg sync.WaitGroup
	for _, part := range contextParts {
		wg.Add(1)
		go func(p shared.ModelContextPart) {
			defer wg.Done()
			if err := writeContextPartToFile(p); err != nil {
				// Handling the error in the goroutine by logging. Depending on your application,
				// you might want a different strategy (e.g., collect errors and handle them after waiting).
				fmt.Fprintf(os.Stderr, "Error writing context part to file: %v", err)
			}
		}(part)
	}
	wg.Wait() // Wait for all goroutines to finish writing
}
