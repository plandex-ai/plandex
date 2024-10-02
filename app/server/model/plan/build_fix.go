package plan

import (
	"encoding/json"
	"fmt"
	"log"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/types"
	"strings"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (fileState *activeBuildStreamFileState) fixFileLineNums() {
	auth := fileState.auth
	filePath := fileState.filePath
	activeBuild := fileState.activeBuild
	clients := fileState.clients
	planId := fileState.plan.Id
	branch := fileState.branch
	config := fileState.settings.ModelPack.GetAutoFix()
	incorrectlyUpdated := fileState.updated

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		log.Printf("fixFileLineNums - Active plan not found for plan ID %s and branch %s\n", planId, branch)
		return
	}

	// hash := sha256.Sum256([]byte(incorrectlyUpdated))
	// sha := hex.EncodeToString(hash[:])

	// log.Printf("%s - fixFileLineNums - Incorrectly updated content hash: %s\n", filePath, sha)

	log.Println("fixFileLineNums - getting file from model: " + filePath)
	// log.Println("File context:", fileContext)

	reasoning := ""
	if len(fileState.syntaxErrors) > 0 {
		reasoning += "The following are syntax errors identified by the tree-sitter library. Here are line numbers:\n\n" + strings.Join(fileState.syntaxErrors, "\n")
	}

	if fileState.verificationErrors != "" {
		if len(fileState.syntaxErrors) > 0 {
			reasoning += "\n\n"
			reasoning += "The following are other problems identified in the file:\n\n"
		} else {
			reasoning += "Here are the problems identified in the file:\n\n"
		}
		reasoning += fileState.verificationErrors
	}

	sysPrompt := prompts.GetBuildFixesLineNumbersSysPrompt(fileState.preBuildState, fmt.Sprintf("%s\n\n```%s```", activeBuild.FileDescription, activeBuild.FileContent), incorrectlyUpdated, reasoning)

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
		Auth: auth,
		Plan: fileState.plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  inputTokens,
			OutputTokens: shared.AvailableModelsByName[fileState.settings.ModelPack.GetAutoFix().BaseModelConfig.ModelName].DefaultReservedOutputTokens,
			ModelName:    fileState.settings.ModelPack.GetAutoFix().BaseModelConfig.ModelName,
		},
	})
	if apiErr != nil {
		activePlan.StreamDoneCh <- apiErr
		return
	}

	log.Println("fixFileLineNums - calling model for file: " + filePath)

	// for _, msg := range fileMessages {
	// 	log.Printf("%s: %s\n", msg.Role, msg.Content)
	// }

	var responseFormat *openai.ChatCompletionResponseFormat
	if config.BaseModelConfig.HasJsonResponseMode {
		responseFormat = &openai.ChatCompletionResponseFormat{Type: "json_object"}
	}

	// log.Println("responseFormat:", responseFormat)
	// log.Println("Model:", config.BaseModelConfig.ModelName)

	modelReq := openai.ChatCompletionRequest{
		Model: config.BaseModelConfig.ModelName,
		Tools: []openai.Tool{
			{
				Type:     "function",
				Function: &prompts.ListReplacementsFn,
			},
		},
		ToolChoice: openai.ToolChoice{
			Type: "function",
			Function: openai.ToolFunction{
				Name: prompts.ListReplacementsFn.Name,
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

		go fileState.listenStreamFixChanges(stream)
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
			log.Printf("Error building file '%s': %v\n", filePath, err)
			fileState.onBuildFileError(fmt.Errorf("error building file '%s': %v", filePath, err))
			return
		}

		var s string
		var res types.ChangesWithLineNums

		for _, choice := range resp.Choices {
			if len(choice.Message.ToolCalls) == 1 &&
				choice.Message.ToolCalls[0].Function.Name == prompts.ListReplacementsFn.Name {
				fnCall := choice.Message.ToolCalls[0].Function
				s = fnCall.Arguments
				break
			}
		}

		if s == "" {
			log.Println("no ListReplacements function call found in response")
			fileState.fixRetryOrAbort(fmt.Errorf("No ListReplacements function call found in response. This usually means the model failed to generate a valid response."))
			return
		}

		bytes := []byte(s)

		err = json.Unmarshal(bytes, &res)
		if err != nil {
			log.Printf("Error unmarshalling fix response: %v\n", err)
			fileState.fixRetryOrAbort(fmt.Errorf("Error unmarshalling fix response: %v | This usually means the model failed to generate valid JSON.", err))
			return
		}

		fileState.onFixResult(res)
	}
}
