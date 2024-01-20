package model

import (
	"context"
	"fmt"
	"plandex-server/model/prompts"
	"time"

	"github.com/sashabaranov/go-openai"
)

type PlanFinishedParams struct {
	Conversation []openai.ChatCompletionMessage
}

func PlanFinished(params PlanFinishedParams) (bool, error) {
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
		return false, err
	}

	if len(resp.Choices) == 0 {
		return false, fmt.Errorf("no response from GPT")
	}

	content := resp.Choices[0].Message.Content

	// Here, we assume that if the AI assistant says "finished", the plan is finished.
	// You might want to adjust this according to your needs.
	isFinished := content == "finished"

	return isFinished, nil
}
