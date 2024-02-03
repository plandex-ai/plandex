package plan

import (
	"context"
	"encoding/json"
	"log"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/sashabaranov/go-openai"
)

func ExecStatusShouldContinue(client *openai.Client, message string, ctx context.Context) (bool, error) {
	// First try to determine if the plan should continue based on the last paragraph without calling the model
	paragraphs := strings.Split(message, "\n\n")
	lastParagraph := paragraphs[len(paragraphs)-1]
	if lastParagraph != "" {
		if strings.Contains(lastParagraph, "All tasks have been completed") ||
			strings.Contains(lastParagraph, "all tasks have been completed") ||
			strings.Contains(lastParagraph, "plan cannot be continued") {
			return false, nil
		}

		if strings.Index(lastParagraph, "Next, ") < 3 {
			return true, nil
		}
	}

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: prompts.GetExecStatusShouldContinue(message), // Ensure this function is correctly defined in your package
		},
	}

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:          model.PlanExecStatusModel,
			Functions:      []openai.FunctionDefinition{prompts.ShouldAutoContinueFn},
			Messages:       messages,
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
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
		ShouldContinue bool `json:"shouldContinue"`
	}

	for _, choice := range resp.Choices {
		if choice.FinishReason == "function_call" &&
			choice.Message.FunctionCall != nil &&
			choice.Message.FunctionCall.Name == "shouldAutoContinue" {
			fnCall := choice.Message.FunctionCall
			strRes = fnCall.Arguments
		}
	}

	if strRes == "" {
		log.Println("No shouldAutoContinue function call found in response")
		spew.Dump(resp)

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

	return res.ShouldContinue, nil
}
