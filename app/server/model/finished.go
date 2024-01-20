package model

import (
	"context"
	"encoding/json"
	"fmt"
	"plandex-server/model/prompts"

	"github.com/sashabaranov/go-openai"
)

type PlanFinishedParams struct {
	Conversation []openai.ChatCompletionMessage
}

type PlanFinishedRes struct {
	Reasoning string `json:"reasoning"`
	Finished  bool   `json:"finished"`
}

func PlanFinished(params PlanFinishedParams) (PlanFinishedRes, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.SysFinished,
		},
	}

	messages = append(messages, params.Conversation...)

	resp, err := Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    PlanSummaryModel,
			Messages: messages,
		},
	)

	if err != nil {
		fmt.Println("PlanFinished err:", err)
		return PlanFinishedRes{}, err
	}

	if len(resp.Choices) == 0 {
		return PlanFinishedRes{}, fmt.Errorf("no response from GPT")
	}

	content := resp.Choices[0].Message.Content

	// Here, we assume that if the AI assistant says "finished", the plan is finished.
	// You might want to adjust this according to your needs.
	isFinished := content == "finished"

	return PlanFinishedRes{
		Reasoning: content,
		Finished:  isFinished,
	}, nil
}
