package plan

import (
	"fmt"
	"log"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"strings"

	"github.com/sashabaranov/go-openai"
)

func (fileState *activeBuildStreamFileState) fixFileLineNums() {
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
	}

	envVar := config.BaseModelConfig.ApiKeyEnvVar
	client := clients[envVar]

	stream, err := model.CreateChatCompletionStreamWithRetries(client, activePlan.Ctx, modelReq)
	if err != nil {
		log.Printf("Error creating plan file stream for path '%s': %v\n", filePath, err)
		fileState.onBuildFileError(fmt.Errorf("error creating plan file stream for path '%s': %v", filePath, err))
		return
	}

	go fileState.listenStreamFixChanges(stream)
}
