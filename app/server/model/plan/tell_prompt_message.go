package plan

import (
	"log"
	"net/http"
	"plandex-server/model/prompts"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) setPromptMessage(isPlanningStage, isContextStage bool, applyScriptSummary string) bool {
	planId := state.plan.Id
	branch := state.branch
	req := state.req
	iteration := state.iteration

	active := GetActivePlan(planId, branch)

	if active == nil {
		log.Printf("execTellPlan: Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return false
	}

	var promptMessage *openai.ChatCompletionMessage
	if req.IsUserContinue {
		if len(state.messages) == 0 {
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeContinueNoMessages,
				Status: http.StatusBadRequest,
				Msg:    "No messages yet. Can't continue plan.",
			}
			return false
		}

		// if the user is continuing the plan, we need to check whether the previous message was a user message or assistant message
		lastMessage := state.messages[len(state.messages)-1]

		log.Println("User is continuing plan. Last message role:", lastMessage.Role)
		// log.Println("User is continuing plan. Last message:\n\n", lastMessage.Content)

		if lastMessage.Role == openai.ChatMessageRoleUser {
			// if last message was a user message, we want to remove it from the messages array and then use that last message as the prompt so we can continue from where the user left off

			log.Println("User is continuing plan. Last message was user message. Using last user message as prompt")

			state.messages = state.messages[:len(state.messages)-1]
			promptMessage = &openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: prompts.GetWrappedPrompt(lastMessage.Content, req.OsDetails, applyScriptSummary, isPlanningStage),
			}

			state.userPrompt = lastMessage.Content
		} else {

			// if the last message was an assistant message, we'll use the user continue prompt
			log.Println("User is continuing plan. Last message was assistant message. Using user continue prompt")

			// otherwise we'll use the continue prompt
			promptMessage = &openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: prompts.GetWrappedPrompt(prompts.UserContinuePrompt, req.OsDetails, applyScriptSummary, isPlanningStage),
			}
		}

		state.messages = append(state.messages, *promptMessage)
	} else {
		var prompt string
		if iteration == 0 {
			if req.IsChatOnly {
				prompt = req.Prompt + prompts.ChatOnlyPrompt
				state.totalRequestTokens += prompts.ChatOnlyPromptTokens
			} else if req.IsUserDebug {
				prompt = req.Prompt + prompts.DebugPrompt
				state.totalRequestTokens += prompts.DebugPromptTokens
			} else if req.IsApplyDebug {
				prompt = req.Prompt + prompts.ApplyDebugPrompt
				state.totalRequestTokens += prompts.ApplyDebugPromptTokens
			} else {
				prompt = req.Prompt
			}
		} else {
			prompt = prompts.AutoContinuePrompt
		}

		state.userPrompt = prompt

		var finalPrompt string
		if isContextStage {
			finalPrompt = prompt
		} else {
			finalPrompt = prompts.GetWrappedPrompt(prompt, req.OsDetails, applyScriptSummary, isPlanningStage)
		}

		// log.Println("Final prompt:", finalPrompt)

		promptMessage = &openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: finalPrompt,
		}
	}

	// log.Println("Prompt message:", promptMessage.Content)

	state.promptMessage = promptMessage
	state.messages = append(state.messages, *promptMessage)

	return true
}
