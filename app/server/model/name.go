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
	content := prompts.GetPlanNamePrompt(planContent)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
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
		return "", fmt.Errorf("error executing hook: %v", apiErr)
	}

	resp, err := CreateChatCompletionWithRetries(
		clients,
		&config,
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
			Temperature: config.Temperature,
			TopP:        config.TopP,
			Messages:    messages,
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
		outputTokens = shared.GetNumTokensEstimate(res)
	}

	go func() {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
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
				GenerationId:  resp.ID,
				PlanId:        plan.Id,
			},
		})

		if apiErr != nil {
			log.Printf("GenPlanName - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

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
	ctx context.Context,
	auth *types.ServerAuth,
	plan *db.Plan,
	settings *shared.PlanSettings,
	clients map[string]ClientInfo,
	pipedContent string,
) (string, error) {
	config := settings.ModelPack.Namer

	content := prompts.GetPipedDataNamePrompt(pipedContent)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
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
		return "", fmt.Errorf("error executing hook: %v", apiErr)
	}

	log.Println("calling piped data name model")
	// log.Printf("model: %s\n", config.BaseModelConfig.ModelName)
	// log.Printf("temperature: %f\n", config.Temperature)
	// log.Printf("topP: %f\n", config.TopP)

	// log.Printf("messages: %v\n", messages)
	// log.Println(spew.Sdump(messages))

	resp, err := CreateChatCompletionWithRetries(
		clients,
		&config,
		ctx,
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
			Temperature: config.Temperature,
			TopP:        config.TopP,
			Messages:    messages,
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
		outputTokens = shared.GetNumTokensEstimate(res)
	}

	go func() {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
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
				GenerationId:  resp.ID,
				PlanId:        plan.Id,
			},
		})

		if apiErr != nil {
			log.Printf("GenPipedDataName - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

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
	ctx context.Context,
	auth *types.ServerAuth,
	plan *db.Plan,
	settings *shared.PlanSettings,
	clients map[string]ClientInfo,
	note string,
) (string, error) {
	config := settings.ModelPack.Namer

	content := prompts.GetNoteNamePrompt(note)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
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
		return "", fmt.Errorf("error executing hook: %v", apiErr)
	}

	log.Println("calling piped data name model")
	// log.Printf("model: %s\n", config.BaseModelConfig.ModelName)
	// log.Printf("temperature: %f\n", config.Temperature)
	// log.Printf("topP: %f\n", config.TopP)

	// log.Printf("messages: %v\n", messages)
	// log.Println(spew.Sdump(messages))

	resp, err := CreateChatCompletionWithRetries(
		clients,
		&config,
		ctx,
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
			Temperature: config.Temperature,
			TopP:        config.TopP,
			Messages:    messages,
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
		outputTokens = shared.GetNumTokensEstimate(res)
	}

	go func() {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
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
				GenerationId:  resp.ID,
				PlanId:        plan.Id,
			},
		})

		if apiErr != nil {
			log.Printf("GenNoteName - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

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
