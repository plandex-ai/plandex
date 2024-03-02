package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"plandex-server/db"
	"plandex-server/model"
	"plandex-server/model/prompts"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func genPlanDescription(client *openai.Client, config shared.TaskRoleConfig, planId, branch string, ctx context.Context) (*db.ConvoMessageDescription, error) {

	descResp, err := model.CreateChatCompletionWithRetries(
		client,
		ctx,
		openai.ChatCompletionRequest{
			Model: config.BaseModelConfig.ModelName,
			Tools: []openai.Tool{
				{
					Type:     "function",
					Function: prompts.DescribePlanFn,
				},
			},
			ToolChoice: openai.ToolChoice{
				Type: "function",
				Function: openai.ToolFunction{
					Name: prompts.DescribePlanFn.Name,
				},
			},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: prompts.SysDescribe,
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: GetActivePlan(planId, branch).CurrentReplyContent,
				},
			},
			Temperature:    config.Temperature,
			TopP:           config.TopP,
			ResponseFormat: config.OpenAIResponseFormat,
		},
	)

	if err != nil {
		fmt.Printf("Error during plan description model call: %v\n", err)
		return nil, err
	}

	var descStrRes string
	var desc shared.ConvoMessageDescription

	for _, choice := range descResp.Choices {
		if len(choice.Message.ToolCalls) == 1 &&
			choice.Message.ToolCalls[0].Function.Name == prompts.DescribePlanFn.Name {
			fnCall := choice.Message.ToolCalls[0].Function
			descStrRes = fnCall.Arguments
			break
		}
	}

	if descStrRes == "" {
		fmt.Println("no describePlan function call found in response")
		return nil, fmt.Errorf("no describePlan function call found in response")
	}

	descByteRes := []byte(descStrRes)

	err = json.Unmarshal(descByteRes, &desc)
	if err != nil {
		fmt.Printf("Error unmarshalling plan description response: %v\n", err)
		return nil, err
	}

	return &db.ConvoMessageDescription{
		PlanId:    planId,
		CommitMsg: desc.CommitMsg,
	}, nil
}
