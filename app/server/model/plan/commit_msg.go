package plan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) genPlanDescription() (*db.ConvoMessageDescription, error) {
	auth := state.auth
	plan := state.plan
	planId := plan.Id
	branch := state.branch
	settings := state.settings
	clients := state.clients
	config := settings.ModelPack.CommitMsg
	envVar := config.BaseModelConfig.ApiKeyEnvVar
	client := clients[envVar]

	activePlan := GetActivePlan(planId, branch)
	if activePlan == nil {
		return nil, fmt.Errorf("active plan not found")
	}

	var responseFormat *openai.ChatCompletionResponseFormat
	if config.BaseModelConfig.HasJsonResponseMode {
		responseFormat = &openai.ChatCompletionResponseFormat{Type: "json_object"}
	}

	numTokens := prompts.ExtraTokensPerRequest + (prompts.ExtraTokensPerMessage * 2) + prompts.SysDescribeNumTokens + activePlan.NumTokens

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		Auth: auth,
		Plan: plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  numTokens,
			OutputTokens: shared.AvailableModelsByName[config.BaseModelConfig.ModelName].DefaultReservedOutputTokens,
			ModelName:    config.BaseModelConfig.ModelName,
		},
	})
	if apiErr != nil {
		return nil, errors.New(apiErr.Msg)
	}

	log.Println("Sending plan description model request")

	descResp, err := model.CreateChatCompletionWithRetries(
		client,
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
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: prompts.SysDescribe,
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: activePlan.CurrentReplyContent,
				},
			},
			Temperature:    config.Temperature,
			TopP:           config.TopP,
			ResponseFormat: responseFormat,
		},
	)

	if err != nil {
		fmt.Printf("Error during plan description model call: %v\n", err)
		return nil, err
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
		outputTokens, err = shared.GetNumTokens(descStrRes)

		if err != nil {
			return nil, fmt.Errorf("error getting num tokens for content: %v", err)
		}
	}

	log.Println("Sending DidSendModelRequest hook")

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
			Purpose:       "Generated commit message for suggested changes",
		},
	})

	if apiErr != nil {
		return nil, errors.New(apiErr.Msg)
	}

	log.Println("DidSendModelRequest hook complete")

	if descStrRes == "" {
		fmt.Println("no describePlan function call found in response")
		return nil, fmt.Errorf("No describePlan function call found in response. This usually means the model failed to generate a valid response.")
	}

	descByteRes := []byte(descStrRes)

	err = json.Unmarshal(descByteRes, &desc)
	if err != nil {
		fmt.Printf("Error unmarshalling plan description response: %v\n", err)
		return nil, err
	}

	return &db.ConvoMessageDescription{
		PlanId:    planId,
		CommitMsg: desc.CommitMsg,
	}, nil
}

func GenCommitMsgForPendingResults(auth *types.ServerAuth, plan *db.Plan, client *openai.Client, settings *shared.PlanSettings, current *shared.CurrentPlanState, ctx context.Context) (string, error) {
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

	contentTokens, err := shared.GetNumTokens(content)

	if err != nil {
		return "", fmt.Errorf("error getting num tokens for content: %v", err)
	}

	numTokens := prompts.ExtraTokensPerRequest + (prompts.ExtraTokensPerMessage * 2) + prompts.SysPendingResultsNumTokens + contentTokens

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		Auth: auth,
		Plan: plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  numTokens,
			OutputTokens: shared.AvailableModelsByName[config.BaseModelConfig.ModelName].DefaultReservedOutputTokens,
			ModelName:    config.BaseModelConfig.ModelName,
		},
	})
	if apiErr != nil {
		return "", errors.New(apiErr.Msg)
	}

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

	resp, err := model.CreateChatCompletionWithRetries(
		client,
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
		outputTokens, err = shared.GetNumTokens(commitMsg)

		if err != nil {
			return "", fmt.Errorf("error getting num tokens for content: %v", err)
		}
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
		},
	})

	if apiErr != nil {
		return "", errors.New(apiErr.Msg)
	}

	return commitMsg, nil
}
