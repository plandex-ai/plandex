package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/types"

	"github.com/sashabaranov/go-openai"
)

func ExecStatus(client *openai.Client, message string, ctx context.Context) (*types.PlanExecStatus, error) {
	var res types.PlanExecStatus

	errCh := make(chan error, 2)

	go func() {
		needsInput, err := ExecStatusNeedsInput(client, message, ctx)
		if err != nil {
			errCh <- err
			return
		}
		res.NeedsInput = needsInput
		errCh <- nil
	}()

	go func() {
		finished, err := ExecStatusIsFinished(client, message, ctx)
		if err != nil {
			errCh <- err
			return
		}
		res.Finished = finished
		errCh <- nil
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			return nil, err
		}
	}

	return &res, nil
}

func ExecStatusIsFinished(client *openai.Client, message string, ctx context.Context) (bool, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.GetExecStatusIsFinishedPrompt(message),
		},
	}

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:          model.PlanExecStatusModel,
			Functions:      []openai.FunctionDefinition{prompts.PlanIsFinishedFn},
			Messages:       messages,
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
		},
	)

	if err != nil {
		fmt.Printf("Error during plan exec status check model call: %v\n", err)
		return false, err
	}

	var strRes string
	var res types.PlanExecStatus

	for _, choice := range resp.Choices {
		if choice.FinishReason == "function_call" &&
			choice.Message.FunctionCall != nil &&
			choice.Message.FunctionCall.Name == "planIsFinished" {
			fnCall := choice.Message.FunctionCall
			strRes = fnCall.Arguments
		}
	}

	if strRes == "" {
		fmt.Println("no planIsFinished function call found in response")
		return false, err
	}

	byteRes := []byte(strRes)

	err = json.Unmarshal(byteRes, &res)
	if err != nil {
		fmt.Printf("Error unmarshalling plan exec status response: %v\n", err)
		return false, err
	}

	return res.Finished, nil
}

func ExecStatusNeedsInput(client *openai.Client, message string, ctx context.Context) (bool, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.GetExecStatusNeedsInputPrompt(message),
		},
	}

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:          model.PlanExecStatusModel,
			Functions:      []openai.FunctionDefinition{prompts.PlanNeedsInputFn},
			Messages:       messages,
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
		},
	)

	if err != nil {
		fmt.Printf("Error during plan exec status check model call: %v\n", err)
		return false, err
	}

	var strRes string
	var res types.PlanExecStatus

	for _, choice := range resp.Choices {
		if choice.FinishReason == "function_call" &&
			choice.Message.FunctionCall != nil &&
			choice.Message.FunctionCall.Name == "planNeedsInput" {
			fnCall := choice.Message.FunctionCall
			strRes = fnCall.Arguments
		}
	}

	if strRes == "" {
		fmt.Println("no planNeedsInput function call found in response")
		return false, err
	}

	byteRes := []byte(strRes)

	err = json.Unmarshal(byteRes, &res)
	if err != nil {
		fmt.Printf("Error unmarshalling plan exec status response: %v\n", err)
		return false, err
	}

	return res.NeedsInput, nil
}
