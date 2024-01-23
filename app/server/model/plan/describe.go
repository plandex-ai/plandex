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

func genPlanDescription(planId string, ctx context.Context) (*db.ConvoMessageDescription, error) {

	descResp, err := model.Client.CreateChatCompletion(
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
					Content: Active.Get(planId).CurrentReplyContent,
				},
			},
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
		},
	)

	if err != nil {
		fmt.Printf("Error during plan description model call: %v\n", err)
		return nil, err
	}

	var descStrRes string
	var desc shared.ConvoMessageDescription

	for _, choice := range descResp.Choices {
		if choice.FinishReason == "function_call" &&
			choice.Message.FunctionCall != nil &&
			choice.Message.FunctionCall.Name == "describePlan" {
			fnCall := choice.Message.FunctionCall
			descStrRes = fnCall.Arguments
		}
	}

	if descStrRes == "" {
		fmt.Println("no describePlan function call found in response")
		return nil, err
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
