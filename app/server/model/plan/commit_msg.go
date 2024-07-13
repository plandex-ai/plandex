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

func genPlanDescription(client *openai.Client, config shared.ModelRoleConfig, planId, branch string, ctx context.Context) (*db.ConvoMessageDescription, error) {
	activePlan := GetActivePlan(planId, branch)
	if activePlan == nil {
		return nil, fmt.Errorf("active plan not found")
	}

	var responseFormat *openai.ChatCompletionResponseFormat
	if config.BaseModelConfig.HasJsonResponseMode {
		responseFormat = &openai.ChatCompletionResponseFormat{Type: "json_object"}
	}

	descResp, err := model.CreateChatCompletionWithRetries(
		client,
		ctx,
		openai.ChatCompletionRequest{
			Model: config.BaseModelConfig.ModelName,
			Tools: []openai.Tool{
				{
					Type:     "function",
					Function: &prompts.DescribePlanFn,
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
					Content: activePlan.CurrentReplyContent,
				},
			},
			Temperature:    config.Temperature,
			TopP:           config.TopP,
			ResponseFormat: responseFormat,
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

func GenCommitMsgForPendingResults(client *openai.Client, config shared.ModelRoleConfig, current *shared.CurrentPlanState, ctx context.Context) (string, error) {
	s := ""

	num := 0
	for _, desc := range current.ConvoMessageDescriptions {
		if desc.MadePlan && desc.DidBuild && len(desc.BuildPathsInvalidated) == 0 && desc.AppliedAt == nil {
			s += desc.CommitMsg + "\n"
			num++
		}
	}

	if num <= 1 {
		return s, nil
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.SysPendingResults,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "Pending changes:\n\n" + s,
		},
	}

	resp, err := model.CreateChatCompletionWithRetries(
		client,
		ctx,
		openai.ChatCompletionRequest{
			Model:       config.BaseModelConfig.ModelName,
			Messages:    messages,
			Temperature: config.Temperature,
			TopP:        config.TopP,
		},
	)

	if err != nil {
		fmt.Println("PlanSummary err:", err)

		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from GPT")
	}

	content := resp.Choices[0].Message.Content

	return content, nil
}
