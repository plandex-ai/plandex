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

func ShortSummary(text string) ([]byte, error) {
	resp, err := Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:     ShortSummaryModel,
			Functions: []openai.FunctionDefinition{prompts.ShortSummaryFn},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: prompts.SysShortSummary,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompts.GetShortSummaryPrompt(text),
				},
			},
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
		},
	)

	if err != nil {
		return nil, err
	}

	var byteRes []byte
	for _, choice := range resp.Choices {
		if choice.FinishReason == "function_call" && choice.Message.FunctionCall != nil && choice.Message.FunctionCall.Name == "summarize" {
			fnCall := choice.Message.FunctionCall

			if strings.HasSuffix(fnCall.Arguments, ",\n}") { // remove trailing comma
				fnCall.Arguments = strings.TrimSuffix(fnCall.Arguments, ",\n}") + "\n}"
			}

			byteRes = []byte(fnCall.Arguments)
		}
	}

	if len(byteRes) == 0 {
		return nil, fmt.Errorf("no summarize function call found in response")
	}

	// validate the JSON response
	var summarizeResp shared.ShortSummaryResponse
	if err := json.Unmarshal(byteRes, &summarizeResp); err != nil {
		return nil, err
	}

	return byteRes, nil
}

func PlanSummary(conversation []openai.ChatCompletionMessage, lastMessageTimestamp string, numMessages int) (*shared.ConversationSummary, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.Identity,
		},
	}

	messages = append(messages, conversation...)

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompts.PlanSummary,
	})

	// fmt.Println("summarizing messages:")
	// spew.Dump(messages)

	resp, err := Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    PlanSummaryModel,
			Messages: messages,
		},
	)

	if err != nil {
		fmt.Println("PlanSummary err:", err)

		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from GPT")
	}

	content := resp.Choices[0].Message.Content

	return &shared.ConversationSummary{
		Summary:              content,
		Tokens:               resp.Usage.CompletionTokens,
		LastMessageTimestamp: lastMessageTimestamp,
		NumMessages:          numMessages,
	}, nil

}
