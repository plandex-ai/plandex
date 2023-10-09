package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"plandex-server/model"
	"strings"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func genPlanDescriptionJson(proposalId string, ctx context.Context) (*shared.PlanDescription, string, error) {
	proposal := proposals.Get(proposalId)

	planDescResp, err := model.Client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Functions: []openai.FunctionDefinition{{
				Name: "describePlan",
				Parameters: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"madePlan": {
							Type:        jsonschema.Boolean,
							Description: "Whether a plan was made that includes file paths and code. Should be false if 'subtasks' is true or 'files' is empty",
						},
						"subtasks": {
							Type:        jsonschema.Boolean,
							Description: "Whether the task is too large to be completed in a single response and was broken up into subtasks. If 'madePlan' is true, this should be false",
						},
						"files": {
							Type:        jsonschema.Array,
							Description: "An array of file paths that precede code blocks in the plan. If 'madePlan' is false or 'subtasks' is true, this should be an empty array.",
							Items: &jsonschema.Definition{
								Type: jsonschema.String,
							},
						},
						"commitMsg": {
							Type:        jsonschema.String,
							Description: "A good commit message for the changes proposed. If 'madePlan' is false or 'subtasks' is true, this should be an empty string",
						},
						// "hasExec": {
						// 	Type:        jsonschema.Boolean,
						// 	Description: "Whether the plan includes any 'exec' blocks that include shell commands. If 'madePlan' is false, this should be false",
						// },
					},
					Required: []string{"madePlan", "subtasks", "commitMsg", "files" /*"hasExec"*/},
				},
			}},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an AI parser. You turn an AI's plan for a programming task into a structured description. You call the 'describePlan' function with arguments 'madePlan', 'subtasks', 'commitMsg', 'files', and 'hasExec'. Only call the 'describePlan' function in your response. Don't call any other function.",
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: proposal.Content,
				},
			},
		},
	)

	var planDescStrRes string
	var planDesc shared.PlanDescription

	if err != nil {
		fmt.Printf("Error during plan description model call: %v\n", err)
		planDesc = shared.PlanDescription{}
		bytes, err := json.Marshal(planDesc)
		return &planDesc, string(bytes), err
	}

	for _, choice := range planDescResp.Choices {
		if choice.FinishReason == "function_call" &&
			choice.Message.FunctionCall != nil &&
			choice.Message.FunctionCall.Name == "describePlan" {
			fnCall := choice.Message.FunctionCall
			planDescStrRes = fnCall.Arguments
		}
	}

	if planDescStrRes == "" {
		fmt.Println("no describePlan function call found in response")
		planDesc = shared.PlanDescription{}
		bytes, err := json.Marshal(planDesc)
		return &planDesc, string(bytes), err
	}

	planDescByteRes := []byte(planDescStrRes)

	err = json.Unmarshal(planDescByteRes, &planDesc)
	if err != nil {
		fmt.Printf("Error unmarshalling plan description response: %v\n", err)
		return nil, "", err
	}

	for i, filePath := range planDesc.Files {
		filePath = strings.TrimSpace(filePath)
		filePath = strings.TrimPrefix(filePath, "-")
		filePath = strings.TrimSpace(filePath)
		planDesc.Files[i] = filePath
		fmt.Println("file path: " + filePath)
	}

	bytes, err := json.Marshal(planDesc)
	if err != nil {
		fmt.Printf("Error marshalling plan description: %v\n", err)
		return nil, "", err
	}

	return &planDesc, string(bytes), nil
}
