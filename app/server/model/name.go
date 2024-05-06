package model

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"plandex-server/model/prompts"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func GenPlanName(client *openai.Client, config shared.ModelRoleConfig, planContent string) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.GetPlanNamePrompt(planContent),
		},
	}

	var responseFormat *openai.ChatCompletionResponseFormat
	if config.BaseModelConfig.HasJsonResponseMode {
		responseFormat = &openai.ChatCompletionResponseFormat{Type: "json_object"}
	}

	resp, err := CreateChatCompletionWithRetries(
		client,
		context.Background(),
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

	if res == "" {
		fmt.Println("no namePlan function call found in response")
		return "", err
	}

	bytes := []byte(res)

	err = json.Unmarshal(bytes, &nameRes)
	if err != nil {
		fmt.Printf("Error unmarshalling plan description response: %v\n", err)
		return "", err
	}

	return nameRes.PlanName, nil

}

func GenPipedDataName(client *openai.Client, config shared.ModelRoleConfig, pipedContent string) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.GetPipedDataNamePrompt(pipedContent),
		},
	}

	var responseFormat *openai.ChatCompletionResponseFormat
	if config.BaseModelConfig.HasJsonResponseMode {
		responseFormat = &openai.ChatCompletionResponseFormat{Type: "json_object"}
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

	if res == "" {
		fmt.Println("no namePipedData function call found in response")
		return "", err
	}

	bytes := []byte(res)

	err = json.Unmarshal(bytes, &nameRes)
	if err != nil {
		fmt.Printf("Error unmarshalling piped data name response: %v\n", err)
		return "", err
	}

	return nameRes.Name, nil

}
