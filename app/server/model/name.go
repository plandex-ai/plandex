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
			Model:          NameModel,
			Functions:      []openai.FunctionDefinition{prompts.PlanNameFn},
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
		if choice.FinishReason == "function_call" &&
			choice.Message.FunctionCall != nil &&
			choice.Message.FunctionCall.Name == "namePlan" {
			fnCall := choice.Message.FunctionCall
			res = fnCall.Arguments
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
