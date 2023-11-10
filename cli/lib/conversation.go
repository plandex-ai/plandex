package lib

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"plandex/types"
	"sort"
	"strconv"
	"strings"

	"github.com/plandex/plandex/shared"
	openai "github.com/sashabaranov/go-openai"
)

func LoadConversation() ([]shared.ConversationMessage, error) {
	var messages []shared.ConversationMessage

	files, err := os.ReadDir(ConversationSubdir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		// Ensure we are only reading files, not directories
		if file.IsDir() {
			continue
		}
		filePath := ConversationSubdir + "/" + file.Name()
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}

		scanner := bufio.NewScanner(file)
		var currentRole string
		var currentTokens int
		var currentTimestamp string
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
						Tokens:    currentTokens,
						Timestamp: currentTimestamp,
					})
					contentBuffer = []string{}
				}
				currentRole = openai.ChatMessageRoleUser
				// Parse the number of tokens from the line (tokens only)
				split := strings.Split(line, "|")
				currentTimestamp = split[1]
				currentTokensStr := split[2]
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
						},
						Tokens:    currentTokens,
						Timestamp: currentTimestamp,
					})
					contentBuffer = []string{}
				}
				currentRole = openai.ChatMessageRoleAssistant
				split := strings.Split(line, "|")
				currentTimestamp = split[1]
				currentTokensStr := split[2]
				currentTokens, err = strconv.Atoi(currentTokensStr)
				if err != nil {
					return nil, err
				}
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
				Tokens:    currentTokens,
				Timestamp: currentTimestamp,
			})
		}

		file.Close()
	}

	return messages, nil
}

func LoadSummaries() ([]shared.ConversationSummary, error) {
	// check if summaries subdirectory exists
	summariesPath := filepath.Join(ConversationSubdir, "summaries")
	exists := false
	_, err := os.Stat(summariesPath)
	if err == nil {
		exists = true
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	var summaries []shared.ConversationSummary
	if exists {

		files, err := os.ReadDir(summariesPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read summaries directory: %s", err)
		}

		for _, file := range files {
			// Ensure we are only reading files, not directories
			if file.IsDir() {
				continue
			}
			filePath := summariesPath + "/" + file.Name()

			bytes, err := os.ReadFile(filePath)

			if err != nil {
				return nil, fmt.Errorf("failed to read summary file: %s", err)
			}

			content := string(bytes)

			header := strings.Split(content, "\n")[0]
			summary := strings.Join(strings.Split(content, "\n")[2:], "\n")

			headerSplit := strings.Split(header, "|")
			timestamp := headerSplit[1]
			tokensStr := headerSplit[2]
			tokens, err := strconv.Atoi(tokensStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse tokens: %s", err)
			}

			summaries = append(summaries, shared.ConversationSummary{
				Summary:              summary,
				LastMessageTimestamp: timestamp,
				Tokens:               tokens,
			})
		}
	}

	// sort by timestamp
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].LastMessageTimestamp < summaries[j].LastMessageTimestamp
	})

	return summaries, nil
}

func appendConversation(params types.AppendConversationParams) error {
	// Create or append to conversation file
	conversationFilePath := filepath.Join(ConversationSubdir, fmt.Sprintf("%s.md", params.Timestamp))

	var header string
	var body string
	var tokens int
	var t string
	if params.PromptParams != nil {
		t = "prompt"
		header = fmt.Sprintf("@@@!>user|%s|%d\n\n", params.Timestamp, params.PromptParams.PromptTokens)
		body = params.PromptParams.Prompt
		tokens = params.PromptParams.PromptTokens
	} else if params.ReplyParams != nil {
		t = "reply"
		header = fmt.Sprintf("@@@!>response|%s|%d\n\n", params.ReplyParams.ResponseTimestamp, params.ReplyParams.ReplyTokens)
		body = params.ReplyParams.Reply
		tokens = params.ReplyParams.ReplyTokens
	} else {
		return fmt.Errorf("invalid params: either prompt or reply params must be provided")
	}

	conversationFileContents := fmt.Sprintf("%s%s\n\n", header, body)
	conversationFile, err := os.OpenFile(conversationFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to write conversation file: %s", err)
	}
	defer conversationFile.Close()

	_, err = conversationFile.WriteString(conversationFileContents)
	if err != nil {
		return fmt.Errorf("failed to write conversation file: %s", err)
	}

	params.PlanState.NumMessages++
	params.PlanState.ConvoTokens += tokens
	err = SetPlanState(params.PlanState, shared.StringTs())
	if err != nil {
		return fmt.Errorf("failed to update plan state: %s", err)
	}

	var desc string
	if t == "prompt" {
		desc = "💬 user prompt"
	} else {
		desc = "🤖 Plandex reply"
	}
	err = GitCommitConvoUpdate(fmt.Sprintf("Message #%d | %s | %d 🪙", params.PlanState.NumMessages, desc, tokens))

	if err != nil {
		return fmt.Errorf("failed to commit conversation update: %s", err)
	}

	return nil
}

func saveLatestConvoSummary(rootId string) error {
	summariesPath := filepath.Join(ConversationSubdir, "summaries")
	exists := false
	_, err := os.Stat(summariesPath)
	if err == nil {
		exists = true
	} else if !os.IsNotExist(err) {
		return err
	}

	if !exists {
		err = os.Mkdir(summariesPath, 0755)
		if err != nil {
			return err
		}
	}

	// get latest summary from directory
	files, err := os.ReadDir(summariesPath)
	if err != nil {
		return fmt.Errorf("failed to read summaries directory: %s", err)
	}

	var latestTimestamp string
	if len(files) > 0 {
		sort.Slice(files, func(i, j int) bool {
			return files[i].Name() < files[j].Name()
		})
		latestFile := files[len(files)-1]
		latestTimestamp = strings.Split(latestFile.Name(), ".")[0]
	}

	summary, err := Api.ConvoSummary(rootId, latestTimestamp)
	if err != nil {
		return fmt.Errorf("failed to get convo summary: %s", err)
	}

	if summary == nil {
		return nil
	}

	summaryFilePath := filepath.Join(summariesPath, fmt.Sprintf("%s.md", summary.LastMessageTimestamp))
	summaryFileContents := fmt.Sprintf("@@@!>summary|%s|%d\n\n%s", summary.LastMessageTimestamp, summary.Tokens, summary.Summary)

	err = os.WriteFile(summaryFilePath, []byte(summaryFileContents), 0644)

	if err != nil {
		return fmt.Errorf("failed to write summary file: %s", err)
	}

	err = GitCommitConvoUpdate(fmt.Sprintf("🤏 Summarized up to message #%d | %d 🪙", summary.NumMessages-1, summary.Tokens))

	if err != nil {
		return fmt.Errorf("failed to commit conversation update: %s", err)
	}

	return err
}
