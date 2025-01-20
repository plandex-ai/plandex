package plan

import (
	"log"
	"plandex-server/model/prompts"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) handleMissingFileResponse(applyScriptSummary string) bool {
	missingFileResponse := state.missingFileResponse
	planId := state.plan.Id
	branch := state.branch
	req := state.req
	isFollowUp := state.isFollowUp
	isPlanningStage := state.isPlanningStage

	active := GetActivePlan(planId, branch)

	if active == nil {
		log.Printf("execTellPlan: Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return false
	}

	log.Println("Missing file response:", missingFileResponse, "setting replyParser")
	// log.Printf("Current reply content:\n%s\n", active.CurrentReplyContent)

	state.replyParser.AddChunk(active.CurrentReplyContent, true)
	res := state.replyParser.Read()
	currentFile := res.CurrentFilePath

	log.Printf("Current file: %s\n", currentFile)
	// log.Println("Current reply content:\n", active.CurrentReplyContent)

	replyContent := active.CurrentReplyContent
	numTokens := active.NumTokens

	if missingFileResponse == shared.RespondMissingFileChoiceSkip {
		replyBeforeCurrentFile := state.replyParser.GetReplyBeforeCurrentPath()
		numTokens = shared.GetNumTokensEstimate(replyBeforeCurrentFile)

		replyContent = replyBeforeCurrentFile
		state.replyParser = types.NewReplyParser()
		state.replyParser.AddChunk(replyContent, true)

		UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
			ap.CurrentReplyContent = replyContent
			ap.NumTokens = numTokens
			ap.SkippedPaths[currentFile] = true
		})

	} else {
		if missingFileResponse == shared.RespondMissingFileChoiceOverwrite {
			UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
				ap.AllowOverwritePaths[currentFile] = true
			})
		}
	}

	state.messages = append(state.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: active.CurrentReplyContent,
	})

	if missingFileResponse == shared.RespondMissingFileChoiceSkip {
		res := state.replyParser.FinishAndRead()
		skipPrompt := prompts.GetSkipMissingFilePrompt(res.CurrentFilePath)
		prompt := prompts.GetWrappedPrompt(skipPrompt, req.OsDetails, applyScriptSummary, isPlanningStage, isFollowUp) + "\n\n" + skipPrompt // repetition of skip prompt to improve instruction following

		skipPromptTokens := shared.GetNumTokensEstimate(skipPrompt)
		state.totalRequestTokens += skipPromptTokens

		state.messages = append(state.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		})

	} else {
		missingPrompt := prompts.GetMissingFileContinueGeneratingPrompt(res.CurrentFilePath)
		prompt := prompts.GetWrappedPrompt(missingPrompt, req.OsDetails, applyScriptSummary, isPlanningStage, isFollowUp) + "\n\n" + missingPrompt // repetition of missing prompt to improve instruction following

		promptTokens := shared.GetNumTokensEstimate(prompt)
		state.totalRequestTokens += promptTokens

		state.messages = append(state.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		})
	}

	return true
}
