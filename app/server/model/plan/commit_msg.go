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
	"plandex-server/utils"
	"time"

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

	var sysPrompt string
	var tools []openai.Tool
	var toolChoice *openai.ToolChoice

	if config.BaseModelConfig.PreferredModelOutputFormat == shared.ModelOutputFormatXml {
		sysPrompt = prompts.SysDescribeXml
	} else {
		sysPrompt = prompts.SysDescribe
		tools = []openai.Tool{
			{
				Type:     "function",
				Function: &prompts.DescribePlanFn,
			},
		}
		choice := openai.ToolChoice{
			Type: "function",
			Function: openai.ToolFunction{
				Name: prompts.DescribePlanFn.Name,
			},
		}
		toolChoice = &choice
	}

	messages := []types.ExtendedChatMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: sysPrompt,
				},
			},
		},
		{
			Role: openai.ChatMessageRoleAssistant,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: activePlan.CurrentReplyContent,
				},
			},
		},
	}

	numTokens := model.GetMessagesTokenEstimate(messages...) + model.TokensPerRequest

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		Auth: auth,
		Plan: plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  numTokens,
			OutputTokens: config.BaseModelConfig.MaxOutputTokens - numTokens,
			ModelName:    config.BaseModelConfig.ModelName,
		},
	})
	if apiErr != nil {
		return nil, apiErr
	}

	log.Println("Sending plan description model request")

	reqStarted := time.Now()
	req := types.ExtendedChatCompletionRequest{
		Model:       config.BaseModelConfig.ModelName,
		Messages:    messages,
		Temperature: config.Temperature,
		TopP:        config.TopP,
	}

	if tools != nil {
		req.Tools = tools
	}
	if toolChoice != nil {
		req.ToolChoice = *toolChoice
	}

	descResp, err := model.CreateChatCompletion(
		clients,
		&config,
		activePlan.Ctx,
		req,
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

	var commitMsg string

	if config.BaseModelConfig.PreferredModelOutputFormat == shared.ModelOutputFormatXml {
		content := descResp.Choices[0].Message.Content
		commitMsg = utils.GetXMLContent(content, "commitMsg")
		if commitMsg == "" {
			return nil, &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "No commitMsg tag found in XML response",
			}
		}
	} else {
		var descStrRes string
		for _, choice := range descResp.Choices {
			if len(choice.Message.ToolCalls) == 1 &&
				choice.Message.ToolCalls[0].Function.Name == prompts.DescribePlanFn.Name {
				fnCall := choice.Message.ToolCalls[0].Function
				descStrRes = fnCall.Arguments
				break
			}
		}

		if descStrRes == "" {
			fmt.Println("no describePlan function call found in response")
			spew.Dump(descResp)
			return nil, &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "No describePlan function call found in response. This usually means the model failed to generate a valid response.",
			}
		}

		var desc shared.ConvoMessageDescription
		err = json.Unmarshal([]byte(descStrRes), &desc)
		if err != nil {
			fmt.Printf("Error unmarshalling plan description response: %v\n", err)
			return nil, &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    fmt.Sprintf("error unmarshalling plan description response: %v", err),
			}
		}
		commitMsg = desc.CommitMsg
	}

	var inputTokens int
	var outputTokens int
	if descResp.Usage.CompletionTokens > 0 {
		inputTokens = descResp.Usage.PromptTokens
		outputTokens = descResp.Usage.CompletionTokens
	} else {
		inputTokens = numTokens
		outputTokens = shared.GetNumTokensEstimate(commitMsg)
	}

	var cachedTokens int
	if descResp.Usage.PromptTokensDetails != nil {
		cachedTokens = descResp.Usage.PromptTokensDetails.CachedTokens
	}

	log.Println("Sending DidSendModelRequest hook")

	go func() {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:    inputTokens,
				OutputTokens:   outputTokens,
				CachedTokens:   cachedTokens,
				ModelName:      config.BaseModelConfig.ModelName,
				ModelProvider:  config.BaseModelConfig.Provider,
				ModelPackName:  settings.ModelPack.Name,
				ModelRole:      shared.ModelRoleCommitMsg,
				Purpose:        "Generated commit message for suggested changes",
				GenerationId:   descResp.ID,
				PlanId:         planId,
				ModelStreamId:  state.modelStreamId,
				ConvoMessageId: state.replyId,

				RequestStartedAt: reqStarted,
				Streaming:        false,
				Req:              &req,
				Res:              &descResp,
				ModelConfig:      &config,
			},
		})

		if apiErr != nil {
			log.Printf("genPlanDescription - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

	return &db.ConvoMessageDescription{
		PlanId:    planId,
		CommitMsg: commitMsg,
	}, nil
}

func GenCommitMsgForPendingResults(auth *types.ServerAuth, plan *db.Plan, clients map[string]model.ClientInfo, settings *shared.PlanSettings, current *shared.CurrentPlanState, ctx context.Context) (string, error) {
	config := settings.ModelPack.CommitMsg

	s := ""

	num := 0
	for _, desc := range current.ConvoMessageDescriptions {
		if desc.WroteFiles && desc.DidBuild && len(desc.BuildPathsInvalidated) == 0 && desc.AppliedAt == nil {
			s += desc.CommitMsg + "\n"
			num++
		}
	}

	if num <= 1 {
		return s, nil
	}

	content := "Pending changes:\n\n" + s

	messages := []types.ExtendedChatMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: prompts.SysPendingResults,
				},
			},
		},
		{
			Role: openai.ChatMessageRoleUser,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: content,
				},
			},
		},
	}

	numTokens := model.GetMessagesTokenEstimate(messages...) + model.TokensPerRequest

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		Auth: auth,
		Plan: plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  numTokens,
			OutputTokens: config.BaseModelConfig.MaxOutputTokens - numTokens,
			ModelName:    config.BaseModelConfig.ModelName,
		},
	})
	if apiErr != nil {
		return "", errors.New(apiErr.Msg)
	}

	reqStarted := time.Now()
	req := types.ExtendedChatCompletionRequest{
		Model:       config.BaseModelConfig.ModelName,
		Messages:    messages,
		Temperature: config.Temperature,
		TopP:        config.TopP,
	}

	resp, err := model.CreateChatCompletion(
		clients,
		&config,
		ctx,
		req,
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

	var cachedTokens int
	if resp.Usage.PromptTokensDetails != nil {
		cachedTokens = resp.Usage.PromptTokensDetails.CachedTokens
	}

	_, apiErr = hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
		Auth: auth,
		Plan: plan,
		DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
			InputTokens:   inputTokens,
			OutputTokens:  outputTokens,
			CachedTokens:  cachedTokens,
			ModelName:     config.BaseModelConfig.ModelName,
			ModelProvider: config.BaseModelConfig.Provider,
			ModelPackName: settings.ModelPack.Name,
			ModelRole:     shared.ModelRoleCommitMsg,
			Purpose:       "Generated commit message for pending changes",
			GenerationId:  resp.ID,

			RequestStartedAt: reqStarted,
			Streaming:        false,
			Req:              &req,
			Res:              &resp,
			ModelConfig:      &config,
		},
	})

	if apiErr != nil {
		return "", errors.New(apiErr.Msg)
	}

	return commitMsg, nil
}
