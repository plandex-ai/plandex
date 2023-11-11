package model

import (
	"context"
	"encoding/json"
	"fmt"
	"plandex-server/model/prompts"
	"strings"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func FileName(text string) ([]byte, int, error) {
	resp, err := Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:     NameModel,
			Functions: []openai.FunctionDefinition{prompts.FileNameFn},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: prompts.SysFileName,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompts.GetFileNamePrompt(text),
				},
			},
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
		},
	)

	if err != nil {
		return nil, 0, err
	}

	var byteRes []byte
	for _, choice := range resp.Choices {
		if choice.FinishReason == "function_call" && choice.Message.FunctionCall != nil && choice.Message.FunctionCall.Name == "nameFile" {
			fnCall := choice.Message.FunctionCall

			if strings.HasSuffix(fnCall.Arguments, ",\n}") { // remove trailing comma
				fnCall.Arguments = strings.TrimSuffix(fnCall.Arguments, ",\n}") + "\n}"
			}

			byteRes = []byte(fnCall.Arguments)
		}
	}

	if len(byteRes) == 0 {
		return nil, 0, fmt.Errorf("no nameFile function call found in response")
	}

	// validate the JSON response
	var nameFileResp shared.FileNameResponse
	if err := json.Unmarshal(byteRes, &nameFileResp); err != nil {
		return nil, resp.Usage.CompletionTokens, err
	}

	return byteRes, resp.Usage.CompletionTokens, nil
}
