package model

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/hooks"
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
	ActivePlan                  *types.ActivePlan
	ModelPackName               string
	Conversation                []*types.ExtendedChatMessage
	ConversationNumTokens       int
	LatestConvoMessageId        string
	LatestConvoMessageCreatedAt time.Time
	NumMessages                 int
}

func PlanSummary(clients map[string]ClientInfo, config shared.ModelRoleConfig, params PlanSummaryParams, ctx context.Context) (*db.ConvoSummary, *shared.ApiError) {
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

	numTokens := GetMessagesTokenEstimate(messages...) + TokensPerRequest

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		Auth: params.Auth,
		Plan: params.Plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  numTokens,
			OutputTokens: config.GetReservedOutputTokens(),
			ModelName:    config.BaseModelConfig.ModelName,
		},
	})
	if apiErr != nil {
		return nil, apiErr
	}

	fmt.Println("summarizing messages:")
	// spew.Dump(messages)

	reqStarted := time.Now()
	req := types.ExtendedChatCompletionRequest{
		Model:       config.BaseModelConfig.ModelName,
		Messages:    messages,
		Temperature: config.Temperature,
		TopP:        config.TopP,
	}

	resp, err := CreateChatCompletionWithRetries(
		clients,
		&config,
		ctx,
		req,
	)

	if err != nil {
		fmt.Println("PlanSummary err:", err)

		return nil, &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    fmt.Sprintf("error generating plan summary: %v", err),
		}
	}

	if len(resp.Choices) == 0 {
		return nil, &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "plan summary - no choices in response. This usually means the model failed to generate a valid response.",
		}
	}

	content := resp.Choices[0].Message.Content

	var inputTokens int
	var outputTokens int
	if resp.Usage.CompletionTokens > 0 {
		inputTokens = resp.Usage.PromptTokens
		outputTokens = resp.Usage.CompletionTokens
	} else {
		inputTokens = numTokens
		outputTokens = shared.GetNumTokensEstimate(content)
	}

	go func() {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: params.Auth,
			Plan: params.Plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:    inputTokens,
				OutputTokens:   outputTokens,
				ModelName:      config.BaseModelConfig.ModelName,
				ModelProvider:  config.BaseModelConfig.Provider,
				ModelPackName:  params.ModelPackName,
				ModelRole:      shared.ModelRolePlanSummary,
				Purpose:        "Generated plan summary",
				GenerationId:   resp.ID,
				PlanId:         params.Plan.Id,
				ModelStreamId:  params.ActivePlan.ModelStreamId,
				ConvoMessageId: params.LatestConvoMessageId,
				BuildId:        params.Plan.Id,

				RequestStartedAt: reqStarted,
				Streaming:        false,
				Req:              &req,
				Res:              &resp,
				ModelConfig:      &config,
			},
		})

		if apiErr != nil {
			log.Printf("PlanSummary - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

	if apiErr != nil {
		return nil, apiErr
	}

	// log.Println("Plan summary content:")
	// log.Println(content)

	summary := content
	if !strings.HasPrefix(summary, "## Summary of the plan so far:") {
		summary = "## Summary of the plan so far:\n\n" + summary
	}

	return &db.ConvoSummary{
		OrgId:                       params.Auth.OrgId,
		PlanId:                      params.Plan.Id,
		Summary:                     summary,
		Tokens:                      resp.Usage.CompletionTokens,
		LatestConvoMessageId:        params.LatestConvoMessageId,
		LatestConvoMessageCreatedAt: params.LatestConvoMessageCreatedAt,
		NumMessages:                 params.NumMessages,
	}, nil

}
