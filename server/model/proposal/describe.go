package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"plandex-server/model"
	"plandex-server/model/prompts"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func genPlanDescriptionJson(proposalId string, ctx context.Context) (*shared.PlanDescription, error) {
	proposal := proposals.Get(proposalId)

	planDescResp, err := model.Client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:     model.CommitMsgModel,
			Functions: []openai.FunctionDefinition{prompts.DescribePlanFn},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: prompts.SysDescribe,
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: proposal.Content,
				},
			},
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
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
