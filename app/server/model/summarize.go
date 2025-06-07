package model

import (
	"context"
	"fmt"
	"net/http"
	"plandex-server/db"
	"plandex-server/model/prompts"
	"plandex-server/types"
	"strings"
	"time"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

type PlanSummaryParams struct {
	Auth                        *types.ServerAuth
	Plan                        *db.Plan
	ModelStreamId               string
	ModelPackName               string
	Conversation                []*types.ExtendedChatMessage
	ConversationNumTokens       int
	LatestConvoMessageId        string
	LatestConvoMessageCreatedAt time.Time
	NumMessages                 int
	SessionId                   string
}

func PlanSummary(clients map[string]ClientInfo, authVars map[string]string, localProvider shared.ModelProvider, config shared.ModelRoleConfig, params PlanSummaryParams, ctx context.Context) (*db.ConvoSummary, *shared.ApiError) {
	messages := []types.ExtendedChatMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: prompts.Identity,
				},
			},
		},
	}

	for _, message := range params.Conversation {
		messages = append(messages, *message)
	}

	messages = append(messages, types.ExtendedChatMessage{
		Role: openai.ChatMessageRoleUser,
		Content: []types.ExtendedChatMessagePart{
			{
				Type: openai.ChatMessagePartTypeText,
				Text: prompts.PlanSummary,
			},
		},
	})

	modelRes, err := ModelRequest(ctx, ModelRequestParams{
		Clients:        clients,
		Auth:           params.Auth,
		AuthVars:       authVars,
		Plan:           params.Plan,
		ModelConfig:    &config,
		Purpose:        "Conversation summary",
		ConvoMessageId: params.LatestConvoMessageId,
		ModelStreamId:  params.ModelStreamId,
		Messages:       messages,
		SessionId:      params.SessionId,
		LocalProvider:  localProvider,
	})

	if err != nil {
		return nil, &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    fmt.Sprintf("error generating plan summary: %v", err),
		}
	}

	summary := modelRes.Content
	if !strings.HasPrefix(summary, "## Summary of the plan so far:") {
		summary = "## Summary of the plan so far:\n\n" + summary
	}

	var tokens int
	if modelRes.Usage != nil {
		tokens = modelRes.Usage.CompletionTokens
	}

	return &db.ConvoSummary{
		OrgId:                       params.Auth.OrgId,
		PlanId:                      params.Plan.Id,
		Summary:                     summary,
		Tokens:                      tokens,
		LatestConvoMessageId:        params.LatestConvoMessageId,
		LatestConvoMessageCreatedAt: params.LatestConvoMessageCreatedAt,
		NumMessages:                 params.NumMessages,
	}, nil

}
