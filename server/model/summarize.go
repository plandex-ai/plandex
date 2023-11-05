package model

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func ShortSummary(text string) ([]byte, error) {
	resp, err := Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Functions: []openai.FunctionDefinition{{
				Name: "summarize",
				Parameters: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"summary": {
							Type:        jsonschema.String,
							Description: "A brief summary of the text. A few sentences at most",
						},
					},
					Required: []string{"summary"},
				},
			}},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an AI summarizer that summarizes text, including programs, documentation, websites, and more. Most text will be related to software development.",
				},
				{
					Role: openai.ChatMessageRoleUser,
					Content: (`
						 Please summarize the text below using the 'summarize' function. Only call the 'summarize' function in your reponse. Don't call any other function.

						 Text:

					` + text),
				},
			},
		},
	)

	if err != nil {
		return nil, err
	}

	var byteRes []byte
	for _, choice := range resp.Choices {
		if choice.FinishReason == "function_call" && choice.Message.FunctionCall != nil && choice.Message.FunctionCall.Name == "summarize" {
			fnCall := choice.Message.FunctionCall

			if strings.HasSuffix(fnCall.Arguments, ",\n}") { // remove trailing comma
				fnCall.Arguments = strings.TrimSuffix(fnCall.Arguments, ",\n}") + "\n}"
			}

			byteRes = []byte(fnCall.Arguments)
		}
	}

	if len(byteRes) == 0 {
		return nil, fmt.Errorf("no summarize function call found in response")
	}

	// validate the JSON response
	var summarizeResp shared.ShortSummaryResponse
	if err := json.Unmarshal(byteRes, &summarizeResp); err != nil {
		return nil, err
	}

	return byteRes, nil
}

func PlanSummary(conversation []openai.ChatCompletionMessage, lastMessageTimestamp string) (*shared.ConversationSummary, error) {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: shared.IdentityPrompt,
		},
	}

	messages = append(messages, conversation...)

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: convoSummaryPrompt,
	})

	// fmt.Println("summarizing messages:")
	// spew.Dump(messages)

	resp, err := Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			// Model:    openai.GPT3Dot5Turbo16K,
			Messages: messages,
		},
	)

	if err != nil {
		fmt.Println("PlanSummary err:", err)

		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from GPT")
	}

	content := resp.Choices[0].Message.Content

	return &shared.ConversationSummary{
		Summary:              content,
		Tokens:               resp.Usage.CompletionTokens,
		LastMessageTimestamp: lastMessageTimestamp,
	}, nil

}

const convoSummaryPrompt = `
Based on the conversation so far, make a summary of the current state of the plan. 

- It should begin with a summary of the user's messages, with particular focus any tasks they have given you. Your summary of the tasks should reflect the latest version of each task--if they have changed over time, summarize the latest state of each task that was given and omit anything that is now obsolete. Condense this information as much as possible while still being clear and retaining the meaning of the original messages. If the user has sent messages like 'continue to the next step of the plan' that don't contain any new information relevant to the plan, you should omit them.

- Next, if the plan includes a statement from the assistant to the effect of "I will break this large task into subtasks" and the plan has been broken down into subtasks, include those subtasks in the summary as a numbered list. Condense these as much as possible while still being clear and retaining the meaning of each subtask. At the end of the list, state which subtask is currently being worked on (unless they are all finished, in which case state that they are all finished).

- Last, summarize the latest version of the plan and any changes you have suggested. If the some of the older changes have been overridden by newer changes, you should only include the newest changes and omit the older ones. 

- If your responses include code blocks labelled with file paths, include the latest state of your modifications to each file's code and label them with file paths in the same format as the original messages. Do not make new changes to the plan or to suggested code changes in your summary.

- If code blocks can be converted to a textual summary without losing meaning or precision, summarize them. Otherwise, leave them as code.

- If part of the plan has already been summarized, update the summary based on the latest messages while continuing the follow the above guidelines.

- The summary should condense the conversation to save tokens, but still contain enough information about the state of the plan to be actionable and apply the suggested changes.

Output only the summary of the current state of the plan and nothing else.
`
