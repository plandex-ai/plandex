package lib

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

func loadConversation() ([]openai.ChatCompletionMessage, error) {
	var messages []openai.ChatCompletionMessage

	files, err := os.ReadDir(ConversationSubdir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		// Ensure we are only reading files, not directories
		if !file.IsDir() {
			filePath := ConversationSubdir + "/" + file.Name()
			file, err := os.Open(filePath)
			if err != nil {
				return nil, err
			}

			scanner := bufio.NewScanner(file)
			var currentRole string
			var contentBuffer []string

			for scanner.Scan() {
				line := scanner.Text()

				// Check if the line starts with user or response indicator
				if strings.HasPrefix(line, "@@@!>user|") {
					if currentRole != "" {
						// Save the previous message before starting a new one
						messages = append(messages, openai.ChatCompletionMessage{
							Role:    currentRole,
							Content: strings.Join(contentBuffer, "\n"),
						})
						contentBuffer = []string{}
					}
					currentRole = openai.ChatMessageRoleUser
					continue
				} else if strings.HasPrefix(line, "@@@!>response|") {
					if currentRole != "" {
						// Save the previous message before starting a new one
						messages = append(messages, openai.ChatCompletionMessage{
							Role:    currentRole,
							Content: strings.Join(contentBuffer, "\n"),
						})
						contentBuffer = []string{}
					}
					currentRole = openai.ChatMessageRoleAssistant
					continue
				}

				// Add content to the buffer
				contentBuffer = append(contentBuffer, line)
			}

			// Add the last message in the file
			if currentRole != "" && len(contentBuffer) > 0 {
				messages = append(messages, openai.ChatCompletionMessage{
					Role:    currentRole,
					Content: strings.Join(contentBuffer, "\n"),
				})
			}

			file.Close()
		}
	}

	return messages, nil
}

func appendConversation(timestamp, prompt, reply string) error {
	// Create or append to conversation file
	responseTimestamp := StringTs()
	conversationFilePath := filepath.Join(ConversationSubdir, fmt.Sprintf("%s.md", timestamp))
	userHeader := fmt.Sprintf("@@@!>user|%s\n\n", timestamp)
	responseHeader := fmt.Sprintf("@@@!>response|%s\n\n", responseTimestamp)

	// TODO: store both summary and full response in conversation file for different use cases/context needs
	conversationFileContents := fmt.Sprintf("%s%s\n\n%s%s", userHeader, prompt, responseHeader, reply)
	conversationFile, err := os.OpenFile(conversationFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to write conversation file: %s\n", err)
	}
	defer conversationFile.Close()

	_, err = conversationFile.WriteString(conversationFileContents)
	if err != nil {
		return fmt.Errorf("failed to write conversation file: %s\n", err)
	}

	return nil
}

type replyTokenCounter struct {
	lines           []string
	lineIndex       int
	maybeFilePath   string
	currentFilePath string
	numTokensByFile map[string]int
}

func NewReplyTokenCounter() *replyTokenCounter {
	return &replyTokenCounter{
		lines:           []string{""},
		numTokensByFile: make(map[string]int),
	}
}

