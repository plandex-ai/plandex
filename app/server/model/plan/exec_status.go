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

	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) execStatusShouldContinue(message string, ctx context.Context) (bool, bool, *shared.ApiError) {
	auth := state.auth
	plan := state.plan
	settings := state.settings
	clients := state.clients
	config := settings.ModelPack.ExecStatus

	// Check subtask completion
	if state.currentSubtask != nil {
		completionMarker := fmt.Sprintf("**%s** has been completed", state.currentSubtask.Title)
		log.Printf("[ExecStatus] Checking for subtask completion marker: %q", completionMarker)
		log.Printf("[ExecStatus] Current subtask: %q (finished=%v)", state.currentSubtask.Title, state.currentSubtask.IsFinished)

		if strings.Contains(message, completionMarker) {
			log.Printf("[ExecStatus] ✓ Subtask completion marker found - will mark as completed")
			return true, true, nil
		}
		log.Printf("[ExecStatus] ✗ No subtask completion marker found in message")

		// Log all subtasks current state for context
		log.Println("[ExecStatus] Current subtasks state:")
		for i, task := range state.subtasks {
			log.Printf("[ExecStatus] Task %d: %q (finished=%v)", i+1, task.Title, task.IsFinished)
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
		log.Printf("[ExecStatus] Error in model call: %v", err)
		return false, false, nil
	}

	var strRes string
	for _, choice := range resp.Choices {
		if len(choice.Message.ToolCalls) == 1 &&
			choice.Message.ToolCalls[0].Function.Name == prompts.ShouldAutoContinueFn.Name {
			strRes = choice.Message.ToolCalls[0].Function.Arguments
			log.Printf("[ExecStatus] Got function response: %s", strRes)
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
		log.Printf("[ExecStatus] No function response found in model output")
		return false, false, nil
	}

	var res types.ExecStatusResponse
	if err := json.Unmarshal([]byte(strRes), &res); err != nil {
		log.Printf("[ExecStatus] Failed to parse response: %v", err)
		return false, false, nil
	}

	log.Printf("[ExecStatus] Decision: subtaskFinished=%v, shouldContinue=%v",
		res.SubtaskFinished, res.ShouldContinue)

	return res.SubtaskFinished, res.ShouldContinue, nil
}
