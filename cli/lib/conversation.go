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
