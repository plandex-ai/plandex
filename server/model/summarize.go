package model

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func ShortSummary(text string) ([]byte, int, error) {
	resp, err := Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Functions: []openai.FunctionDefinition{{
				Name: "summarize",
				Parameters: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"summary": {
							Type:        jsonschema.String,
							Description: "A brief summary of the text. A few sentences at most",
						},
					},
					Required: []string{"summary"},
				},
			}},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an AI summarizer that summarizes text, including programs, documentation, websites, and more. Most text will be related to software development.",
				},
				{
					Role: openai.ChatMessageRoleUser,
					Content: (`
						 Please summarize the text below using the 'summarize' function. Only call the 'summarize' function in your reponse. Don't call any other function.

						 Text:

					` + text),
				},
			},
		},
	)

	if err != nil {
		return nil, 0, err
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
		return nil, 0, fmt.Errorf("no summarize function call found in response")
	}

	// validate the JSON response
	var summarizeResp shared.ShortSummaryResponse
	if err := json.Unmarshal(byteRes, &summarizeResp); err != nil {
		return nil, resp.Usage.CompletionTokens, err
	}

	return byteRes, resp.Usage.CompletionTokens, nil
}

func PlanSummary(conversation []*shared.ConversationMessage) ([]byte, int, error) {
	messages := []openai.ChatCompletionMessage{}

	for _, msg := range conversation {
		messages = append(messages, msg.Message)
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: ``,
	})

	resp, err := Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Functions: []openai.FunctionDefinition{{
				Name: "summarize",
				Parameters: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"summary": {
							Type:        jsonschema.String,
							Description: "A summary of the current state of the plan.",
						},
					},
					Required: []string{"summary"},
				},
			}},
			Messages: messages,
		},
	)

	if err != nil {
		return nil, 0, err
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
		return nil, 0, fmt.Errorf("no summarize function call found in response")
	}

	// validate the JSON response
	var summarizeResp shared.ShortSummaryResponse
	if err := json.Unmarshal(byteRes, &summarizeResp); err != nil {
		return nil, resp.Usage.CompletionTokens, err
	}

	return byteRes, resp.Usage.CompletionTokens, nil
}
