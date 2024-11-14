package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) execStatusShouldContinue(message string, latestSummaryCh chan *db.ConvoSummary, ctx context.Context) (bool, string, error) {
	auth := state.auth
	plan := state.plan
	settings := state.settings
	clients := state.clients
	config := settings.ModelPack.ExecStatus

	envVar := config.BaseModelConfig.ApiKeyEnvVar
	client := clients[envVar]

	log.Println("Checking if plan should continue based on exec status")

	// First try to determine if the plan should continue based on the last paragraph without calling the model
	paragraphs := strings.Split(message, "\n\n")
	lastParagraph := paragraphs[len(paragraphs)-1]
	lastParagraphLower := strings.ToLower(lastParagraph)

	// log.Printf("Last paragraph: %s\n", lastParagraphLower)

	if lastParagraphLower != "" {
		nextIdx := strings.Index(lastParagraph, "Next, ")
		if nextIdx >= 0 {
			log.Println("Plan can be continued based on last paragraph")
			return true, lastParagraph, nil
		}
	}

	var prevAssistantMsg string
	if len(state.convo) > 1 {
		// iterate backwards from len(state.convo) - 2 to 0
		for i := len(state.convo) - 2; i >= 0; i-- {
			if state.convo[i].Role == "assistant" {
				prevAssistantMsg = state.convo[i].Message
				break
			}
		}
	}

	var latestSummary *db.ConvoSummary

	if latestSummaryCh != nil {
		log.Println("Waiting for latest summary")
		select {
		case <-ctx.Done():
			log.Println("Context cancelled while waiting for latest summary")
			return false, "", fmt.Errorf("context cancelled while waiting for latest summary")
		case latestSummary = <-latestSummaryCh:
			log.Println("Got latest summary")
		}
	}

	var summary string
	if latestSummary != nil {
		summary = latestSummary.Summary
	}

	content := prompts.GetExecStatusShouldContinue(summary, prevAssistantMsg, state.userPrompt, message)

	contentTokens, err := shared.GetNumTokens(content)

	if err != nil {
		log.Printf("Error getting num tokens for content: %v\n", err)
		return false, "", fmt.Errorf("error getting num tokens for content: %v", err)
	}

	numTokens := prompts.ExtraTokensPerRequest + prompts.ExtraTokensPerMessage + contentTokens

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
		return false, "", err
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: content,
		},
	}

	log.Println("Calling model to check if plan should continue")
	log.Println("Has latest summary:", summary != "")
	log.Println("Has prev assistant msg:", prevAssistantMsg != "")

	// log.Println("messages:")
	// log.Println(spew.Sdump(messages))

	var responseFormat *openai.ChatCompletionResponseFormat
	if config.BaseModelConfig.HasJsonResponseMode {
		responseFormat = &openai.ChatCompletionResponseFormat{Type: "json_object"}
	}

	resp, err := model.CreateChatCompletionWithRetries(
		client,
		ctx,
		openai.ChatCompletionRequest{
			Model: config.BaseModelConfig.ModelName,
			Tools: []openai.Tool{
				{
					Type:     "function",
					Function: &prompts.ShouldAutoContinueFn,
				},
			},
			ToolChoice: openai.ToolChoice{
				Type: "function",
				Function: openai.ToolFunction{
					Name: prompts.ShouldAutoContinueFn.Name,
				},
			},
			Messages:       messages,
			ResponseFormat: responseFormat,
			Temperature:    config.Temperature,
			TopP:           config.TopP,
		},
	)

	if err != nil {
		log.Printf("Error during plan exec status check model call: %v\n", err)
		// return false, fmt.Errorf("error during plan exec status check model call: %v", err)

		// Instead of erroring out, just don't continue the plan
		return false, "", nil
	}

	var strRes string
	var res struct {
		MessageFinishedSubtasks []string `json:"messageSubtasksFinished"`
		Comments                []struct {
			Txt               string `json:"txt"`
			IsTodoPlaceholder bool   `json:"isTodoPlaceholder"`
		} `json:"comments"`
		Reasoning      string `json:"reasoning"`
		ShouldContinue bool   `json:"shouldContinue"`
	}

	for _, choice := range resp.Choices {
		if len(choice.Message.ToolCalls) == 1 &&
			choice.Message.ToolCalls[0].Function.Name == prompts.ShouldAutoContinueFn.Name {
			fnCall := choice.Message.ToolCalls[0].Function
			strRes = fnCall.Arguments
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
		outputTokens, err = shared.GetNumTokens(strRes)

		if err != nil {
			return false, "", fmt.Errorf("error getting num tokens for res: %v", err)
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
			Purpose:       "Evaluate if plan should auto-continue",
		},
	})

	if apiErr != nil {
		return false, "", fmt.Errorf("error executing hook: %v", apiErr)
	}

	if strRes == "" {
		log.Println("No shouldAutoContinue function call found in response")
		log.Println(spew.Sdump(resp))

		// return false, fmt.Errorf("no shouldAutoContinue function call found in response")

		// Instead of erroring out, just don't continue the plan
		return false, "", nil
	}

	err = json.Unmarshal([]byte(strRes), &res)
	if err != nil {
		log.Printf("Error unmarshalling plan exec status response: %v\n", err)

		// return false, fmt.Errorf("error unmarshalling plan exec status response: %v", err)

		// Instead of erroring out, just don't continue the plan
		return false, "", nil
	}

	log.Println("Plan exec status response:")
	log.Println(spew.Sdump(res))

	return res.ShouldContinue, res.Reasoning, nil
}
