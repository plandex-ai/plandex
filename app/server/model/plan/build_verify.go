package plan

import (
	"encoding/json"
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (fileState *activeBuildStreamFileState) verifyFileBuild() {
	auth := fileState.auth
	filePath := fileState.filePath
	planId := fileState.plan.Id
	branch := fileState.branch
	clients := fileState.clients
	config := fileState.settings.ModelPack.GetVerifier()
	updated := fileState.activeBuild.ToVerifyUpdatedState

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		log.Printf("Active plan not found for plan ID %s and branch %s\n", planId, branch)
		return
	}

	log.Printf("verifyFileBuild - Verifying file %s\n", filePath)

	// log.Println("File context:", fileContext)

	// log.Printf("preBuildState has content: %v\n", preBuildState != "")
	// log.Printf("updated has content: %v\n", updated != "")
	// log.Printf("activeBuild.FileDescription has content: %v\n", activeBuild.FileDescription != "")
	// log.Printf("activeBuild.FileContent has content: %v\n", activeBuild.FileContent != "")

	verifyState, err := fileState.GetVerifyState()

	if err != nil {
		log.Printf("Error getting verify state for file '%s': %v\n", filePath, err)
		fileState.onBuildFileError(fmt.Errorf("error getting verify state for file '%s': %v", filePath, err))
		return
	}

	if verifyState == nil {
		log.Printf("verifyFileBuild - Verify state not found for file '%s'\n", filePath)
		log.Println("finishing file build")
		fileState.onFinishBuildFile(nil, "")
		return
	}

	var diff string
	if verifyState.preBuildFileState != "" {
		diff, err = db.GetDiffsForBuild(verifyState.preBuildFileState, updated)

		if err != nil {
			log.Printf("Error getting diffs for file '%s': %v\n", filePath, err)
			fileState.onBuildFileError(fmt.Errorf("error getting diffs for file '%s': %v", filePath, err))
			return
		}
	}

	log.Println("verifyFileBuild - got diff for file: " + filePath)

	sysPrompt := prompts.GetVerifyPrompt(
		verifyState.preBuildFileState,
		updated,
		verifyState.proposedChanges,
		diff,
	)

	// log.Println("verifyFileBuild - verify prompt:\n", sysPrompt)

	fileMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: sysPrompt,
		},
	}

	promptTokens, err := shared.GetNumTokens(sysPrompt)

	if err != nil {
		log.Printf("Error getting num tokens for prompt: %v\n", err)
		fileState.onBuildFileError(fmt.Errorf("error getting num tokens for prompt: %v", err))
		return
	}

	inputTokens := prompts.ExtraTokensPerRequest + prompts.ExtraTokensPerMessage + promptTokens

	fileState.inputTokens = inputTokens

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		User:  auth.User,
		OrgId: auth.OrgId,
		Plan:  fileState.plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  inputTokens,
			OutputTokens: shared.AvailableModelsByName[fileState.settings.ModelPack.GetVerifier().BaseModelConfig.ModelName].DefaultReservedOutputTokens,
			ModelName:    fileState.settings.ModelPack.GetVerifier().BaseModelConfig.ModelName,
		},
	})
	if apiErr != nil {
		log.Printf("verifyFileBuild - Error executing will send model request hook: %v\n", apiErr)
		activePlan.StreamDoneCh <- apiErr
		return
	}

	log.Println("verifyFileBuild - Calling verify model for file: " + filePath)

	// for _, msg := range fileMessages {
	// 	log.Printf("%s: %s\n", msg.Role, msg.Content)
	// }

	var responseFormat *openai.ChatCompletionResponseFormat
	if config.BaseModelConfig.HasJsonResponseMode {
		responseFormat = &openai.ChatCompletionResponseFormat{Type: "json_object"}
	}

	modelReq := openai.ChatCompletionRequest{
		Model: config.BaseModelConfig.ModelName,
		Tools: []openai.Tool{
			{
				Type:     "function",
				Function: &prompts.VerifyOutputFn,
			},
		},
		ToolChoice: openai.ToolChoice{
			Type: "function",
			Function: openai.ToolFunction{
				Name: prompts.VerifyOutputFn.Name,
			},
		},
		Messages:       fileMessages,
		Temperature:    config.Temperature,
		TopP:           config.TopP,
		ResponseFormat: responseFormat,
		StreamOptions: &openai.StreamOptions{
			IncludeUsage: true,
		},
	}

	envVar := config.BaseModelConfig.ApiKeyEnvVar
	client := clients[envVar]

	if config.BaseModelConfig.HasStreamingFunctionCalls {
		stream, err := model.CreateChatCompletionStreamWithRetries(client, activePlan.Ctx, modelReq)
		if err != nil {
			log.Printf("Error creating plan file stream for path '%s': %v\n", filePath, err)
			fileState.onBuildFileError(fmt.Errorf("error creating plan file stream for path '%s': %v", filePath, err))
			return
		}

		go fileState.listenStreamVerifyOutput(stream)
	} else {
		buildInfo := &shared.BuildInfo{
			Path:      filePath,
			NumTokens: 0,
			Finished:  false,
		}
		activePlan.Stream(shared.StreamMessage{
			Type:      shared.StreamMessageBuildInfo,
			BuildInfo: buildInfo,
		})

		resp, err := model.CreateChatCompletionWithRetries(client, activePlan.Ctx, modelReq)

		if err != nil {
			log.Printf("Error verifying file '%s': %v\n", filePath, err)
			fileState.onBuildFileError(fmt.Errorf("error verifying file '%s': %v", filePath, err))
			return
		}

		var s string
		var res types.VerifyResult

		for _, choice := range resp.Choices {
			if len(choice.Message.ToolCalls) == 1 &&
				choice.Message.ToolCalls[0].Function.Name == prompts.VerifyOutputFn.Name {
				fnCall := choice.Message.ToolCalls[0].Function
				s = fnCall.Arguments
				break
			}
		}

		if s == "" {
			log.Println("no VerifyOutput function call found in response")
			fileState.verifyRetryOrAbort(fmt.Errorf("no Verify Output function call found in response"))
			return
		}

		bytes := []byte(s)

		err = json.Unmarshal(bytes, &res)
		if err != nil {
			log.Printf("Error unmarshalling verify response: %v\n", err)
			fileState.verifyRetryOrAbort(fmt.Errorf("error unmarshalling verify response: %v", err))
			return
		}

		fileState.onVerifyResult(res)
	}

}
