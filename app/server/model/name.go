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

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func GenPlanName(
	auth *types.ServerAuth,
	plan *db.Plan,
	settings *shared.PlanSettings,
	client *openai.Client,
	planContent string,
	ctx context.Context,
) (string, error) {
	config := settings.ModelPack.Namer
	content := prompts.GetPlanNamePrompt(planContent)

	contentTokens, err := shared.GetNumTokens(content)

	if err != nil {
		return "", fmt.Errorf("error getting num tokens for content: %v", err)
	}

	numTokens := prompts.ExtraTokensPerRequest + prompts.ExtraTokensPerMessage + contentTokens

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: content,
		},
	}

	var responseFormat *openai.ChatCompletionResponseFormat
	if config.BaseModelConfig.HasJsonResponseMode {
		responseFormat = &openai.ChatCompletionResponseFormat{Type: "json_object"}
	}

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
		return "", err
	}

	resp, err := CreateChatCompletionWithRetries(
		client,
		ctx,
		openai.ChatCompletionRequest{
			Model: config.BaseModelConfig.ModelName,
			Tools: []openai.Tool{
				{
					Type:     "function",
					Function: &prompts.PlanNameFn,
				},
			},
			ToolChoice: openai.ToolChoice{
				Type: "function",
				Function: openai.ToolFunction{
					Name: prompts.PlanNameFn.Name,
				},
			},
			Temperature:    config.Temperature,
			TopP:           config.TopP,
			Messages:       messages,
			ResponseFormat: responseFormat,
		},
	)

	var res string
	var nameRes prompts.PlanNameRes

	if err != nil {
		fmt.Printf("Error during plan name model call: %v\n", err)
		return "", err
	}

	for _, choice := range resp.Choices {
		if len(choice.Message.ToolCalls) == 1 &&
			choice.Message.ToolCalls[0].Function.Name == prompts.PlanNameFn.Name {
			fnCall := choice.Message.ToolCalls[0].Function
			res = fnCall.Arguments
			break
		}
	}

	var inputTokens int
	var outputTokens int
	if resp.Usage.CompletionTokens > 0 {
		inputTokens = resp.Usage.PromptTokens
		outputTokens = resp.Usage.CompletionTokens
	} else {
		inputTokens = numTokens
		outputTokens, err = shared.GetNumTokens(res)

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
			ModelRole:     shared.ModelRolePlanSummary,
			Purpose:       "Generated plan name",
		},
	})

	if res == "" {
		fmt.Println("no namePlan function call found in response")
		return "", fmt.Errorf("No namePlan function call found in response. This usually means the model failed to generate a valid response.")
	}

	bytes := []byte(res)

	err = json.Unmarshal(bytes, &nameRes)
	if err != nil {
		fmt.Printf("Error unmarshalling plan description response: %v\n", err)
		return "", err
	}

	return nameRes.PlanName, nil

}

func GenPipedDataName(
	auth *types.ServerAuth,
	plan *db.Plan,
	settings *shared.PlanSettings,
	client *openai.Client,
	pipedContent string,
) (string, error) {
	config := settings.ModelPack.Namer

	content := prompts.GetPipedDataNamePrompt(pipedContent)

	contentTokens, err := shared.GetNumTokens(content)

	if err != nil {
		return "", fmt.Errorf("error getting num tokens for content: %v", err)
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: content,
		},
	}

	numTokens := prompts.ExtraTokensPerRequest + prompts.ExtraTokensPerMessage + contentTokens

	var responseFormat *openai.ChatCompletionResponseFormat
	if config.BaseModelConfig.HasJsonResponseMode {
		responseFormat = &openai.ChatCompletionResponseFormat{Type: "json_object"}
	}

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
		return "", err
	}

	log.Println("calling piped data name model")
	// log.Printf("model: %s\n", config.BaseModelConfig.ModelName)
	// log.Printf("temperature: %f\n", config.Temperature)
	// log.Printf("topP: %f\n", config.TopP)

	// log.Printf("messages: %v\n", messages)
	// log.Println(spew.Sdump(messages))

	resp, err := CreateChatCompletionWithRetries(
		client,
		context.Background(),
		openai.ChatCompletionRequest{
			Model: config.BaseModelConfig.ModelName,
			Tools: []openai.Tool{
				{
					Type:     "function",
					Function: &prompts.PipedDataNameFn,
				},
			},
			ToolChoice: openai.ToolChoice{
				Type: "function",
				Function: openai.ToolFunction{
					Name: prompts.PipedDataNameFn.Name,
				},
			},
			Temperature:    config.Temperature,
			TopP:           config.TopP,
			Messages:       messages,
			ResponseFormat: responseFormat,
		},
	)

	var res string
	var nameRes prompts.PipedDataNameRes

	if err != nil {
		fmt.Printf("Error during piped data name model call: %v\n", err)
		return "", err
	}

	for _, choice := range resp.Choices {
		if len(choice.Message.ToolCalls) == 1 &&
			choice.Message.ToolCalls[0].Function.Name == prompts.PipedDataNameFn.Name {
			fnCall := choice.Message.ToolCalls[0].Function
			res = fnCall.Arguments
			break
		}
	}

	var inputTokens int
	var outputTokens int
	if resp.Usage.CompletionTokens > 0 {
		inputTokens = resp.Usage.PromptTokens
		outputTokens = resp.Usage.CompletionTokens
	} else {
		inputTokens = numTokens
		outputTokens, err = shared.GetNumTokens(res)

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
			ModelRole:     shared.ModelRolePlanSummary,
			Purpose:       "Generated name for data piped into context",
		},
	})

	if apiErr != nil {
		return "", fmt.Errorf("error executing hook: %v", apiErr)
	}

	if res == "" {
		fmt.Println("no namePipedData function call found in response")
		return "", fmt.Errorf("No namePipedData function call found in response. This usually means the model failed to generate a valid response.")
	}

	bytes := []byte(res)

	err = json.Unmarshal(bytes, &nameRes)
	if err != nil {
		fmt.Printf("Error unmarshalling piped data name response: %v\n", err)
		return "", err
	}

	return nameRes.Name, nil

}

