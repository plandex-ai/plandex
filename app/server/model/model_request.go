package model

import (
	"context"
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/types"
	shared "plandex-shared"
	"time"

	"github.com/sashabaranov/go-openai"
)

type ModelRequestParams struct {
	Clients     map[string]ClientInfo
	Auth        *types.ServerAuth
	Plan        *db.Plan
	ModelConfig *shared.ModelRoleConfig
	Purpose     string

	Messages   []types.ExtendedChatMessage
	Prediction string
	Stop       []string
	Tools      []openai.Tool
	ToolChoice *openai.ToolChoice

	EstimatedOutputTokens int // optional

	ModelStreamId  string
	ConvoMessageId string
	BuildId        string
	ModelPackName  string

	BeforeReq func()
	AfterReq  func()

	OnStream func(string, string) bool

	WillCacheNumTokens int
}

func ModelRequest(
	ctx context.Context,
	params ModelRequestParams,
) (*types.ModelResponse, error) {
	clients := params.Clients
	auth := params.Auth
	plan := params.Plan
	messages := params.Messages
	prediction := params.Prediction
	stop := params.Stop
	tools := params.Tools
	toolChoice := params.ToolChoice
	modelConfig := params.ModelConfig
	modelStreamId := params.ModelStreamId
	convoMessageId := params.ConvoMessageId
	buildId := params.BuildId
	modelPackName := params.ModelPackName
	purpose := params.Purpose

	if purpose == "" {
		return nil, fmt.Errorf("purpose is required")
	}

	inputTokensEstimate := GetMessagesTokenEstimate(messages...) + TokensPerRequest

	config := modelConfig.GetRoleForInputTokens(inputTokensEstimate)
	modelConfig = &config

	if params.EstimatedOutputTokens != 0 {
		config = modelConfig.GetRoleForOutputTokens(params.EstimatedOutputTokens)
		modelConfig = &config
	}

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		Auth: auth,
		Plan: plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  inputTokensEstimate,
			OutputTokens: modelConfig.BaseModelConfig.MaxOutputTokens - inputTokensEstimate,
			ModelName:    modelConfig.BaseModelConfig.ModelName,
		},
	})

	if apiErr != nil {
		return nil, apiErr
	}

	if params.BeforeReq != nil {
		params.BeforeReq()
	}

	reqStarted := time.Now()

	req := types.ExtendedChatCompletionRequest{
		Model:       modelConfig.BaseModelConfig.ModelName,
		Messages:    messages,
		Temperature: modelConfig.Temperature,
		TopP:        modelConfig.TopP,
		Stop:        stop,
		Tools:       tools,
		ToolChoice:  toolChoice,
	}

	if prediction != "" {
		req.Prediction = &types.OpenAIPrediction{
			Type:    "content",
			Content: prediction,
		}
	}

	res, err := CreateChatCompletionWithInternalStream(clients, modelConfig, ctx, req, params.OnStream, reqStarted)

	if err != nil {
		return nil, err
	}

	if params.AfterReq != nil {
		params.AfterReq()
	}

	// log.Printf("\n\n**\n\nModel response: %s\n\n**\n\n", res.Content)

	var inputTokens int
	var outputTokens int
	var cachedTokens int

	if res.Usage != nil {
		if res.Usage.PromptTokensDetails != nil {
			cachedTokens = res.Usage.PromptTokensDetails.CachedTokens
		}
		inputTokens = res.Usage.PromptTokens
		outputTokens = res.Usage.CompletionTokens
	} else {
		inputTokens = inputTokensEstimate
		outputTokens = shared.GetNumTokensEstimate(res.Content)

		if params.WillCacheNumTokens > 0 {
			cachedTokens = params.WillCacheNumTokens
		}
	}

	go func() {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:    inputTokens,
				OutputTokens:   outputTokens,
				CachedTokens:   cachedTokens,
				ModelName:      modelConfig.BaseModelConfig.ModelName,
				ModelProvider:  modelConfig.BaseModelConfig.Provider,
				ModelPackName:  modelPackName,
				ModelRole:      shared.ModelRoleBuilder,
				Purpose:        purpose,
				GenerationId:   res.GenerationId,
				PlanId:         plan.Id,
				ModelStreamId:  modelStreamId,
				ConvoMessageId: convoMessageId,
				BuildId:        buildId,

				RequestStartedAt: reqStarted,
				Streaming:        true,
				Req:              &req,
				StreamResult:     res.Content,
				ModelConfig:      modelConfig,
				FirstTokenAt:     res.FirstTokenAt,
			},
		})

		if apiErr != nil {
			log.Printf("buildWholeFile - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

	return res, nil
}
