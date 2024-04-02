package plan

import (
	"context"
	"encoding/json"
	"log"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func ExecStatusShouldContinue(client *openai.Client, config shared.TaskRoleConfig, prompt, message string, ctx context.Context) (bool, error) {
	log.Println("Checking if plan should continue based on exec status")

	// First try to determine if the plan should continue based on the last paragraph without calling the model
	paragraphs := strings.Split(message, "\n\n")
	lastParagraph := paragraphs[len(paragraphs)-1]
	lastParagraphLower := strings.ToLower(lastParagraph)

	// log.Printf("Last paragraph: %s\n", lastParagraphLower)

	if lastParagraphLower != "" {
		if strings.Contains(lastParagraphLower, "all tasks have been completed") ||
			strings.Contains(lastParagraphLower, "plan cannot be continued") {
			log.Println("Plan cannot be continued based on last paragraph")
			return false, nil
		}

		nextIdx := strings.Index(lastParagraph, "Next, ")
		if nextIdx >= 0 {
			log.Println("Plan can be continued based on last paragraph")
			return true, nil
		}
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.GetExecStatusShouldContinue(prompt, message),
		},
	}

	log.Println("Calling model to check if plan should continue")

	resp, err := model.CreateChatCompletionWithRetries(
		client,
		ctx,
		openai.ChatCompletionRequest{
			Model: config.BaseModelConfig.ModelName,
			Tools: []openai.Tool{
				{
					Type:     "function",
					Function: prompts.ShouldAutoContinueFn,
				},
			},
			ToolChoice: openai.ToolChoice{
				Type: "function",
				Function: openai.ToolFunction{
					Name: prompts.ShouldAutoContinueFn.Name,
				},
			},
			Messages:       messages,
			ResponseFormat: config.OpenAIResponseFormat,
			Temperature:    config.Temperature,
			TopP:           config.TopP,
		},
	)

	if err != nil {
		log.Printf("Error during plan exec status check model call: %v\n", err)
		// return false, fmt.Errorf("error during plan exec status check model call: %v", err)

		// Instead of erroring out, just don't continue the plan
		return false, nil
	}

	var strRes string
	var res struct {
		Reasoning      string `json:"reasoning"`
		ShouldContinue bool   `json:"shouldContinue"`
	}

	for _, choice := range resp.Choices {
		if len(choice.Message.ToolCalls) == 1 &&
			choice.Message.ToolCalls[0].Function.Name == prompts.ShouldAutoContinueFn.Name {
			fnCall := choice.Message.ToolCalls[0].Function
			strRes = fnCall.Arguments
			break
		}
	}

	if strRes == "" {
		log.Println("No shouldAutoContinue function call found in response")
		log.Println(spew.Sdump(resp))

		// return false, fmt.Errorf("no shouldAutoContinue function call found in response")

		// Instead of erroring out, just don't continue the plan
		return false, nil
	}

	err = json.Unmarshal([]byte(strRes), &res)
	if err != nil {
		log.Printf("Error unmarshalling plan exec status response: %v\n", err)

		// return false, fmt.Errorf("error unmarshalling plan exec status response: %v", err)

		// Instead of erroring out, just don't continue the plan
		return false, nil
	}

	log.Printf("Plan exec status response: %v\n", res)

	return res.ShouldContinue, nil
}
