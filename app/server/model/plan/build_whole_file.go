package plan

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/types"
	"plandex-server/utils"
	"time"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

func (fileState *activeBuildStreamFileState) buildWholeFileFallback(buildCtx context.Context, proposedContent string, desc string, comments string, sessionId string) (string, error) {
	auth := fileState.auth
	filePath := fileState.filePath
	clients := fileState.clients
	planId := fileState.plan.Id
	branch := fileState.branch
	originalFile := fileState.preBuildState
	config := fileState.settings.ModelPack.GetWholeFileBuilder()

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		log.Printf("Active plan not found for plan ID %s and branch %s\n", planId, branch)
		fileState.onBuildFileError(fmt.Errorf("active plan not found for plan ID %s and branch %s", planId, branch))
		return "", fmt.Errorf("active plan not found for plan ID %s and branch %s", planId, branch)
	}

	originalFileWithLineNums := shared.AddLineNums(originalFile)
	proposedContentWithLineNums := shared.AddLineNums(proposedContent)

	sysPrompt, headNumTokens := prompts.GetWholeFilePrompt(filePath, originalFileWithLineNums, proposedContentWithLineNums, desc, comments)

	messages := []types.ExtendedChatMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: sysPrompt,
				},
			},
		},
	}

	inputTokens := model.GetMessagesTokenEstimate(messages...) + model.TokensPerRequest
	maxExpectedOutputTokens := shared.GetNumTokensEstimate(originalFile + proposedContent)

	modelConfig := config.GetRoleForInputTokens(inputTokens)
	modelConfig = modelConfig.GetRoleForOutputTokens(maxExpectedOutputTokens)

	log.Println("buildWholeFile - calling model for whole file write")

	log.Println("buildWholeFile - modelConfig.BaseModelConfig.PredictedOutputEnabled:", modelConfig.BaseModelConfig.PredictedOutputEnabled)

	var prediction string

	if modelConfig.BaseModelConfig.PredictedOutputEnabled && comments != "" {
		prediction = `
<PlandexWholeFile>
` + originalFile + `
</PlandexWholeFile>
`

	}

	// This allows proper accounting for cached input tokens even when the stream is cancelled -- OpenAI only for now
	var willCacheNumTokens int
	if modelConfig.BaseModelConfig.Provider == shared.ModelProviderOpenAI {
		willCacheNumTokens = headNumTokens
	}

	log.Println("buildWholeFile - calling model.ModelRequest")
	// spew.Dump(messages)

	modelRes, err := model.ModelRequest(buildCtx, model.ModelRequestParams{
		Clients:     clients,
		Auth:        auth,
		Plan:        fileState.plan,
		ModelConfig: &config,
		Purpose:     "File edit",

		Messages:   messages,
		Prediction: prediction,

		ModelStreamId:  fileState.modelStreamId,
		ConvoMessageId: fileState.convoMessageId,
		BuildId:        fileState.build.Id,

		BeforeReq: func() {
			fileState.builderRun.BuiltWholeFile = true
			fileState.builderRun.BuildWholeFileStartedAt = time.Now()
		},

		AfterReq: func() {
			fileState.builderRun.BuildWholeFileFinishedAt = time.Now()
		},

		WillCacheNumTokens:    willCacheNumTokens,
		EstimatedOutputTokens: maxExpectedOutputTokens,

		SessionId: sessionId,
	})

	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Printf("buildWholeFileFallback - context canceled during model request for file %s", filePath)
			return "", err
		}

		return "", fmt.Errorf("error calling model: %v", err)
	}

	fileState.builderRun.GenerationIds = append(fileState.builderRun.GenerationIds, modelRes.GenerationId)
	fileState.builderRun.BuildWholeFileFinishedAt = time.Now()

	content := modelRes.Content

	// log.Printf("buildWholeFile - %s - content:\n%s\n", filePath, content)

	wholeFile := utils.GetXMLContent(content, "PlandexWholeFile")

	if wholeFile == "" {
		log.Printf("buildWholeFile - no whole file found in response\n")
		return fileState.wholeFileRetryOrError(buildCtx, proposedContent, desc, comments, sessionId, fmt.Errorf("no whole file found in response"))
	}

	return wholeFile, nil
}

func (fileState *activeBuildStreamFileState) wholeFileRetryOrError(buildCtx context.Context, proposedContent string, desc string, comments string, sessionId string, err error) (string, error) {
	if fileState.wholeFileNumRetry < MaxBuildErrorRetries {
		fileState.wholeFileNumRetry++

		log.Printf("buildWholeFile - retrying whole file file '%s' due to error: %v\n", fileState.filePath, err)

		activePlan := GetActivePlan(fileState.plan.Id, fileState.branch)

		if activePlan == nil {
			log.Printf("buildWholeFile - active plan not found for plan ID %s and branch %s\n", fileState.plan.Id, fileState.branch)
			// fileState.onBuildFileError(fmt.Errorf("active plan not found for plan ID %s and branch %s", fileState.plan.Id, fileState.branch))
			return "", fmt.Errorf("active plan not found for plan ID %s and branch %s", fileState.plan.Id, fileState.branch)
		}

		select {
		case <-buildCtx.Done():
			log.Printf("buildWholeFile - context canceled\n")
			return "", context.Canceled
		case <-time.After(time.Duration(fileState.wholeFileNumRetry*fileState.wholeFileNumRetry)*200*time.Millisecond + time.Duration(rand.Intn(500))*time.Millisecond):
			break
		}

		return fileState.buildWholeFileFallback(buildCtx, proposedContent, desc, comments, sessionId)
	} else {
		// fileState.onBuildFileError(err)
		return "", err
	}

}
