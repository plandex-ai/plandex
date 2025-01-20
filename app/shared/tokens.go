package shared

import (
	"fmt"

	"github.com/pkoukk/tiktoken-go"
	"github.com/sashabaranov/go-openai"
)

var tkm *tiktoken.Tiktoken

func init() {
	var err error
	tkm, err = tiktoken.EncodingForModel("gpt-4o")
	if err != nil {
		panic(fmt.Sprintf("error getting encoding for model: %v", err))
	}
}

func GetNumTokensEstimate(text string) int {
	return len(tkm.Encode(text, nil, nil))
}

const (
	// Per OpenAI's documentation:
	// Every message follows this format: {"role": "role_name", "content": "content"}
	// which has a 4-token overhead per message
	TokensPerMessage = 4

	// System, user, or assistant - each role name costs 1 token
	TokensPerName = 1

	// Tokens per request
	TokensPerRequest = 3
)

func GetMessagesTokenEstimate(messages ...openai.ChatCompletionMessage) int {
	tokens := 0

	for _, msg := range messages {
		tokens += TokensPerMessage
		tokens += TokensPerName
		tokens += GetNumTokensEstimate(msg.Content)
	}

	return tokens
}
