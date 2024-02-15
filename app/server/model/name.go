package model

import (
	"context"
	"encoding/json"
	"fmt"
	"plandex-server/model/prompts"

	"github.com/sashabaranov/go-openai"
)

func GenPlanName(client *openai.Client, planContent string) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.SysPlanName,
		},
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompts.GetPlanNamePrompt(planContent),
	})

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: NameModel,
			Tools: []openai.Tool{
				{
					Type:     "function",
					Function: prompts.PlanNameFn,
				},
			},
			ToolChoice: openai.ToolChoice{
				Type: "function",
				Function: openai.ToolFunction{
					Name: prompts.PlanNameFn.Name,
				},
			},

			Messages:       messages,
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
		},
	)

	var res string
	var nameRes prompts.PlanNameRes

	if err != nil {
		fmt.Printf("Error during plan name model call: %v\n", err)
		return "", err
	}

	for _, choice := range resp.Choices {
		if len(choice.Message.ToolCalls) == 1 &&
			choice.Message.ToolCalls[0].Function.Name == prompts.PlanNameFn.Name {
			fnCall := choice.Message.ToolCalls[0].Function
			res = fnCall.Arguments
			break
		}
	}

	if res == "" {
		fmt.Println("no namePlan function call found in response")
		return "", err
	}

	bytes := []byte(res)

	err = json.Unmarshal(bytes, &nameRes)
	if err != nil {
		fmt.Printf("Error unmarshalling plan description response: %v\n", err)
		return "", err
	}

	return nameRes.PlanName, nil

}
