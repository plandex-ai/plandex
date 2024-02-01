package model

import (
	"context"
	"fmt"
	"plandex-server/db"
	"plandex-server/model/prompts"
	"time"

	"github.com/sashabaranov/go-openai"
)

type PlanSummaryParams struct {
	Conversation                []*openai.ChatCompletionMessage
	LatestConvoMessageId        string
	LatestConvoMessageCreatedAt time.Time
	NumMessages                 int
	OrgId                       string
	PlanId                      string
}

func PlanSummary(client *openai.Client, params PlanSummaryParams) (*db.ConvoSummary, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.Identity,
		},
	}

	for _, message := range params.Conversation {
		messages = append(messages, *message)
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompts.PlanSummary,
	})

	// fmt.Println("summarizing messages:")
	// spew.Dump(messages)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    PlanSummaryModel,
			Messages: messages,
		},
	)

	if err != nil {
		fmt.Println("PlanSummary err:", err)

		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from GPT")
	}

	content := resp.Choices[0].Message.Content

	return &db.ConvoSummary{
		OrgId:                       params.OrgId,
		PlanId:                      params.PlanId,
		Summary:                     content,
		Tokens:                      resp.Usage.CompletionTokens,
		LatestConvoMessageId:        params.LatestConvoMessageId,
		LatestConvoMessageCreatedAt: params.LatestConvoMessageCreatedAt,
		NumMessages:                 params.NumMessages,
	}, nil

}
