package prompts

import (
	"fmt"

	"github.com/plandex/plandex/shared"
)

func init() {
	var err error

	AutoContextPreambleNumTokens, err = shared.GetNumTokens(AutoContextPreamble)

	if err != nil {
		panic(fmt.Sprintf("Error getting number of tokens for context preamble: %v", err))
	}

	SysPlanningBasicTokens, err = shared.GetNumTokens(SysPlanningBasic)

	if err != nil {
		panic(fmt.Sprintf("Error getting number of tokens for sys msg: %v", err))
	}

	SysPlanningAutoContextTokens, err = shared.GetNumTokens(SysPlanningAutoContext)

	if err != nil {
		panic(fmt.Sprintf("Error getting number of tokens for sys msg: %v", err))
	}

	PlanningPromptWrapperTokens, err = shared.GetNumTokens(fmt.Sprintf(planningPromptWrapperFormatStr, "", "", "", ""))

	if err != nil {
		panic(fmt.Sprintf("Error getting number of tokens for prompt wrapper: %v", err))
	}

	ImplementationPromptWrapperTokens, err = shared.GetNumTokens(fmt.Sprintf(implementationPromptWrapperFormatStr, "", "", "", ""))

	if err != nil {
		panic(fmt.Sprintf("Error getting number of tokens for prompt wrapper: %v", err))
	}

	AutoContinuePromptTokens, err = shared.GetNumTokens(AutoContinuePrompt)

	if err != nil {
		panic(fmt.Sprintf("Error getting number of tokens for auto continue prompt: %v", err))
	}

	DebugPromptTokens, err = shared.GetNumTokens(DebugPrompt)

	if err != nil {
		panic(fmt.Sprintf("Error getting number of tokens for debug prompt: %v", err))
	}

	ChatOnlyPromptTokens, err = shared.GetNumTokens(ChatOnlyPrompt)

	if err != nil {
		panic(fmt.Sprintf("Error getting number of tokens for chat only prompt: %v", err))
	}

	ApplyScriptPlanningPromptNumTokens, err = shared.GetNumTokens(ApplyScriptPlanningPrompt)

	if err != nil {
		panic(fmt.Sprintf("Error getting number of tokens for apply script planning prompt: %v", err))
	}

	ApplyScriptImplementationPromptNumTokens, err = shared.GetNumTokens(ApplyScriptImplementationPrompt)

	if err != nil {
		panic(fmt.Sprintf("Error getting number of tokens for apply script prompt: %v", err))
	}

	ApplyScriptSummaryNumTokens, err = shared.GetNumTokens(ApplyScriptPromptSummary)

	if err != nil {
		panic(fmt.Sprintf("Error getting number of tokens for apply script prompt summary: %v", err))
	}

	NoApplyScriptPromptNumTokens, err = shared.GetNumTokens(NoApplyScriptPrompt)

	if err != nil {
		panic(fmt.Sprintf("Error getting number of tokens for no execute script prompt: %v", err))
	}
}
