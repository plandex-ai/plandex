package lib

import (
	"bufio"
	"os"
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
