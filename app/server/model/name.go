package model

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/model/prompts"
	"plandex-server/types"
	"plandex-server/utils"
	"time"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

func GenPlanName(
	auth *types.ServerAuth,
	plan *db.Plan,
	settings *shared.PlanSettings,
	clients map[string]ClientInfo,
	planContent string,
	ctx context.Context,
) (string, error) {
	config := settings.ModelPack.Namer

	var tools []openai.Tool
	var toolChoice *openai.ToolChoice

	var sysPrompt string
	if config.BaseModelConfig.PreferredModelOutputFormat == shared.ModelOutputFormatXml {
		sysPrompt = prompts.SysPlanNameXml
	} else {
		sysPrompt = prompts.SysPlanName
		tools = []openai.Tool{
			{
				Type:     "function",
				Function: &prompts.PlanNameFn,
			},
		}
		choice := openai.ToolChoice{
			Type: "function",
			Function: openai.ToolFunction{
				Name: prompts.PlanNameFn.Name,
			},
		}
		toolChoice = &choice
	}

	content := prompts.GetPlanNamePrompt(sysPrompt, planContent)

	messages := []types.ExtendedChatMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: content,
				},
			},
		},
	}

	numTokens := GetMessagesTokenEstimate(messages...) + TokensPerRequest

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
		return "", fmt.Errorf("error executing hook: %v", apiErr)
	}

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

	resp, err := CreateChatCompletion(
		clients,
		&config,
		ctx,
		req,
	)

	var planName string

	if err != nil {
		fmt.Printf("Error during plan name model call: %v\n", err)
		return "", err
	}

	if config.BaseModelConfig.PreferredModelOutputFormat == shared.ModelOutputFormatXml {
		content := resp.Choices[0].Message.Content
		planName = utils.GetXMLContent(content, "planName")
		if planName == "" {
			return "", fmt.Errorf("No planName tag found in XML response")
		}
	} else {
		var res string
		for _, choice := range resp.Choices {
			if len(choice.Message.ToolCalls) == 1 &&
				choice.Message.ToolCalls[0].Function.Name == prompts.PlanNameFn.Name {
				fnCall := choice.Message.ToolCalls[0].Function
				res = fnCall.Arguments
				break
			}
		}

		if res == "" {
			fmt.Println("no namePlan function call found in response")
			return "", fmt.Errorf("No namePlan function call found in response. This usually means the model failed to generate a valid response.")
		}

		var nameRes prompts.PlanNameRes
		bytes := []byte(res)
		err = json.Unmarshal(bytes, &nameRes)
		if err != nil {
			fmt.Printf("Error unmarshalling plan description response: %v\n", err)
			return "", err
		}
		planName = nameRes.PlanName
	}

	var inputTokens int
	var outputTokens int
	if resp.Usage.CompletionTokens > 0 {
		inputTokens = resp.Usage.PromptTokens
		outputTokens = resp.Usage.CompletionTokens
	} else {
		inputTokens = numTokens
		outputTokens = shared.GetNumTokensEstimate(planName)
	}

	var cachedTokens int
	if resp.Usage.PromptTokensDetails != nil {
		cachedTokens = resp.Usage.PromptTokensDetails.CachedTokens
	}

	go func() {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:      inputTokens,
				OutputTokens:     outputTokens,
				CachedTokens:     cachedTokens,
				ModelName:        config.BaseModelConfig.ModelName,
				ModelProvider:    config.BaseModelConfig.Provider,
				ModelPackName:    settings.ModelPack.Name,
				ModelRole:        shared.ModelRolePlanSummary,
				Purpose:          "Generated plan name",
				GenerationId:     resp.ID,
				PlanId:           plan.Id,
				RequestStartedAt: reqStarted,
				Streaming:        false,
				Req:              &req,
				Res:              &resp,
				ModelConfig:      &config,
			},
		})

		if apiErr != nil {
			log.Printf("GenPlanName - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

	return planName, nil
}

func GenPipedDataName(
	ctx context.Context,
	auth *types.ServerAuth,
	plan *db.Plan,
	settings *shared.PlanSettings,
	clients map[string]ClientInfo,
	pipedContent string,
) (string, error) {
	config := settings.ModelPack.Namer

	var sysPrompt string
	var tools []openai.Tool
	var toolChoice *openai.ToolChoice

	if config.BaseModelConfig.PreferredModelOutputFormat == shared.ModelOutputFormatXml {
		sysPrompt = prompts.SysPipedDataNameXml
	} else {
		sysPrompt = prompts.SysPipedDataName
		tools = []openai.Tool{
			{
				Type:     "function",
				Function: &prompts.PipedDataNameFn,
			},
		}
		choice := openai.ToolChoice{
			Type: "function",
			Function: openai.ToolFunction{
				Name: prompts.PipedDataNameFn.Name,
			},
		}
		toolChoice = &choice
	}

	content := prompts.GetPipedDataNamePrompt(sysPrompt, pipedContent)

	messages := []types.ExtendedChatMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: content,
				},
			},
		},
	}

	numTokens := GetMessagesTokenEstimate(messages...) + TokensPerRequest

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
		return "", fmt.Errorf("error executing hook: %v", apiErr)
	}

	log.Println("calling piped data name model")

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

	resp, err := CreateChatCompletion(
		clients,
		&config,
		ctx,
		req,
	)

	var name string

	if err != nil {
		fmt.Printf("Error during piped data name model call: %v\n", err)
		return "", err
	}

	if config.BaseModelConfig.PreferredModelOutputFormat == shared.ModelOutputFormatXml {
		content := resp.Choices[0].Message.Content
		name = utils.GetXMLContent(content, "name")
		if name == "" {
			return "", fmt.Errorf("No name tag found in XML response")
		}
	} else {
		var res string
		for _, choice := range resp.Choices {
			if len(choice.Message.ToolCalls) == 1 &&
				choice.Message.ToolCalls[0].Function.Name == prompts.PipedDataNameFn.Name {
				fnCall := choice.Message.ToolCalls[0].Function
				res = fnCall.Arguments
				break
			}
		}

		if res == "" {
			fmt.Println("no namePipedData function call found in response")
			return "", fmt.Errorf("No namePipedData function call found in response. This usually means the model failed to generate a valid response.")
		}

		var nameRes prompts.PipedDataNameRes
		bytes := []byte(res)
		err = json.Unmarshal(bytes, &nameRes)
		if err != nil {
			fmt.Printf("Error unmarshalling piped data name response: %v\n", err)
			return "", err
		}
		name = nameRes.Name
	}

	var inputTokens int
	var outputTokens int
	if resp.Usage.CompletionTokens > 0 {
		inputTokens = resp.Usage.PromptTokens
		outputTokens = resp.Usage.CompletionTokens
	} else {
		inputTokens = numTokens
		outputTokens = shared.GetNumTokensEstimate(name)
	}

	var cachedTokens int
	if resp.Usage.PromptTokensDetails != nil {
		cachedTokens = resp.Usage.PromptTokensDetails.CachedTokens
	}

	go func() {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:      inputTokens,
				OutputTokens:     outputTokens,
				CachedTokens:     cachedTokens,
				ModelName:        config.BaseModelConfig.ModelName,
				ModelProvider:    config.BaseModelConfig.Provider,
				ModelPackName:    settings.ModelPack.Name,
				ModelRole:        shared.ModelRolePlanSummary,
				Purpose:          "Generated name for data piped into context",
				GenerationId:     resp.ID,
				PlanId:           plan.Id,
				RequestStartedAt: reqStarted,
				Streaming:        false,
				Req:              &req,
				Res:              &resp,
				ModelConfig:      &config,
			},
		})

		if apiErr != nil {
			log.Printf("GenPipedDataName - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

	return name, nil
}

func GenNoteName(
	ctx context.Context,
	auth *types.ServerAuth,
	plan *db.Plan,
	settings *shared.PlanSettings,
	clients map[string]ClientInfo,
	note string,
) (string, error) {
	config := settings.ModelPack.Namer

	var sysPrompt string
	var tools []openai.Tool
	var toolChoice *openai.ToolChoice

	if config.BaseModelConfig.PreferredModelOutputFormat == shared.ModelOutputFormatXml {
		sysPrompt = prompts.SysNoteNameXml
	} else {
		sysPrompt = prompts.SysNoteName
		tools = []openai.Tool{
			{
				Type:     "function",
				Function: &prompts.NoteNameFn,
			},
		}
		choice := openai.ToolChoice{
			Type: "function",
			Function: openai.ToolFunction{
				Name: prompts.NoteNameFn.Name,
			},
		}
		toolChoice = &choice
	}

	content := prompts.GetNoteNamePrompt(sysPrompt, note)

	messages := []types.ExtendedChatMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: content,
				},
			},
		},
	}

	numTokens := GetMessagesTokenEstimate(messages...) + TokensPerRequest

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
		return "", fmt.Errorf("error executing hook: %v", apiErr)
	}

	log.Println("calling note name model")

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

	resp, err := CreateChatCompletion(
		clients,
		&config,
		ctx,
		req,
	)

	var name string

	if err != nil {
		fmt.Printf("Error during note name model call: %v\n", err)
		return "", err
	}

	if config.BaseModelConfig.PreferredModelOutputFormat == shared.ModelOutputFormatXml {
		content := resp.Choices[0].Message.Content
		name = utils.GetXMLContent(content, "name")
		if name == "" {
			return "", fmt.Errorf("No name tag found in XML response")
		}
	} else {
		var res string
		for _, choice := range resp.Choices {
			if len(choice.Message.ToolCalls) == 1 &&
				choice.Message.ToolCalls[0].Function.Name == prompts.NoteNameFn.Name {
				fnCall := choice.Message.ToolCalls[0].Function
				res = fnCall.Arguments
				break
			}
		}

		if res == "" {
			fmt.Println("no nameNote function call found in response")
			return "", fmt.Errorf("No nameNote function call found in response. This usually means the model failed to generate a valid response.")
		}

		var nameRes prompts.NoteNameRes
		bytes := []byte(res)
		err = json.Unmarshal(bytes, &nameRes)
		if err != nil {
			fmt.Printf("Error unmarshalling note name response: %v\n", err)
			return "", err
		}
		name = nameRes.Name
	}

	var inputTokens int
	var outputTokens int
	if resp.Usage.CompletionTokens > 0 {
		inputTokens = resp.Usage.PromptTokens
		outputTokens = resp.Usage.CompletionTokens
	} else {
		inputTokens = numTokens
		outputTokens = shared.GetNumTokensEstimate(name)
	}

	var cachedTokens int
	if resp.Usage.PromptTokensDetails != nil {
		cachedTokens = resp.Usage.PromptTokensDetails.CachedTokens
	}

	go func() {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:      inputTokens,
				OutputTokens:     outputTokens,
				CachedTokens:     cachedTokens,
				ModelName:        config.BaseModelConfig.ModelName,
				ModelProvider:    config.BaseModelConfig.Provider,
				ModelPackName:    settings.ModelPack.Name,
				ModelRole:        shared.ModelRolePlanSummary,
				Purpose:          "Generated name for note added to context",
				GenerationId:     resp.ID,
				PlanId:           plan.Id,
				RequestStartedAt: reqStarted,
				Streaming:        false,
				Req:              &req,
				Res:              &resp,
				ModelConfig:      &config,
			},
		})

		if apiErr != nil {
			log.Printf("GenNoteName - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

	return name, nil
}
