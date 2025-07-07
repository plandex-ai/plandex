package model

import (
	"context"
	"encoding/json"
	"fmt"
	"plandex-server/db"
	"plandex-server/model/prompts"
	"plandex-server/types"
	"plandex-server/utils"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

func GenPlanName(
	auth *types.ServerAuth,
	plan *db.Plan,
	settings *shared.PlanSettings,
	orgUserConfig *shared.OrgUserConfig,
	clients map[string]ClientInfo,
	authVars map[string]string,
	planContent string,
	sessionId string,
	ctx context.Context,
) (string, error) {
	config := settings.GetModelPack().Namer

	var tools []openai.Tool
	var toolChoice *openai.ToolChoice

	baseModelConfig := config.GetBaseModelConfig(authVars, settings, orgUserConfig)

	var sysPrompt string
	if baseModelConfig.PreferredOutputFormat == shared.ModelOutputFormatXml {
		sysPrompt = prompts.SysPlanNameXml
	} else {
		sysPrompt = prompts.SysPlanName
		tools = []openai.Tool{
			{
				Type:     "function",
				Function: &prompts.PlanNameFn,
			},
		}
		choice := openai.ToolChoice{
			Type: "function",
			Function: openai.ToolFunction{
				Name: prompts.PlanNameFn.Name,
			},
		}
		toolChoice = &choice
	}

	prompt := prompts.GetPlanNamePrompt(sysPrompt, planContent)

	messages := []types.ExtendedChatMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: prompt,
				},
			},
		},
	}

	modelRes, err := ModelRequest(ctx, ModelRequestParams{
		Clients:       clients,
		AuthVars:      authVars,
		Auth:          auth,
		Plan:          plan,
		ModelConfig:   &config,
		OrgUserConfig: orgUserConfig,
		Purpose:       "Plan name",
		Messages:      messages,
		Tools:         tools,
		ToolChoice:    toolChoice,
		SessionId:     sessionId,
		Settings:      settings,
	})

	if err != nil {
		fmt.Printf("Error during plan name model call: %v\n", err)
		return "", err
	}

	var planName string
	content := modelRes.Content

	if baseModelConfig.PreferredOutputFormat == shared.ModelOutputFormatXml {
		planName = utils.GetXMLContent(content, "planName")
		if planName == "" {
			return "", fmt.Errorf("No planName tag found in XML response")
		}
	} else {
		if content == "" {
			fmt.Println("no namePlan function call found in response")
			return "", fmt.Errorf("No namePlan function call found in response. The model failed to generate a valid response.")
		}

		var nameRes prompts.PlanNameRes
		err = json.Unmarshal([]byte(content), &nameRes)
		if err != nil {
			fmt.Printf("Error unmarshalling plan description response: %v\n", err)
			return "", err
		}
		planName = nameRes.PlanName
	}

	return planName, nil
}

type GenPipedDataNameParams struct {
	Ctx           context.Context
	Auth          *types.ServerAuth
	Plan          *db.Plan
	Settings      *shared.PlanSettings
	OrgUserConfig *shared.OrgUserConfig
	AuthVars      map[string]string
	SessionId     string
	Clients       map[string]ClientInfo
	PipedContent  string
}

