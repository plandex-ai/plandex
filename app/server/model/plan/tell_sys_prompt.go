package plan

import (
	"fmt"
	"log"
	"plandex-server/model/prompts"
	"plandex-server/types"
	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

type getTellSysPromptParams struct {
	autoContextEnabled  bool
	smartContextEnabled bool
	basicContextMsg     *types.ExtendedChatMessagePart
	autoContextMsg      *types.ExtendedChatMessagePart
	smartContextMsg     *types.ExtendedChatMessagePart
}

func (state *activeTellStreamState) getTellSysPrompt(params getTellSysPromptParams) ([]types.ExtendedChatMessagePart, error) {
	autoContextEnabled := params.autoContextEnabled
	smartContextEnabled := params.smartContextEnabled
	basicContextMsg := params.basicContextMsg
	autoContextMsg := params.autoContextMsg
	smartContextMsg := params.smartContextMsg
	req := state.req
	active := state.activePlan
	currentStage := state.currentStage

	// var sysCreate string

	sysParts := []types.ExtendedChatMessagePart{}

	createPromptParams := prompts.CreatePromptParams{
		ExecMode:     req.ExecEnabled,
		AutoContext:  autoContextEnabled,
		IsUserDebug:  req.IsUserDebug,
		IsApplyDebug: req.IsApplyDebug,
	}

	// log.Println("getTellSysPrompt - prompt params:", spew.Sdump(params))

	if currentStage.TellStage == shared.TellStagePlanning {
		if basicContextMsg != nil {
			basicContextMsg.CacheControl = &types.CacheControlSpec{
				Type: types.CacheControlTypeEphemeral,
			}
			sysParts = append(sysParts, *basicContextMsg)
		}

		if currentStage.PlanningPhase == shared.PlanningPhaseContext {
			log.Println("Planning phase is context -- adding auto context prompt")

			var txt string
			if req.IsChatOnly {
				txt = prompts.GetAutoContextChatPrompt(createPromptParams)
			} else {
				txt = prompts.GetAutoContextTellPrompt(createPromptParams)
			}

			sysParts = append(sysParts, types.ExtendedChatMessagePart{
				Type: openai.ChatMessagePartTypeText,
				Text: txt,
				CacheControl: &types.CacheControlSpec{
					Type: types.CacheControlTypeEphemeral,
				},
			})
		} else if currentStage.PlanningPhase == shared.PlanningPhasePlanning {

			var txt string
			if req.IsChatOnly {
				txt = prompts.GetChatSysPrompt(createPromptParams)
			} else {
				txt = prompts.GetPlanningPrompt(createPromptParams)
			}

			if len(state.subtasks) > 0 {
				sysParts = append(sysParts, types.ExtendedChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: txt,
				})
				sysParts = append(sysParts, types.ExtendedChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: state.formatSubtasks(),
					CacheControl: &types.CacheControlSpec{
						Type: types.CacheControlTypeEphemeral,
					},
				})
			} else {
				sysParts = append(sysParts, types.ExtendedChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: txt,
					CacheControl: &types.CacheControlSpec{
						Type: types.CacheControlTypeEphemeral,
					},
				})
			}

			if !req.IsChatOnly {
				if len(active.SkippedPaths) > 0 {
					skippedPrompt := prompts.SkippedPathsPrompt
					for skippedPath := range active.SkippedPaths {
						skippedPrompt += fmt.Sprintf("- %s\n", skippedPath)
					}
					sysParts = append(sysParts, types.ExtendedChatMessagePart{
						Type: openai.ChatMessagePartTypeText,
						Text: skippedPrompt,
					})
				}
			}
		}

		if autoContextMsg != nil {
			sysParts = append(sysParts, *autoContextMsg)
		}

		if smartContextMsg != nil {
			log.Println("smartContextMsg not supported during planning stage - only basic or auto context is supported")
			return nil, fmt.Errorf("smartContextMsg not supported during planning stage - only basic or auto context is supported")
		}

	} else if currentStage.TellStage == shared.TellStageImplementation {
		if state.currentSubtask == nil {
			return nil, fmt.Errorf("no current subtask during implementation stage")
		}

		if !smartContextEnabled && basicContextMsg != nil {
			basicContextMsg.CacheControl = &types.CacheControlSpec{
				Type: types.CacheControlTypeEphemeral,
			}
			sysParts = append(sysParts, *basicContextMsg)
		}

		if len(state.subtasks) > 0 {
			sysParts = append(sysParts, types.ExtendedChatMessagePart{
				Type: openai.ChatMessagePartTypeText,
				Text: prompts.GetImplementationPrompt(state.currentSubtask.Title),
			})
			sysParts = append(sysParts,
				types.ExtendedChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: state.formatSubtasks(),
					CacheControl: &types.CacheControlSpec{
						Type: types.CacheControlTypeEphemeral,
					},
				})
		} else {
			sysParts = append(sysParts, types.ExtendedChatMessagePart{
				Type: openai.ChatMessagePartTypeText,
				Text: prompts.GetImplementationPrompt(state.currentSubtask.Title),
				CacheControl: &types.CacheControlSpec{
					Type: types.CacheControlTypeEphemeral,
				},
			})
		}

		if !req.IsChatOnly {
			if len(active.SkippedPaths) > 0 {
				skippedPrompt := prompts.SkippedPathsPrompt
				for skippedPath := range active.SkippedPaths {
					skippedPrompt += fmt.Sprintf("- %s\n", skippedPath)
				}
				sysParts = append(sysParts, types.ExtendedChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: skippedPrompt,
				})
			}
		}

		if smartContextMsg != nil {
			sysParts = append(sysParts, *smartContextMsg)
		}

		if autoContextMsg != nil {
			log.Println("autoContextMsg not supported during implementation stage - only basic or smart context is supported")
			return nil, fmt.Errorf("autoContextMsg not supported during implementation stage - only basic or smart context is supported")
		}
	}

	return sysParts, nil
}
