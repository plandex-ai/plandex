package model

import (
	"context"
	"encoding/json"
	"fmt"

	"plandex-server/types"

	"github.com/davecgh/go-spew/spew"
	"github.com/sashabaranov/go-openai"
)

func Sectionize(text string) ([]byte, error) {
	resp, err := Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an AI specialized in processing text and code, dividing them into logical sections and subsections based on their content. After sectionization, you should call the 'create_sections' function with your sectionization results.",
				},
				{
					Role: openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Sectionize the text below, identifying logical sections and subsections, and call 'create_sections' with the sectionized results: %s",
						text),
				},
			},
		})

	if err != nil {
		return nil, err
	}

	var byteRes []byte
	for _, choice := range resp.Choices {
		if choice.FinishReason == "stop" && choice.Message.Role == "assistant" && choice.Message.FunctionCall != nil && choice.Message.FunctionCall.Name == "create_sections" {
			fnCall := choice.Message.FunctionCall
			byteRes = []byte(fnCall.Arguments)
		}
	}

	if len(byteRes) == 0 {
		return nil, fmt.Errorf("no 'create_sections' function call found in response")
	}

	// validate the JSON response
	var sectionizeResp types.SectionizeResponse
	if err := json.Unmarshal(byteRes, &sectionizeResp); err != nil {
		return nil, err
	}

	spew.Dump(sectionizeResp)

	return byteRes, nil
}
