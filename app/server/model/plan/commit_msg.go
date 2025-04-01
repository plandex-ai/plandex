package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/notify"
	"plandex-server/types"
	"plandex-server/utils"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) genPlanDescription() (*db.ConvoMessageDescription, *shared.ApiError) {
	auth := state.auth
	plan := state.plan
	planId := plan.Id
	branch := state.branch
	settings := state.settings
	clients := state.clients
	config := settings.ModelPack.CommitMsg

	activePlan := GetActivePlan(planId, branch)
	if activePlan == nil {
		go notify.NotifyErr(notify.SeverityError, fmt.Errorf("active plan not found for plan %s and branch %s", planId, branch))

		return nil, &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    fmt.Sprintf("active plan not found for plan %s and branch %s", planId, branch),
		}
	}

	var sysPrompt string
	var tools []openai.Tool
	var toolChoice *openai.ToolChoice

	if config.BaseModelConfig.PreferredModelOutputFormat == shared.ModelOutputFormatXml {
		sysPrompt = prompts.SysDescribeXml
	} else {
		sysPrompt = prompts.SysDescribe
		tools = []openai.Tool{
			{
				Type:     "function",
				Function: &prompts.DescribePlanFn,
			},
		}
		choice := openai.ToolChoice{
			Type: "function",
			Function: openai.ToolFunction{
				Name: prompts.DescribePlanFn.Name,
			},
		}
		toolChoice = &choice
	}

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
		{
			Role: openai.ChatMessageRoleAssistant,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: activePlan.CurrentReplyContent,
				},
			},
		},
	}

	reqParams := model.ModelRequestParams{
		Clients:        clients,
		Auth:           auth,
		Plan:           plan,
		ModelConfig:    &config,
		Purpose:        "Response summary",
		Messages:       messages,
		ModelStreamId:  state.modelStreamId,
		ConvoMessageId: state.replyId,
		SessionId:      activePlan.SessionId,
	}

	if tools != nil {
		reqParams.Tools = tools
	}
	if toolChoice != nil {
		reqParams.ToolChoice = toolChoice
	}

	modelRes, err := model.ModelRequest(activePlan.Ctx, reqParams)

	if err != nil {
		go notify.NotifyErr(notify.SeverityError, fmt.Errorf("error during plan description model call: %v", err))

		return nil, &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    fmt.Sprintf("error during plan description model call: %v", err),
		}
	}

	log.Println("Plan description model call complete")

	content := modelRes.Content

	var commitMsg string

	if config.BaseModelConfig.PreferredModelOutputFormat == shared.ModelOutputFormatXml {
		commitMsg = utils.GetXMLContent(content, "commitMsg")
		if commitMsg == "" {
			go notify.NotifyErr(notify.SeverityError, fmt.Errorf("no commitMsg tag found in XML response"))

			return nil, &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "No commitMsg tag found in XML response",
			}
		}
	} else {

		if content == "" {
			fmt.Println("no describePlan function call found in response")

			go notify.NotifyErr(notify.SeverityError, fmt.Errorf("no describePlan function call found in response"))

			return nil, &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "No describePlan function call found in response. The model failed to generate a valid response.",
			}
		}

		var desc shared.ConvoMessageDescription
		err = json.Unmarshal([]byte(content), &desc)
		if err != nil {
			fmt.Printf("Error unmarshalling plan description response: %v\n", err)

			go notify.NotifyErr(notify.SeverityError, fmt.Errorf("error unmarshalling plan description response: %v", err))

			return nil, &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    fmt.Sprintf("error unmarshalling plan description response: %v", err),
			}
		}
		commitMsg = desc.CommitMsg
	}

	return &db.ConvoMessageDescription{
		PlanId:    planId,
		CommitMsg: commitMsg,
	}, nil
}

func GenCommitMsgForPendingResults(auth *types.ServerAuth, plan *db.Plan, clients map[string]model.ClientInfo, settings *shared.PlanSettings, current *shared.CurrentPlanState, sessionId string, ctx context.Context) (string, error) {
	config := settings.ModelPack.CommitMsg

	s := ""

	num := 0
	for _, desc := range current.ConvoMessageDescriptions {
		if desc.WroteFiles && desc.DidBuild && len(desc.BuildPathsInvalidated) == 0 && desc.AppliedAt == nil {
			s += desc.CommitMsg + "\n"
			num++
		}
	}

	if num <= 1 {
		return s, nil
	}

	prompt := "Pending changes:\n\n" + s

	messages := []types.ExtendedChatMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: prompts.SysPendingResults,
				},
			},
		},
		{
			Role: openai.ChatMessageRoleUser,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: prompt,
				},
			},
		},
	}

	modelRes, err := model.ModelRequest(ctx, model.ModelRequestParams{
		Clients:     clients,
		Auth:        auth,
		Plan:        plan,
		ModelConfig: &config,
		Purpose:     "Commit message",
		Messages:    messages,
		SessionId:   sessionId,
	})

	if err != nil {
		fmt.Println("Generate commit message error:", err)

		return "", err
	}

	content := modelRes.Content

	if content == "" {
		return "", fmt.Errorf("no response from model")
	}

	return content, nil
}
