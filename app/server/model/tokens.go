package model

import (
	"plandex-server/types"
	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

const (
	// Per OpenAI's documentation:
	// Every message follows this format: {"role": "role_name", "content": "content"}
	// which has a 4-token overhead per message
	TokensPerMessage = 4

	// System, user, or assistant - each role name costs 1 token
	TokensPerName = 1

	// Tokens per request
	TokensPerRequest = 3

	TokensPerExtendedPart = 6
)

func GetMessagesTokenEstimate(messages ...types.ExtendedChatMessage) int {
	tokens := 0

	for _, msg := range messages {
		tokens += TokensPerMessage // Base message overhead
		tokens += TokensPerName    // Role name

		if len(msg.Content) > 0 {
			// For each extended part, we need to account for the JSON structure
			// Each part follows format: {"type": "type_value", "text": "content"}
			// or {"type": "type_value", "image_url": {"url": "url_value"}}
			for _, part := range msg.Content {
				if part.Type == openai.ChatMessagePartTypeText {
					tokens += TokensPerExtendedPart // Overhead for the part object structure
					tokens += shared.GetNumTokensEstimate(part.Text)
				}

				// images are handled separately

			}
		}

	}

	return tokens
}
