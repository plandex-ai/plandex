package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/types"
	"strings"

	shared "plandex-shared"

	"github.com/davecgh/go-spew/spew"
	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) execStatusShouldContinue(message string, ctx context.Context) (bool, bool, *shared.ApiError) {
	auth := state.auth
	plan := state.plan
	settings := state.settings
	clients := state.clients
	config := settings.ModelPack.ExecStatus

	log.Println("Checking if plan should continue based on response text")

	if state.currentSubtask != nil {
		s := fmt.Sprintf("**%s** has been completed", state.currentSubtask.Title)

		log.Println("Checking if message contains subtask completion")
		// log.Println(s)
		// log.Println("---")
		// log.Println(message)

		if strings.Contains(message, s) {
			log.Println("Subtask marked completed in message. Will continue")
			return true, true, nil
		}
	}

	log.Println("Checking if plan should continue based on exec status")

	content := prompts.GetExecStatusShouldContinue(state.userPrompt, message)

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
		return false, false, apiErr
	}

	log.Println("Calling model to check if plan should continue")

	// log.Println("messages:")
	// log.Println(spew.Sdump(messages))

	resp, err := model.CreateChatCompletionWithRetries(
		clients,
		&config,
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
			Messages:    messages,
			Temperature: config.Temperature,
			TopP:        config.TopP,
		},
	)

	if err != nil {
		log.Printf("Error during plan exec status check model call: %v\n", err)
		// return false, fmt.Errorf("error during plan exec status check model call: %v", err)

		// Instead of erroring out, just don't continue the plan
		return false, false, nil
	}

	var strRes string
	var res types.ExecStatusResponse

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
		outputTokens = shared.GetNumTokensEstimate(strRes)
	}

	go func() {
		_, apiErr = hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:    inputTokens,
				OutputTokens:   outputTokens,
				ModelName:      config.BaseModelConfig.ModelName,
				ModelProvider:  config.BaseModelConfig.Provider,
				ModelPackName:  settings.ModelPack.Name,
				ModelRole:      shared.ModelRolePlanSummary,
				Purpose:        "Evaluate if plan should auto-continue",
				GenerationId:   resp.ID,
				PlanId:         plan.Id,
				ModelStreamId:  state.activePlan.ModelStreamId,
				ConvoMessageId: state.replyId,
			},
		})

		if apiErr != nil {
			log.Printf("execStatusShouldContinue - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

	if strRes == "" {
		log.Println("No shouldAutoContinue function call found in response")
		log.Println(spew.Sdump(resp))

		// return false, fmt.Errorf("no shouldAutoContinue function call found in response")

		// Instead of erroring out, just don't continue the plan
		return false, false, nil
	}

	err = json.Unmarshal([]byte(strRes), &res)
	if err != nil {
		log.Printf("Error unmarshalling plan exec status response: %v\n", err)

		// return false, fmt.Errorf("error unmarshalling plan exec status response: %v", err)

		// Instead of erroring out, just don't continue the plan
		return false, false, nil
	}

	log.Println("Plan exec status response:")
	log.Println(spew.Sdump(res))

	return res.SubtaskFinished, res.ShouldContinue, nil
}
