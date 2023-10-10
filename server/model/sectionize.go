package model

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/plandex/plandex/server/types"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func Sectionize(text string) ([]byte, error) {
	resp, err := Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Functions: []openai.FunctionDefinition{{
				Name: "sectionize",
				Parameters: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"text": {
							Type:        jsonschema.String,
							Description: "The text that needs to be sectionized",
						},
					},
					Required: []string{"text"},
				},
			}},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an AI that breaks up text into logical sections. For code, each function would be its own section. Sections can also contain subsections.",
				},
				{
					Role: openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Please sectionize the text below: \n %s", text),
				},
			},
		},
	)

	if err != nil {
		return nil, err
	}

	var byteRes []byte
	for _, choice := range resp.Choices {
		if choice.FinishReason == "function_call" && choice.Message.FunctionCall != nil && choice.Message.FunctionCall.Name == "sectionize" {
			fnCall := choice.Message.FunctionCall
			byteRes = []byte(fnCall.Arguments)
		}
	}

	if len(byteRes) == 0 {
		return nil, fmt.Errorf("no sectionize function call found in response")
	}

	// validate the JSON response
	var sectionizeResp types.SectionizeResponse
	if err := json.Unmarshal(byteRes, &sectionizeResp); err != nil {
		return nil, err
	}

	spew.Dump(sectionizeResp)

	return byteRes, nil
}

	spew.Dump(sectionizeResp)

	return byteRes, nil
}