func GenPipedDataName(
	params GenPipedDataNameParams,
) (string, error) {
	ctx := params.Ctx
	auth := params.Auth
	plan := params.Plan
	settings := params.Settings
	clients := params.Clients
	authVars := params.AuthVars
	pipedContent := params.PipedContent
	sessionId := params.SessionId
	orgUserConfig := params.OrgUserConfig

	config := settings.GetModelPack().Namer

	var sysPrompt string
	var tools []openai.Tool
	var toolChoice *openai.ToolChoice

	baseModelConfig := config.GetBaseModelConfig(authVars, settings, orgUserConfig)

	if baseModelConfig.PreferredOutputFormat == shared.ModelOutputFormatXml {
		sysPrompt = prompts.SysPipedDataNameXml
	} else {
		sysPrompt = prompts.SysPipedDataName
		tools = []openai.Tool{
			{
				Type:     "function",
				Function: &prompts.PipedDataNameFn,
			},
		}
		choice := openai.ToolChoice{
			Type: "function",
			Function: openai.ToolFunction{
				Name: prompts.PipedDataNameFn.Name,
			},
		}
		toolChoice = &choice
	}

	prompt := prompts.GetPipedDataNamePrompt(sysPrompt, pipedContent)

	messages := []types.ExtendedChatMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: prompt,
				},
			},
		},
	}

	modelRes, err := ModelRequest(ctx, ModelRequestParams{
		Clients:       clients,
		Auth:          auth,
		AuthVars:      authVars,
		Plan:          plan,
		ModelConfig:   &config,
		Purpose:       "Piped data name",
		Messages:      messages,
		Tools:         tools,
		ToolChoice:    toolChoice,
		SessionId:     sessionId,
		Settings:      settings,
		OrgUserConfig: orgUserConfig,
	})

	if err != nil {
		fmt.Printf("Error during piped data name model call: %v\n", err)
		return "", err
	}

	var name string
	content := modelRes.Content

	if baseModelConfig.PreferredOutputFormat == shared.ModelOutputFormatXml {
		name = utils.GetXMLContent(content, "name")
		if name == "" {
			return "", fmt.Errorf("No name tag found in XML response")
		}
	} else {
		if content == "" {
			fmt.Println("no namePipedData function call found in response")
			return "", fmt.Errorf("No namePipedData function call found in response. The model failed to generate a valid response.")
		}

		var nameRes prompts.PipedDataNameRes
		err = json.Unmarshal([]byte(content), &nameRes)
		if err != nil {
			fmt.Printf("Error unmarshalling piped data name response: %v\n", err)
			return "", err
		}
		name = nameRes.Name
	}

	return name, nil
}

func GenNoteName(
	ctx context.Context,
	auth *types.ServerAuth,
	plan *db.Plan,
	settings *shared.PlanSettings,
	orgUserConfig *shared.OrgUserConfig,
	clients map[string]ClientInfo,
	authVars map[string]string,
	note string,
	sessionId string,
) (string, error) {
	config := settings.GetModelPack().Namer

	var sysPrompt string
	var tools []openai.Tool
	var toolChoice *openai.ToolChoice

	baseModelConfig := config.GetBaseModelConfig(authVars, settings, orgUserConfig)

	if baseModelConfig.PreferredOutputFormat == shared.ModelOutputFormatXml {
		sysPrompt = prompts.SysNoteNameXml
	} else {
		sysPrompt = prompts.SysNoteName
		tools = []openai.Tool{
			{
				Type:     "function",
				Function: &prompts.NoteNameFn,
			},
		}
		choice := openai.ToolChoice{
			Type: "function",
			Function: openai.ToolFunction{
				Name: prompts.NoteNameFn.Name,
			},
		}
		toolChoice = &choice
	}

	prompt := prompts.GetNoteNamePrompt(sysPrompt, note)

	messages := []types.ExtendedChatMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: prompt,
				},
			},
		},
	}

	modelRes, err := ModelRequest(ctx, ModelRequestParams{
		Clients:       clients,
		Auth:          auth,
		AuthVars:      authVars,
		Plan:          plan,
		ModelConfig:   &config,
		Purpose:       "Note name",
		Messages:      messages,
		Tools:         tools,
		ToolChoice:    toolChoice,
		SessionId:     sessionId,
		Settings:      settings,
		OrgUserConfig: orgUserConfig,
	})

	if err != nil {
		fmt.Printf("Error during note name model call: %v\n", err)
		return "", err
	}

	var name string
	content := modelRes.Content

	if baseModelConfig.PreferredOutputFormat == shared.ModelOutputFormatXml {
		name = utils.GetXMLContent(content, "name")
		if name == "" {
			return "", fmt.Errorf("No name tag found in XML response")
		}
	} else {
		if content == "" {
			fmt.Println("no nameNote function call found in response")
			return "", fmt.Errorf("No nameNote function call found in response. The model failed to generate a valid response.")
		}

		var nameRes prompts.NoteNameRes
		err = json.Unmarshal([]byte(content), &nameRes)
		if err != nil {
			fmt.Printf("Error unmarshalling note name response: %v\n", err)
			return "", err
		}
		name = nameRes.Name
	}

	return name, nil
}