func (r *replyTokenCounter) AddChunk(chunk string) {
	// fmt.Println("Adding chunk:", strconv.Quote(chunk)) // Logging the chunk that's being processed

	hasNewLine := false
	nextChunk := ""

	if chunk == "\n" {
		// fmt.Println("Chunk is \\n, adding new line")
		r.lines = append(r.lines, "")
		hasNewLine = true
		r.lineIndex++
	} else {
		chunkLines := strings.Split(chunk, "\n")

		// fmt.Println("Chunk lines:", len(chunkLines))

		currentLine := r.lines[r.lineIndex]
		currentLine += chunkLines[0]

		// fmt.Println("Current line:", strconv.Quote(currentLine))
		r.lines[r.lineIndex] = currentLine

		if len(chunkLines) > 1 {
			r.lines = append(r.lines, chunkLines[1])
			r.lineIndex++
			hasNewLine = true

			if len(chunkLines) > 2 {
				tail := chunkLines[2:]
				nextChunk = "\n" + strings.Join(tail, "\n")
				defer func() {
					// fmt.Println("Recursive add next queued chunk:", strconv.Quote(nextChunk))
					r.AddChunk(nextChunk)
				}()
			}
		}
	}

	if r.lineIndex == 0 || !hasNewLine {
		return
	}

	prevFullLine := r.lines[r.lineIndex-1]
	// fmt.Println("Previous full line:", strconv.Quote(prevFullLine)) // Logging the full line that's being checked

	if r.maybeFilePath != "" {
		// fmt.Println("Maybe file path is:", r.maybeFilePath) // Logging the maybeFilePath
		if strings.HasPrefix(prevFullLine, "```") {
			r.currentFilePath = r.maybeFilePath
			r.maybeFilePath = ""
			// fmt.Println("Confirmed file path:", r.currentFilePath) // Logging the confirmed file path
		} else if prevFullLine != "" {
			// turns out previous maybeFilePath was not a file path since there's a non-empty line before finding opening ticks
			r.maybeFilePath = ""
		}
		return
	}

	if r.currentFilePath == "" {
		var gotPath string
		if (strings.HasPrefix(prevFullLine, "-") && strings.HasSuffix(prevFullLine, ":")) || strings.HasPrefix(prevFullLine, "-file:") || strings.HasPrefix(prevFullLine, "- file:") {
			p := strings.TrimPrefix(prevFullLine, "-")
			p = strings.TrimSpace(p)
			p = strings.TrimPrefix(p, "file:")
			p = strings.TrimSuffix(p, ":")
			p = strings.TrimSpace(p)
			gotPath = p
		}

		if gotPath != "" {
			// fmt.Println("Detected possible file path:", gotPath) // Logging the possible file path

			if r.maybeFilePath == "" {
				r.maybeFilePath = gotPath
			} else {
				r.maybeFilePath = gotPath
			}
		}
	} else {
		if strings.HasPrefix(prevFullLine, "```") {
			r.currentFilePath = ""
			// fmt.Println("Exited file block.")
		} else {
			tokens := int(GetNumTokens(prevFullLine))
			r.numTokensByFile[r.currentFilePath] += tokens
			// r.contentByFile[r.currentFilePath] += prevFullLine + "\n"
			// fmt.Printf("Added %d tokens to %s\n", tokens, r.currentFilePath) // Logging token addition
		}
	}
}

func (r *replyTokenCounter) FinishAndRead() map[string]int {
	r.AddChunk("\n")
	return r.numTokensByFile
}

// func getTokensPerFilePath(reply string) map[string]int {
// 	lines := strings.Split(reply, "\n")

// 	numTokensByFile := make(map[string]int)

// 	currentFilePath := ""
// 	foundOpeningTicks := false

// 	for i, line := range lines {
// 		if currentFilePath == "" {
// 			line = strings.TrimSpace(line)

// 			// fmt.Println("checking line for file path: ", line)

// 			if (strings.HasPrefix(line, "-") && strings.HasSuffix(line, ":")) || strings.HasPrefix(line, "- file:") {
// 				// fmt.Println("Found file path line: ", line)

// 				// check if '```' is in the next two lines
// 				if i+2 < len(lines) && (strings.HasPrefix(lines[i+1], "```") || strings.HasPrefix(lines[i+2], "```")) {
// 					// this is a file path
// 					currentFilePath = strings.TrimPrefix(line, "-")
// 					currentFilePath = strings.TrimPrefix(line, "- file:")
// 					currentFilePath = strings.TrimSuffix(currentFilePath, ":")
// 					numTokensByFile[currentFilePath] = 0
// 					// fmt.Printf("Found file path: %s\n", currentFilePath)
// 				}

// 			}
// 		} else if foundOpeningTicks {
// 			if strings.HasPrefix(line, "```") {
// 				// found closing ticks
// 				currentFilePath = ""
// 				foundOpeningTicks = false
// 				// fmt.Println("Found closing ticks")
// 			} else {
// 				numTokensByFile[currentFilePath] += int(GetNumTokens(line))
// 				// fmt.Printf("Added %d tokens to file %s\n", int(GetNumTokens(line)), currentFilePath)
// 			}
// 		} else if strings.HasPrefix(line, "```") {
// 			// found opening ticks
// 			foundOpeningTicks = true
// 			// fmt.Println("Found opening ticks")
// 		}
// 	}

// 	return numTokensByFile
// }
