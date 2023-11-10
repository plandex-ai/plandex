package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"plandex-server/model"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func genPlanDescriptionJson(proposalId string, ctx context.Context) (*shared.PlanDescription, error) {
	proposal := proposals.Get(proposalId)

	planDescResp, err := model.Client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: model.CommitMsgModel,
			Functions: []openai.FunctionDefinition{{
				Name: "describePlan",
				Parameters: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"commitMsg": {
							Type:        jsonschema.String,
							Description: "A good, succinct commit message for the changes proposed.",
						},
					},
					Required: []string{"commitMsg"},
				},
			}},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an AI parser. You turn an AI's plan for a programming task into a structured description. You call the 'describePlan' function with the 'commitMsg' argument. Only call the 'describePlan' function in your response. Don't call any other function.",
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
		return &planDesc, err
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
		return &planDesc, err
	}

	planDescByteRes := []byte(planDescStrRes)

	err = json.Unmarshal(planDescByteRes, &planDesc)
	if err != nil {
		fmt.Printf("Error unmarshalling plan description response: %v\n", err)
		return nil, err
	}

	return &planDesc, nil
}
