// this file is unused as it hasn't yet been needed
// keeping it around for now in case it's useful at some point

package model

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type fixed struct {
	FixedJson string `json:"fixedJson"`
}

func FixJson(invalidJson string) ([]byte, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: `You are an AI JSON fixer. You take invalid JSON and fix it so that it becomes valid. You should call the 'fixed' function with the 'fixedJson' parameter, a string containing valid JSON. You must *always* call the 'fixed' function and never call any other function.`,
		},
		{
			Role: openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("Fix the following invalid JSON and call the 'fixed' function with the result:\n\n%s",
				invalidJson),
		},
	}

	resp, err := Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Functions: []openai.FunctionDefinition{{
				Name: "fixed",
				Parameters: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"fixedJson": {
							Type:        jsonschema.String,
							Description: "Valid json.",
							Items: &jsonschema.Definition{
								Type: jsonschema.String,
							},
						},
					},
					Required: []string{"fixedJson"},
				},
			}},
			Messages: messages,
		})

	if err != nil {
		return nil, err
	}

	spew.Dump(resp)

	var byteRes []byte
	for _, choice := range resp.Choices {
		if choice.FinishReason == "function_call" && choice.Message.Role == "assistant" && choice.Message.FunctionCall != nil && choice.Message.FunctionCall.Name == "fixed" {
			fnCall := choice.Message.FunctionCall
			byteRes = []byte(fnCall.Arguments)
		}
	}

	if len(byteRes) == 0 {
		return nil, fmt.Errorf("no 'fixed' function call found in response")
	}

	// Clean the body of the response
	// cleanBody, err := shared.CleanJson(byteRes)
	// if err != nil {
	// 	return nil, fmt.Errorf("error cleaning JSON: %w", err)
	// }

	// validate the JSON response
	var fixResp fixed
	err = json.Unmarshal(byteRes, &fixResp)

	if err != nil {
		fmt.Printf("Failed JSON: %s\n", byteRes) // Printing the JSON causing the failure
		return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	return byteRes, nil
}
