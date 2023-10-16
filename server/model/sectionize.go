package model

import (
	"context"
	"encoding/json"
	"fmt"

	"plandex-server/model/lib"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type sectionizeResult = struct {
	Sections []string `json:"sections"`
}

func Sectionize(text string) ([]byte, error) {
	resp, err := Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Functions: []openai.FunctionDefinition{{
				Name: "sectionized",
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
					Role: openai.ChatMessageRoleSystem,
					Content: `You are an AI specialized in breaking code up into sections based on purpose and functionality. 
					
					Your instructions:

					1. Break up the provided text into sections based on purpose and functionality. 
						1a. If the text is code, try to group together related operations. 
						1b. If the text is natural language, try to group together related ideas.

					2. Call the 'sectionized' function with the the 'sections' parameter. For each section, provide a string containing *just the first line that begins the section.*									

					When breaking up the text into sections, follow these guidelines.
					- Sections should be roughly 50-100 lines in size.
					- A file should be broken up into no more than 5-10 sections.
					- For a short file, it's good to have a small number of sections, like 2-3 sections.
					- Sections should be understandable in isolation.
					- If in doubt, lean towards fewer sections rather than more sections.
					- Blocks of code that are commented out should be given their own sections.
					- Lean towards not breaking up functions or methods into multiple sections unless they are very long.
					- Lean towards not breaking up variable definitions, type definitions, or control flow blocks into multiple sections unless they are very long.
					- Lean towards not breaking up comment blocks into multiple sections unless they are very long.
					- Lean towards not breaking up paragraphs into multiple sections unless they are very long.
					- Lean towards keeping 'header' information like package declarations, imports, and initialization logic together in one section unless it would be very long.					
					- If there's no way to break up the text according to these instructions, call 'sectionized' with an empty array.

					Important: you *must always* call the 'sectionized' function and never call any other function.
					`,
				},
				{
					Role: openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Break up the following text into sections as described in your instructions above and call the 'sectionized' function with the result: \n\n%s",
						text),
				},
			},
		})

	if err != nil {
		return nil, err
	}

	spew.Dump(resp)

	var byteRes []byte
	for _, choice := range resp.Choices {
		if choice.FinishReason == "function_call" && choice.Message.Role == "assistant" && choice.Message.FunctionCall != nil && choice.Message.FunctionCall.Name == "sectionized" {
			fnCall := choice.Message.FunctionCall
			byteRes = []byte(fnCall.Arguments)
		}
	}

	if len(byteRes) == 0 {
		return nil, fmt.Errorf("no 'sectionized' function call found in response")
	}

	// Clean the body of the response
	cleanBody := lib.CleanSectionJson(byteRes)

	// validate the JSON response
	var res sectionizeResult
	err = json.Unmarshal(cleanBody, &res)

	if err != nil {
		fmt.Printf("Failed JSON: %s\n", cleanBody) // Printing the JSON causing the failure
		return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	// spew.Dump(sectionizedResp)

	sectionizedResp := shared.SectionizeResponse{
		SectionEnds: lib.GetSectionEnds(text, res.Sections),
	}

	byteRes, err = json.Marshal(sectionizedResp)
	if err != nil {
		return nil, err
	}

	return byteRes, nil
}
