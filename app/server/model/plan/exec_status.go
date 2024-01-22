package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"plandex-server/model"
	"plandex-server/model/prompts"

	"github.com/sashabaranov/go-openai"
)

type PlanExecStatus struct {
	NeedsInput bool `json:"needs_input"`
	Finished   bool `json:"finished"`
}

func ExecStatus(conversation []openai.ChatCompletionMessage, ctx context.Context) (*PlanExecStatus, error) {
	var res PlanExecStatus

	errCh := make(chan error, 2)

	go func() {
		needsInput, err := ExecStatusNeedsInput(conversation, ctx)
		if err != nil {
			errCh <- err
		}
		res.NeedsInput = needsInput
		errCh <- nil
	}()

	go func() {
		finished, err := ExecStatusIsFinished(conversation, ctx)
		if err != nil {
			errCh <- err
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

func ExecStatusIsFinished(conversation []openai.ChatCompletionMessage, ctx context.Context) (bool, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.GetExecStatusIsFinishedPrompt(conversation),
		},
	}

	resp, err := model.Client.CreateChatCompletion(
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
	var res PlanExecStatus

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

func ExecStatusNeedsInput(conversation []openai.ChatCompletionMessage, ctx context.Context) (bool, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.GetExecStatusNeedsInputPrompt(&conversation[len(conversation)-1]),
		},
	}

	messages = append(messages, conversation...)

	resp, err := model.Client.CreateChatCompletion(
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
	var res PlanExecStatus

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