func GenNoteName(
	auth *types.ServerAuth,
	plan *db.Plan,
	settings *shared.PlanSettings,
	client *openai.Client,
	note string,
) (string, error) {
	config := settings.ModelPack.Namer

	content := prompts.GetNoteNamePrompt(note)

	contentTokens, err := shared.GetNumTokens(content)

	if err != nil {
		return "", fmt.Errorf("error getting num tokens for content: %v", err)
	}

	numTokens := prompts.ExtraTokensPerRequest + prompts.ExtraTokensPerMessage + contentTokens

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: content,
		},
	}

	var responseFormat *openai.ChatCompletionResponseFormat
	if config.BaseModelConfig.HasJsonResponseMode {
		responseFormat = &openai.ChatCompletionResponseFormat{Type: "json_object"}
	}

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
		return "", err
	}

	log.Println("calling piped data name model")
	// log.Printf("model: %s\n", config.BaseModelConfig.ModelName)
	// log.Printf("temperature: %f\n", config.Temperature)
	// log.Printf("topP: %f\n", config.TopP)

	// log.Printf("messages: %v\n", messages)
	// log.Println(spew.Sdump(messages))

	resp, err := CreateChatCompletionWithRetries(
		client,
		context.Background(),
		openai.ChatCompletionRequest{
			Model: config.BaseModelConfig.ModelName,
			Tools: []openai.Tool{
				{
					Type:     "function",
					Function: &prompts.NoteNameFn,
				},
			},
			ToolChoice: openai.ToolChoice{
				Type: "function",
				Function: openai.ToolFunction{
					Name: prompts.NoteNameFn.Name,
				},
			},
			Temperature:    config.Temperature,
			TopP:           config.TopP,
			Messages:       messages,
			ResponseFormat: responseFormat,
		},
	)

	var res string
	var nameRes prompts.NoteNameRes

	if err != nil {
		fmt.Printf("Error during piped data name model call: %v\n", err)
		return "", err
	}

	for _, choice := range resp.Choices {
		if len(choice.Message.ToolCalls) == 1 &&
			choice.Message.ToolCalls[0].Function.Name == prompts.NoteNameFn.Name {
			fnCall := choice.Message.ToolCalls[0].Function
			res = fnCall.Arguments
			break
		}
	}

	var inputTokens int
	var outputTokens int
	if resp.Usage.CompletionTokens > 0 {
		inputTokens = resp.Usage.PromptTokens
		outputTokens = resp.Usage.CompletionTokens
	} else {
		inputTokens = numTokens
		outputTokens, err = shared.GetNumTokens(res)

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
			ModelRole:     shared.ModelRolePlanSummary,
			Purpose:       "Generated name for note added to context",
		},
	})

	if apiErr != nil {
		return "", fmt.Errorf("error executing hook: %v", apiErr)
	}

	if res == "" {
		fmt.Println("no nameNote function call found in response")
		return "", fmt.Errorf("No nameNote function call found in response. This usually means the model failed to generate a valid response.")
	}

	bytes := []byte(res)

	err = json.Unmarshal(bytes, &nameRes)
	if err != nil {
		fmt.Printf("Error unmarshalling piped data name response: %v\n", err)
		return "", err
	}

	return nameRes.Name, nil

}
