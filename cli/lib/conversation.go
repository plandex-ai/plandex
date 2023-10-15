package lib

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"plandex/types"
	"strconv"
	"strings"

	"github.com/plandex/plandex/shared"
	openai "github.com/sashabaranov/go-openai"
)

func loadConversation() ([]shared.ConversationMessage, error) {
	var messages []shared.ConversationMessage

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
			var currentTokens int
			var contentBuffer []string

			for scanner.Scan() {
				line := scanner.Text()

				// Check if the line starts with user or response indicator
				if strings.HasPrefix(line, "@@@!>user|") {
					if currentRole != "" {
						// Save the previous message before starting a new one
						messages = append(messages, shared.ConversationMessage{
							Message: openai.ChatCompletionMessage{
								Role:    currentRole,
								Content: strings.Join(contentBuffer, "\n"),
							},
							Tokens: currentTokens,
						})
						contentBuffer = []string{}
					}
					currentRole = openai.ChatMessageRoleUser
					// Parse the number of tokens from the line (tokens only)
					currentTokensStr := strings.Split(line, "|")[2]
					currentTokens, err = strconv.Atoi(currentTokensStr)
					if err != nil {
						return nil, err
					}

					continue
				} else if strings.HasPrefix(line, "@@@!>response|") {
					if currentRole != "" {
						// Save the previous message before starting a new one
						messages = append(messages, shared.ConversationMessage{
							Message: openai.ChatCompletionMessage{
								Role:    currentRole,
								Content: strings.Join(contentBuffer, "\n"),
							}})
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
				messages = append(messages, shared.ConversationMessage{
					Message: openai.ChatCompletionMessage{
						Role:    currentRole,
						Content: strings.Join(contentBuffer, "\n"),
					},
					Tokens: currentTokens,
				})
			}

			file.Close()
		}
	}

	return messages, nil
}

func appendConversation(params types.AppendConversationParams) error {
	// Create or append to conversation file
	responseTimestamp := StringTs()
	conversationFilePath := filepath.Join(ConversationSubdir, fmt.Sprintf("%s.md", params.Timestamp))
	userHeader := fmt.Sprintf("@@@!>user|%s|%d\n\n", params.Timestamp, params.PromptTokens)
	responseHeader := fmt.Sprintf("@@@!>response|%s|%d\n\n", responseTimestamp, params.ReplyTokens)

	conversationFileContents := fmt.Sprintf("%s%s\n\n%s%s", userHeader, params.Prompt, responseHeader, params.Reply)
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

func setSummary(params types.ConversationSummaryParams) error {
	// Create or append to summary file
	summaryFilePath := filepath.Join(ConversationSubdir, fmt.Sprintf("%s.summary.md", params.MessageTimestamp))
	summaryFileContents := fmt.Sprintf("@@@!>response-summary|%s|%d\n\n%s", params.CurrentTimestamp, params.Summary, params.SummaryTokens)
	summaryFile, err := os.OpenFile(summaryFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to write summary file: %s\n", err)
	}
	defer summaryFile.Close()

	_, err = summaryFile.WriteString(summaryFileContents)
	if err != nil {
		return fmt.Errorf("failed to write summary file: %s\n", err)
	}

	return nil
}
