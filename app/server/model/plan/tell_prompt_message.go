package plan

import (
	"log"
	"net/http"
	"plandex-server/model/prompts"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) resolvePromptMessage(isPlanningStage, isContextStage, didLoadChatOnlyContext bool, applyScriptSummary string) (*openai.ChatCompletionMessage, bool) {
	req := state.req
	iteration := state.iteration
	active := state.activePlan
	isFollowUp := state.isFollowUp

	if isFollowUp && (req.IsApplyDebug) {
		log.Println("resolvePromptMessage: IsApplyDebug or IsUserDebug - setting isFollowUp to false")
		isFollowUp = false
	}

	var promptMessage *openai.ChatCompletionMessage

	var lastMessage *openai.ChatCompletionMessage

	if req.IsUserContinue {
		if len(state.messages) == 0 {
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeContinueNoMessages,
				Status: http.StatusBadRequest,
				Msg:    "No messages yet. Can't continue plan.",
			}
			return nil, false
		}

		// if the user is continuing the plan, we need to check whether the previous message was a user message or assistant message
		lastMessage = &state.messages[len(state.messages)-1]

		log.Println("User is continuing plan. Last message role:", lastMessage.Role)
	}

	if req.IsChatOnly {
		var prompt string
		if req.IsUserContinue {
			if lastMessage.Role == openai.ChatMessageRoleUser {
				log.Println("User is continuing plan in chat only mode. Last message was user message. Using last user message as prompt")
				prompt = lastMessage.Content
				state.userPrompt = prompt
				state.messages = state.messages[:len(state.messages)-1]
			} else {
				log.Println("User is continuing plan in chat only mode. Last message was assistant message. Using user continue prompt")
				prompt = prompts.UserContinuePrompt
			}
		} else {
			prompt = req.Prompt
		}

		wrapped := prompts.GetWrappedChatOnlyPrompt(prompts.ChatUserPromptParams{
			CreatePromptParams: prompts.CreatePromptParams{
				AutoContext:               req.AutoContext,
				ExecMode:                  req.ExecEnabled,
				LastResponseLoadedContext: didLoadChatOnlyContext,
			},
			Prompt:    prompt,
			OsDetails: req.OsDetails,
		})

		promptMessage = &openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: wrapped,
		}

		state.messages = append(state.messages, *promptMessage)
	} else if req.IsUserContinue {
		// log.Println("User is continuing plan. Last message:\n\n", lastMessage.Content)
		if lastMessage.Role == openai.ChatMessageRoleUser {
			// if last message was a user message, we want to remove it from the messages array and then use that last message as the prompt so we can continue from where the user left off

			log.Println("User is continuing plan in tell mode. Last message was user message. Using last user message as prompt")

			state.messages = state.messages[:len(state.messages)-1]

			params := prompts.UserPromptParams{
				CreatePromptParams: prompts.CreatePromptParams{
					ExecMode:    req.ExecEnabled,
					AutoContext: req.AutoContext,
				},
				Prompt:          lastMessage.Content,
				OsDetails:       req.OsDetails,
				IsPlanningStage: isPlanningStage,
				IsFollowUp:      isFollowUp,
			}

			promptMessage = &openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: prompts.GetWrappedPrompt(params),
			}

			state.userPrompt = lastMessage.Content
		} else {

			// if the last message was an assistant message, we'll use the user continue prompt
			log.Println("User is continuing plan in tell mode. Last message was assistant message. Using user continue prompt")

			params := prompts.UserPromptParams{
				CreatePromptParams: prompts.CreatePromptParams{
					ExecMode:    req.ExecEnabled,
					AutoContext: req.AutoContext,
				},
				Prompt:             prompts.UserContinuePrompt,
				OsDetails:          req.OsDetails,
				IsPlanningStage:    isPlanningStage,
				IsFollowUp:         isFollowUp,
				ApplyScriptSummary: applyScriptSummary,
			}

			promptMessage = &openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: prompts.GetWrappedPrompt(params),
			}
		}

		state.messages = append(state.messages, *promptMessage)
	} else {
		var prompt string
		if iteration == 0 {
			if req.IsUserDebug {
				prompt = req.Prompt + prompts.DebugPrompt
			} else if req.IsApplyDebug {
				prompt = req.Prompt + prompts.ApplyDebugPrompt
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
			params := prompts.UserPromptParams{
				CreatePromptParams: prompts.CreatePromptParams{
					ExecMode:    req.ExecEnabled,
					AutoContext: req.AutoContext,
				},
				Prompt:             prompt,
				OsDetails:          req.OsDetails,
				IsPlanningStage:    isPlanningStage,
				IsFollowUp:         isFollowUp,
				ApplyScriptSummary: applyScriptSummary,
			}

			finalPrompt = prompts.GetWrappedPrompt(params)
		}

		// log.Println("Final prompt:", finalPrompt)

		promptMessage = &openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: finalPrompt,
		}
	}

	// log.Println("Prompt message:", promptMessage.Content)

	return promptMessage, true
}
