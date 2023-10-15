package model

import (
	"context"
	"encoding/json"
	"fmt"

	"plandex-server/types"

	"github.com/davecgh/go-spew/spew"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func Sectionize(text string) ([]byte, error) {
	resp, err := Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo16K,
			Functions: []openai.FunctionDefinition{{
				Name: "sectionize",
				Parameters: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"sections": {
							Type:        jsonschema.Array,
							Description: "A list of sections.",
							Items: &jsonschema.Definition{
								Type: jsonschema.String,
							},
						},
					},
					Required: []string{"sections"},
				},
			}},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an AI specialized in processing text and code, dividing them into logical sections based on their content. You should call the 'sectionize' function with an array of sections. Make sure the sections add up to the original text, don't overlap, don't leave any gaps, and are in the right order (i.e. the first section should come first, the second section should come second, etc.). If the text is code, try not to split up functions unless they are very long. Only call the 'sectionize' function and don't call any other function.",
				},
				{
					Role: openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Sectionize the text below, identifying logical sections, and call 'sectionize' with the sectionized results.\n\nBelow is the text to be sectionized:\n\n%s",
						text),
				},
			},
		})

	if err != nil {
		return nil, err
	}

	var byteRes []byte
	for _, choice := range resp.Choices {
		if choice.FinishReason == "stop" && choice.Message.Role == "assistant" && choice.Message.FunctionCall != nil && choice.Message.FunctionCall.Name == "sectionize" {
			fnCall := choice.Message.FunctionCall
			byteRes = []byte(fnCall.Arguments)
		}
	}

	if len(byteRes) == 0 {
		return nil, fmt.Errorf("no 'sectionize' function call found in response")
	}

	// validate the JSON response
	var sectionizeResp types.SectionizeResponse
	if err := json.Unmarshal(byteRes, &sectionizeResp); err != nil {
		return nil, err
	}

	spew.Dump(sectionizeResp)

	return byteRes, nil
}
