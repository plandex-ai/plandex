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
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
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

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: sysPrompt,
		},
	}

	inputTokens := shared.GetMessagesTokenEstimate(messages...) + shared.TokensPerRequest

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		Auth: auth,
		Plan: fileState.plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  inputTokens,
			OutputTokens: config.GetReservedOutputTokens(),
			ModelName:    config.BaseModelConfig.ModelName,
		},
	})
	if apiErr != nil {
		activePlan.StreamDoneCh <- apiErr
		return
	}

	log.Println("buildWholeFile - calling model for applied changes validation")

	var resp openai.ChatCompletionResponse
	var err error

	modelReq := &openai.ChatCompletionRequest{
		Model:       config.BaseModelConfig.ModelName,
		Messages:    messages,
		Temperature: config.Temperature,
		TopP:        config.TopP,
	}

	log.Println("buildWholeFile - config.BaseModelConfig.PredictedOutputEnabled:", config.BaseModelConfig.PredictedOutputEnabled)

	if config.BaseModelConfig.PredictedOutputEnabled {
		extendedReq := &model.ExtendedChatCompletionRequest{
			ChatCompletionRequest: modelReq,
			Prediction: &model.OpenAIPrediction{
				Type: "content",
				Content: `
## Comments

No comments

<PlandexWholeFile>
` + originalFile + `
</PlandexWholeFile>
`,
			},
		}
		resp, err = model.CreateChatCompletionWithRetries(clients, &config, activePlan.Ctx, *extendedReq)
	} else {
		resp, err = model.CreateChatCompletionWithRetries(clients, &config, activePlan.Ctx, *modelReq)
	}

	if err != nil {
		log.Printf("buildWholeFile - error calling model: %v\n", err)
		fileState.wholeFileRetryOrError(proposedContent, desc, fmt.Errorf("error calling model: %v", err))
		return
	}

	log.Println("buildWholeFile - usage:")
	log.Println(spew.Sdump(resp.Usage))

	go func() {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: fileState.plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:    resp.Usage.PromptTokens,
				OutputTokens:   resp.Usage.CompletionTokens,
				ModelName:      config.BaseModelConfig.ModelName,
				ModelProvider:  config.BaseModelConfig.Provider,
				ModelPackName:  fileState.settings.ModelPack.Name,
				ModelRole:      shared.ModelRoleBuilder,
				Purpose:        "File edit (whole file)",
				GenerationId:   resp.ID,
				PlanId:         fileState.plan.Id,
				ModelStreamId:  fileState.tellState.activePlan.ModelStreamId,
				ConvoMessageId: fileState.convoMessageId,
				BuildId:        fileState.build.Id,
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
