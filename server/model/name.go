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

func FileName(text string) ([]byte, int, error) {
	resp, err := Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: NameModel,
			Functions: []openai.FunctionDefinition{{
				Name: "nameFile",
				Parameters: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"fileName": {
							Type:        jsonschema.String,
							Description: "A *short* file name for the text based on the content. Use dashes as word separators. No spaces or special characters. **2-3 words max**.",
						},
					},
					Required: []string{"fileName"},
				},
			}},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an AI namer that creates a valid filename for the content. Content can be any text, including programs/code, documentation, websites, and more. Most text will be related to software development.",
				},
				{
					Role: openai.ChatMessageRoleUser,
					Content: (`
						 Create a file name using the 'nameFile' function. Only call the 'nameFile' function in your reponse. Don't call any other function.

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
