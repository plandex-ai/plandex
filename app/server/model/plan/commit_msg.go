package plan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/types"

	shared "plandex-shared"

	"github.com/davecgh/go-spew/spew"
	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) genPlanDescription() (*db.ConvoMessageDescription, *shared.ApiError) {
	auth := state.auth
	plan := state.plan
	planId := plan.Id
	branch := state.branch
	settings := state.settings
	clients := state.clients
	config := settings.ModelPack.CommitMsg

	activePlan := GetActivePlan(planId, branch)
	if activePlan == nil {
		return nil, &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    fmt.Sprintf("active plan not found for plan %s and branch %s", planId, branch),
		}
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.SysDescribe,
		},
		{
			Role:    openai.ChatMessageRoleAssistant,
			Content: activePlan.CurrentReplyContent,
		},
	}

	numTokens := shared.GetMessagesTokenEstimate(messages...) + shared.TokensPerRequest

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		Auth: auth,
		Plan: plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  numTokens,
			OutputTokens: config.GetReservedOutputTokens(),
			ModelName:    config.BaseModelConfig.ModelName,
		},
	})
	if apiErr != nil {
		return nil, apiErr
	}

	log.Println("Sending plan description model request")

	descResp, err := model.CreateChatCompletionWithRetries(
		clients,
		&config,
		activePlan.Ctx,
		openai.ChatCompletionRequest{
			Model: config.BaseModelConfig.ModelName,
			Tools: []openai.Tool{
				{
					Type:     "function",
					Function: &prompts.DescribePlanFn,
				},
			},
			ToolChoice: openai.ToolChoice{
				Type: "function",
				Function: openai.ToolFunction{
					Name: prompts.DescribePlanFn.Name,
				},
			},
			Messages:    messages,
			Temperature: config.Temperature,
			TopP:        config.TopP,
		},
	)

	if err != nil {
		fmt.Printf("Error during plan description model call: %v\n", err)
		return nil, &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    fmt.Sprintf("error during plan description model call: %v", err),
		}
	}

	log.Println("Plan description model call complete")

	var descStrRes string
	var desc shared.ConvoMessageDescription

	for _, choice := range descResp.Choices {
		if len(choice.Message.ToolCalls) == 1 &&
			choice.Message.ToolCalls[0].Function.Name == prompts.DescribePlanFn.Name {
			fnCall := choice.Message.ToolCalls[0].Function
			descStrRes = fnCall.Arguments
			break
		}
	}

	var inputTokens int
	var outputTokens int
	if descResp.Usage.CompletionTokens > 0 {
		inputTokens = descResp.Usage.PromptTokens
		outputTokens = descResp.Usage.CompletionTokens
	} else {
		inputTokens = numTokens
		outputTokens = shared.GetNumTokensEstimate(descStrRes)
	}

	log.Println("Sending DidSendModelRequest hook")

	go func() {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:    inputTokens,
				OutputTokens:   outputTokens,
				ModelName:      config.BaseModelConfig.ModelName,
				ModelProvider:  config.BaseModelConfig.Provider,
				ModelPackName:  settings.ModelPack.Name,
				ModelRole:      shared.ModelRoleCommitMsg,
				Purpose:        "Generated commit message for suggested changes",
				GenerationId:   descResp.ID,
				PlanId:         planId,
				ModelStreamId:  activePlan.ModelStreamId,
				ConvoMessageId: state.replyId,
			},
		})

		if apiErr != nil {
			log.Printf("genPlanDescription - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

	if descStrRes == "" {
		fmt.Println("no describePlan function call found in response")

		spew.Dump(descResp)

		return nil, &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "No describePlan function call found in response. This usually means the model failed to generate a valid response.",
		}
	}

	descByteRes := []byte(descStrRes)

	err = json.Unmarshal(descByteRes, &desc)
	if err != nil {
		fmt.Printf("Error unmarshalling plan description response: %v\n", err)
		return nil, &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    fmt.Sprintf("error unmarshalling plan description response: %v", err),
		}
	}

	return &db.ConvoMessageDescription{
		PlanId:    planId,
		CommitMsg: desc.CommitMsg,
	}, nil
}

func GenCommitMsgForPendingResults(auth *types.ServerAuth, plan *db.Plan, clients map[string]model.ClientInfo, settings *shared.PlanSettings, current *shared.CurrentPlanState, ctx context.Context) (string, error) {
	config := settings.ModelPack.CommitMsg

	s := ""

	num := 0
	for _, desc := range current.ConvoMessageDescriptions {
		if desc.MadePlan && desc.DidBuild && len(desc.BuildPathsInvalidated) == 0 && desc.AppliedAt == nil {
			s += desc.CommitMsg + "\n"
			num++
		}
	}

	if num <= 1 {
		return s, nil
	}

	content := "Pending changes:\n\n" + s

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.SysPendingResults,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: content,
		},
	}

	numTokens := shared.GetMessagesTokenEstimate(messages...) + shared.TokensPerRequest

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		Auth: auth,
		Plan: plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  numTokens,
			OutputTokens: config.GetReservedOutputTokens(),
			ModelName:    config.BaseModelConfig.ModelName,
		},
	})
	if apiErr != nil {
		return "", errors.New(apiErr.Msg)
	}

	resp, err := model.CreateChatCompletionWithRetries(
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

		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from GPT")
	}

	commitMsg := resp.Choices[0].Message.Content

	var inputTokens int
	var outputTokens int
	if resp.Usage.CompletionTokens > 0 {
		inputTokens = resp.Usage.PromptTokens
		outputTokens = resp.Usage.CompletionTokens
	} else {
		inputTokens = numTokens
		outputTokens = shared.GetNumTokensEstimate(commitMsg)
	}

	_, apiErr = hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
		Auth: auth,
		Plan: plan,
		DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
			InputTokens:   inputTokens,
			OutputTokens:  outputTokens,
			ModelName:     config.BaseModelConfig.ModelName,
			ModelProvider: config.BaseModelConfig.Provider,
			ModelPackName: settings.ModelPack.Name,
			ModelRole:     shared.ModelRoleCommitMsg,
			Purpose:       "Generated commit message for pending changes",
			GenerationId:  resp.ID,
		},
	})

	if apiErr != nil {
		return "", errors.New(apiErr.Msg)
	}

	return commitMsg, nil
}
