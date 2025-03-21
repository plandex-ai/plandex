package plan

import (
	"errors"
	"fmt"
	"log"
	"plandex-server/model/prompts"
	"plandex-server/types"
	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

const AllTasksCompletedMsg = "All tasks have been completed. There is no current task to implement."

type getTellSysPromptParams struct {
	planStageSharedMsgs   []*types.ExtendedChatMessagePart
	planningPhaseOnlyMsgs []*types.ExtendedChatMessagePart
	implementationMsgs    []*types.ExtendedChatMessagePart
	contextTokenLimit     int
	dryRunWithoutContext  bool
}

func (state *activeTellStreamState) getTellSysPrompt(params getTellSysPromptParams) ([]types.ExtendedChatMessagePart, error) {
	planningSharedMsgs := params.planStageSharedMsgs
	plannerOnlyMsgs := params.planningPhaseOnlyMsgs
	implementationMsgs := params.implementationMsgs
	contextTokenLimit := params.contextTokenLimit
	req := state.req
	active := state.activePlan
	currentStage := state.currentStage

	sysParts := []types.ExtendedChatMessagePart{}

	createPromptParams := prompts.CreatePromptParams{
		ExecMode:          req.ExecEnabled,
		AutoContext:       req.AutoContext,
		IsUserDebug:       req.IsUserDebug,
		IsApplyDebug:      req.IsApplyDebug,
		IsGitRepo:         req.IsGitRepo,
		ContextTokenLimit: contextTokenLimit,
	}

	// log.Println("getTellSysPrompt - prompt params:", spew.Sdump(params))

	if currentStage.TellStage == shared.TellStagePlanning {
		if len(planningSharedMsgs) == 0 && !params.dryRunWithoutContext {
			log.Println("planningSharedMsgs is empty - required for planning stage")
			return nil, fmt.Errorf("planningSharedMsgs is empty - required for planning stage")
		}

		for _, msg := range planningSharedMsgs {
			sysParts = append(sysParts, *msg)
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
		} else if currentStage.PlanningPhase == shared.PlanningPhaseTasks {

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

		for _, msg := range plannerOnlyMsgs {
			sysParts = append(sysParts, *msg)
		}

		if len(implementationMsgs) > 0 {
			return nil, fmt.Errorf("implementationMsgs not supported during planning phase")
		}

	} else if currentStage.TellStage == shared.TellStageImplementation {
		if state.currentSubtask == nil {
			return nil, errors.New(AllTasksCompletedMsg)
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

		if implementationMsgs != nil {
			for _, msg := range implementationMsgs {
				sysParts = append(sysParts, *msg)
			}
		} else if !params.dryRunWithoutContext {
			log.Println("implementationMsgs is nil - required for implementation stage")
			return nil, fmt.Errorf("implementationMsgs is nil - required for implementation stage")
		}

		if planningSharedMsgs != nil {
			log.Println("planningSharedMsgs not supported during implementation stage - only basic or smart context is supported")
			return nil, fmt.Errorf("planningSharedMsgs not supported during implementation stage - only basic or smart context is supported")
		}
	}

	return sysParts, nil
}
