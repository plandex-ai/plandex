package plan

import (
	"fmt"
	"log"
	"math/rand"
	"plandex-server/db"
	diff_pkg "plandex-server/diff"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/types"
	"strings"
	"time"

	shared "plandex-shared"

	"github.com/davecgh/go-spew/spew"
	"github.com/sashabaranov/go-openai"
)

func (fileState *activeBuildStreamFileState) buildWholeFileFallback(proposedContent string, desc string) {
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
		return
	}

	sysPrompt := prompts.GetWholeFilePrompt(filePath, originalFile, desc, proposedContent)

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

	maxExpectedOutputTokens := shared.GetNumTokensEstimate(originalFile + proposedContent)
	modelConfig := config.GetRoleForOutputTokens(maxExpectedOutputTokens)

	inputTokens := model.GetMessagesTokenEstimate(messages...) + model.TokensPerRequest

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		Auth: auth,
		Plan: fileState.plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  inputTokens,
			OutputTokens: modelConfig.GetReservedOutputTokens(),
			ModelName:    modelConfig.BaseModelConfig.ModelName,
		},
	})
	if apiErr != nil {
		activePlan.StreamDoneCh <- apiErr
		return
	}

	log.Println("buildWholeFile - calling model for whole file write")

	var resp openai.ChatCompletionResponse
	var err error

	log.Println("buildWholeFile - modelConfig.BaseModelConfig.PredictedOutputEnabled:", modelConfig.BaseModelConfig.PredictedOutputEnabled)

	var prediction *types.OpenAIPrediction

	if modelConfig.BaseModelConfig.PredictedOutputEnabled {
		prediction = &types.OpenAIPrediction{
			Type: "content",
			Content: `
## Comments

No comments

<PlandexWholeFile>
` + originalFile + `
</PlandexWholeFile>
`,
		}
	}

	extendedReq := &types.ExtendedChatCompletionRequest{
		Model:       modelConfig.BaseModelConfig.ModelName,
		Messages:    messages,
		Temperature: modelConfig.Temperature,
		TopP:        modelConfig.TopP,
		Prediction:  prediction,
	}

	reqStarted := time.Now()
	fileState.builderRun.BuiltWholeFile = true
	fileState.builderRun.BuildWholeFileStartedAt = reqStarted

	resp, err = model.CreateChatCompletionWithRetries(clients, &config, activePlan.Ctx, *extendedReq)

	if err != nil {
		log.Printf("buildWholeFile - error calling model: %v\n", err)
		fileState.wholeFileRetryOrError(proposedContent, desc, fmt.Errorf("error calling model: %v", err))
		return
	}

	fileState.builderRun.GenerationIds = append(fileState.builderRun.GenerationIds, resp.ID)
	fileState.builderRun.BuildWholeFileFinishedAt = time.Now()

	log.Println("buildWholeFile - usage:")
	log.Println(spew.Sdump(resp.Usage))

	go func() {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: fileState.plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:    resp.Usage.PromptTokens,
				OutputTokens:   resp.Usage.CompletionTokens,
				ModelName:      modelConfig.BaseModelConfig.ModelName,
				ModelProvider:  modelConfig.BaseModelConfig.Provider,
				ModelPackName:  fileState.settings.ModelPack.Name,
				ModelRole:      shared.ModelRoleBuilder,
				Purpose:        "File edit (whole file)",
				GenerationId:   resp.ID,
				PlanId:         fileState.plan.Id,
				ModelStreamId:  fileState.tellState.activePlan.ModelStreamId,
				ConvoMessageId: fileState.convoMessageId,
				BuildId:        fileState.build.Id,

				RequestStartedAt: reqStarted,
				Streaming:        false,
				Req:              extendedReq,
				Res:              &resp,
				ModelConfig:      &config,
			},
		})

		if apiErr != nil {
			log.Printf("buildWholeFile - error executing DidSendModelRequest hook: %v", apiErr)
		}
	}()

	if len(resp.Choices) == 0 {
		log.Printf("buildWholeFile - no choices in response\n")
		fileState.wholeFileRetryOrError(proposedContent, desc, fmt.Errorf("no choices in response"))
		return
	}

	refsChoice := resp.Choices[0]
	content := refsChoice.Message.Content

	log.Printf("buildWholeFile - %s - content:\n%s\n", filePath, content)

	wholeFile := GetXMLContent(content, "PlandexWholeFile")

	if wholeFile == "" {
		log.Printf("buildWholeFile - no whole file found in response\n")
		fileState.wholeFileRetryOrError(proposedContent, desc, fmt.Errorf("no whole file found in response"))
		return
	}

	updatedFile := wholeFile

	buildInfo := &shared.BuildInfo{
		Path:      filePath,
		NumTokens: 0,
		Finished:  true,
	}

	log.Printf("streaming build info for whole file finished %s\n", filePath)

	activePlan.Stream(shared.StreamMessage{
		Type:      shared.StreamMessageBuildInfo,
		BuildInfo: buildInfo,
	})

	time.Sleep(50 * time.Millisecond)

	replacements, err := diff_pkg.GetDiffReplacements(originalFile, updatedFile)
	if err != nil {
		log.Printf("buildWholeFile - error getting diff replacements: %v\n", err)
		fileState.onBuildFileError(fmt.Errorf("error getting diff replacements: %v", err))
		return
	}

	for _, replacement := range replacements {
		replacement.Summary = strings.TrimSpace(desc)
	}

	res := db.PlanFileResult{
		TypeVersion:    1,
		OrgId:          fileState.plan.OrgId,
		PlanId:         fileState.plan.Id,
		PlanBuildId:    fileState.build.Id,
		ConvoMessageId: fileState.convoMessageId,
		Content:        "",
		Path:           filePath,
		Replacements:   replacements,
	}

	fileState.onFinishBuildFile(&res)
}

func (fileState *activeBuildStreamFileState) wholeFileRetryOrError(proposedContent string, desc string, err error) {
	if fileState.wholeFileNumRetry < MaxBuildErrorRetries {
		fileState.wholeFileNumRetry++

		log.Printf("buildWholeFile - retrying whole file file '%s' due to error: %v\n", fileState.filePath, err)

		activePlan := GetActivePlan(fileState.plan.Id, fileState.branch)

		if activePlan == nil {
			log.Printf("buildWholeFile - active plan not found for plan ID %s and branch %s\n", fileState.plan.Id, fileState.branch)
			fileState.onBuildFileError(fmt.Errorf("active plan not found for plan ID %s and branch %s", fileState.plan.Id, fileState.branch))
			return
		}

		select {
		case <-activePlan.Ctx.Done():
			log.Printf("buildWholeFile - context canceled\n")
			return
		case <-time.After(time.Duration(fileState.wholeFileNumRetry*fileState.wholeFileNumRetry)*200*time.Millisecond + time.Duration(rand.Intn(500))*time.Millisecond):
			break
		}

		fileState.buildWholeFileFallback(proposedContent, desc)
	} else {
		fileState.onBuildFileError(err)
	}

}
