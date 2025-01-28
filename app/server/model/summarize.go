package model

import (
	"context"
	"errors"
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/model/prompts"
	"plandex-server/types"
	"strings"
	"time"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

type PlanSummaryParams struct {
	Auth                        *types.ServerAuth
	Plan                        *db.Plan
	ActivePlan                  *types.ActivePlan
	ModelPackName               string
	Conversation                []*openai.ChatCompletionMessage
	ConversationNumTokens       int
	LatestConvoMessageId        string
	LatestConvoMessageCreatedAt time.Time
	NumMessages                 int
}

func PlanSummary(clients map[string]*openai.Client, config shared.ModelRoleConfig, params PlanSummaryParams, ctx context.Context) (*db.ConvoSummary, error) {
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

	numTokens := shared.GetMessagesTokenEstimate(messages...) + shared.TokensPerRequest

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		Auth: params.Auth,
		Plan: params.Plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  numTokens,
			OutputTokens: shared.AvailableModelsByName[config.BaseModelConfig.ModelName].DefaultReservedOutputTokens,
			ModelName:    config.BaseModelConfig.ModelName,
		},
	})
	if apiErr != nil {
		return nil, errors.New(apiErr.Msg)
	}

	fmt.Println("summarizing messages:")
	// spew.Dump(messages)

	resp, err := CreateChatCompletionWithRetries(
		clients,
		&config,
		ctx,
		openai.ChatCompletionRequest{
			Model:       config.BaseModelConfig.ModelName,
			Messages:    messages,
			Temperature: config.Temperature,
			TopP:        config.TopP,
		},
	)

	if err != nil {
		fmt.Println("PlanSummary err:", err)

		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("plan summary - no choices in response. This usually means the model failed to generate a valid response.")
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
				InputTokens:   inputTokens,
				OutputTokens:  outputTokens,
				ModelName:     config.BaseModelConfig.ModelName,
				ModelProvider: config.BaseModelConfig.Provider,
				ModelPackName: params.ModelPackName,
				ModelRole:     shared.ModelRolePlanSummary,
				Purpose:       "Generated plan summary",
				GenerationId:  resp.ID,
			},
		})

		if apiErr != nil {
			log.Printf("PlanSummary - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

	if apiErr != nil {
		return nil, errors.New(apiErr.Msg)
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
