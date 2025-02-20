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
	"time"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

// controls the number steps to spent trying to finish a single subtask
// if a subtask is not finished in this number of steps, we'll give up and mark it done
// necessary to prevent infinite loops
const MaxPreviousMessages = 4

type execStatusShouldContinueResult struct {
	subtaskFinished bool
}

func (state *activeTellStreamState) execStatusShouldContinue(currentMessage string, ctx context.Context) (execStatusShouldContinueResult, *shared.ApiError) {
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

		if strings.Contains(currentMessage, completionMarker) {
			log.Printf("[ExecStatus] ✓ Subtask completion marker found")

			var potentialProblem bool

			if len(state.chunkProcessor.replyOperations) == 0 {
				// subtask was marked as completed, but there are no operations to execute
				// we'll let this fall through to the LLM call to verify, since something might have gone wrong
				log.Printf("[ExecStatus] ✗ Subtask completion marker found, but there are no operations to execute")
				potentialProblem = true
			} else {
				wroteToPaths := map[string]bool{}
				for _, op := range state.chunkProcessor.replyOperations {
					if op.Type == shared.OperationTypeFile {
						wroteToPaths[op.Path] = true
					}
				}

				for _, path := range state.currentSubtask.UsesFiles {
					if !wroteToPaths[path] {
						log.Printf("[ExecStatus] ✗ Subtask completion marker found, but the operations did not write to the file %q from the 'Uses' list", path)
						potentialProblem = true
						break
					}
				}
			}

			if !potentialProblem {
				log.Printf("[ExecStatus] ✓ Subtask completion marker found and no potential problem - will mark as completed")
				return execStatusShouldContinueResult{
					subtaskFinished: true,
				}, nil
			} else {
				log.Printf("[ExecStatus] ✗ Subtask completion marker found, but the operations are questionable -- will verify with LLM call")
			}
		} else {
			log.Printf("[ExecStatus] ✗ No subtask completion marker found in message")
		}

		// Log all subtasks current state for context
		log.Println("[ExecStatus] Current subtasks state:")
		for i, task := range state.subtasks {
			log.Printf("[ExecStatus] Task %d: %q (finished=%v)", i+1, task.Title, task.IsFinished)
		}
	}

	log.Println("Checking if plan should continue based on exec status")

	fullSubtask := state.currentSubtask.Title
	fullSubtask += "\n\n" + state.currentSubtask.Description

	previousMessages := []string{}
	for _, msg := range state.convo {
		if msg.Subtask != nil && msg.Subtask.Title == state.currentSubtask.Title {
			previousMessages = append(previousMessages, msg.Message)
		}
	}

	if len(previousMessages) >= MaxPreviousMessages {
		log.Printf("[ExecStatus] ✗ Max previous messages reached - will mark as completed and move on to next subtask")
		return execStatusShouldContinueResult{
			subtaskFinished: true,
		}, nil
	}

	content := prompts.GetExecStatusFinishedSubtask(state.userPrompt, fullSubtask, currentMessage, previousMessages)

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

	numTokens := model.GetMessagesTokenEstimate(messages...) + model.TokensPerRequest

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
		return execStatusShouldContinueResult{}, apiErr
	}

	log.Println("Calling model to check if plan should continue")

	reqStarted := time.Now()
	req := types.ExtendedChatCompletionRequest{
		Model: config.BaseModelConfig.ModelName,
		Tools: []openai.Tool{
			{
				Type:     "function",
				Function: &prompts.DidFinishSubtaskFn,
			},
		},
		ToolChoice: openai.ToolChoice{
			Type: "function",
			Function: openai.ToolFunction{
				Name: prompts.DidFinishSubtaskFn.Name,
			},
		},
		Messages:    messages,
		Temperature: config.Temperature,
		TopP:        config.TopP,
	}

	resp, err := model.CreateChatCompletionWithRetries(
		clients,
		&config,
		ctx,
		req,
	)

	if err != nil {
		log.Printf("[ExecStatus] Error in model call: %v", err)
		return execStatusShouldContinueResult{}, nil
	}

	var strRes string
	for _, choice := range resp.Choices {
		if len(choice.Message.ToolCalls) == 1 &&
			choice.Message.ToolCalls[0].Function.Name == prompts.DidFinishSubtaskFn.Name {
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
				ModelRole:      shared.ModelRoleExecStatus,
				Purpose:        "Evaluate if task was finished",
				GenerationId:   resp.ID,
				PlanId:         plan.Id,
				ModelStreamId:  state.activePlan.ModelStreamId,
				ConvoMessageId: state.replyId,

				RequestStartedAt: reqStarted,
				Streaming:        false,
				Req:              &req,
				Res:              &resp,
				ModelConfig:      &config,
			},
		})

		if apiErr != nil {
			log.Printf("execStatusShouldContinue - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

	if strRes == "" {
		log.Printf("[ExecStatus] No function response found in model output")
		return execStatusShouldContinueResult{}, nil
	}

	var res types.ExecStatusResponse
	if err := json.Unmarshal([]byte(strRes), &res); err != nil {
		log.Printf("[ExecStatus] Failed to parse response: %v", err)
		return execStatusShouldContinueResult{}, nil
	}

	log.Printf("[ExecStatus] Decision: subtaskFinished=%v, reasoning=%v",
		res.SubtaskFinished, res.Reasoning)

	return execStatusShouldContinueResult{
		subtaskFinished: res.SubtaskFinished,
	}, nil
}
