package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"plandex-server/model"
	"plandex-server/model/prompts"

	"github.com/sashabaranov/go-openai"
)

func ExecStatusShouldContinue(client *openai.Client, message string, ctx context.Context) (bool, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.GetExecStatusShouldContinue(message), // Ensure this function is correctly defined in your package
		},
	}

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:          model.PlanExecStatusModel,
			Functions:      []openai.FunctionDefinition{prompts.ShouldAutoContinueFn},
			Messages:       messages,
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
		},
	)

	if err != nil {
		fmt.Printf("Error during plan exec status check model call: %v\n", err)
		return false, err
	}

	var strRes string
	var res struct {
		ShouldContinue bool `json:"shouldContinue"`
	}

	for _, choice := range resp.Choices {
		if choice.FinishReason == "function_call" &&
			choice.Message.FunctionCall != nil &&
			choice.Message.FunctionCall.Name == "shouldAutoContinue" {
			fnCall := choice.Message.FunctionCall
			strRes = fnCall.Arguments
		}
	}

	if strRes == "" {
		fmt.Println("No shouldAutoContinue function call found in response")
		return false, fmt.Errorf("no shouldAutoContinue function call found in response")
	}

	err = json.Unmarshal([]byte(strRes), &res)
	if err != nil {
		fmt.Printf("Error unmarshalling plan exec status response: %v\n", err)
		return false, err
	}

	return res.ShouldContinue, nil
}
