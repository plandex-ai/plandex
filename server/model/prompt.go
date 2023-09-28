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

type fileResult struct {
	filePath string
	content  string
	isExec   bool
}

type firstParseResponse struct {
	Reply     string   `json:"reply"`
	CommitMsg string   `json:"commitMsg"`
	Files     []string `json:"files"`
}

type parseFileResponse struct {
	Content string `json:"content"`
}

func Prompt(req shared.PromptRequest) ([]byte, error) {
	contextText := formatModelContext(req.ModelContext)

	systemMessageText := `
		You are Plandex, an AI programming and system administration assistant.
		You help programmers with tasks, especially those that involve multiple files and shell commands. You offer a structured, versioned, and iterative approach to AI-driven development. 
		You and the programmer collaborate to create a 'plan' for the task at hand. A plan is a set of files and an 'exec' script with an attached context.
		Based on user-provided context, please create a plan for the task. When suggesting changes that would modify files from the context or create new files, always precede them with the file path.
		Context from the user:` + contextText + `
		Current state of the plan:` + formatCurrentPlan(req.CurrentPlan)

	systemMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemMessageText,
	}

	messages := []openai.ChatCompletionMessage{
		systemMessage,
	}

	if len(req.Conversation) > 0 {
		messages = append(messages, req.Conversation...)
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: req.Prompt,
	})

	for _, message := range messages {
		fmt.Printf("%s: %s\n", message.Role, message.Content)
	}

	initialResp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT4,
			Messages: messages,
		},
	)

	if err != nil {
		fmt.Printf("Error while creating chat completion: %v\n", err)
		return nil, err
	}

	initialStrRes := initialResp.Choices[0].Message.Content
	fmt.Println("Initial response from model:")
	fmt.Println(initialStrRes)

	if req.ChatOnly {
		chatResp := shared.PromptResponse{
			Reply: initialStrRes,
		}
		chatByteRes, err := json.Marshal(chatResp)
		if err != nil {
			fmt.Printf("Error marshalling chat response: %v\n", err)
			return nil, err
		}
		return chatByteRes, nil
	}

	firstParseResp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Functions: []openai.FunctionDefinition{{
				Name: "parse",
				Parameters: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"reply": {
							Type:        jsonschema.String,
							Description: "Your previous response without code or commands.",
						},
						"commitMsg": {
							Type:        jsonschema.String,
							Description: "A brief commit message for the changes proposed in your previous response.",
						},
						"files": {
							Type:        jsonschema.Array,
							Description: "An array of file paths from your previous response that should be created or updated.",
							Items: &jsonschema.Definition{
								Type: jsonschema.String,
							},
						},
					},
					Required: []string{"reply", "commitMsg", "files"},
				},
			}},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an AI parser...",
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: initialStrRes,
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("Error during first parse completion: %v\n", err)
		return nil, err
	}

	var firstParseStrRes string
	for _, choice := range firstParseResp.Choices {
		if choice.FinishReason == "function_call" && choice.Message.FunctionCall != nil && choice.Message.FunctionCall.Name == "parse" {
			fnCall := choice.Message.FunctionCall
			firstParseStrRes = fnCall.Arguments
		}
	}

	if firstParseStrRes == "" {
		return nil, fmt.Errorf("no parse function call found in response")
	}

	firstParseByteRes := []byte(firstParseStrRes)
	var firstParsePromptResp firstParseResponse
	err = json.Unmarshal(firstParseByteRes, &firstParsePromptResp)
	if err != nil {
		fmt.Printf("Error unmarshalling first parse response: %v\n", err)
		fmt.Printf("JSON causing the error: %s\n", firstParseByteRes)
		return nil, err
	}

	finalResp := shared.PromptResponse{
		Reply:     firstParsePromptResp.Reply,
		CommitMsg: firstParsePromptResp.CommitMsg,
		Files:     map[string]string{},
	}

	// Channel to receive results from goroutines
	results := make(chan fileResult, len(firstParsePromptResp.Files))

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Make sure all resources are cleaned up

	// Error channel to catch the first occurring error
	errChan := make(chan error, 1)

	// Channel to signal completion of goroutines
	done := make(chan bool, len(firstParsePromptResp.Files))

	for _, filePath := range firstParsePromptResp.Files {
		go func(filePath string) {
			// Listen for context cancellation
			select {
			case <-ctx.Done():
				done <- true
				return
			default:
			}

			fmt.Println("Getting file from model: " + filePath)

			// get relevant file context (if any)
			var fileContext *shared.ModelContextPart
			for _, part := range req.ModelContext {
				if part.FilePath == filePath {
					fileContext = &part
					break
				}
			}

			fileMessages := []openai.ChatCompletionMessage{}
			if fileContext != nil {
				fileMessages = append(fileMessages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleSystem,
					Content: fmt.Sprintf("Original file %s:\n```\n%s\n```", filePath, fileContext.Body),
				})
			}

			fileMessages = append(fileMessages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: initialStrRes,
			},
				openai.ChatCompletionMessage{
					Role: openai.ChatMessageRoleUser,
					Content: fmt.Sprintf(`
						Based on your previous response, call the 'writeFile' function with the full content of the file %s as raw text, including any updates. You must include the entire file and not leave anything out, even if it is already present in the original file. Do not include any placeholders or references to the original file. Output the updated entire file. Only call the 'writeFile' function in your reponse. Don't call any other function.
							`, filePath),
				})

			fileResp, err := client.CreateChatCompletion(
				ctx,
				openai.ChatCompletionRequest{
					Model: openai.GPT4,
					// Temperature: 0,
					Functions: []openai.FunctionDefinition{{
						Name: "writeFile",
						Parameters: &jsonschema.Definition{
							Type: jsonschema.Object,
							Properties: map[string]jsonschema.Definition{
								"content": {
									Type:        jsonschema.String,
									Description: "The full content of the file, including any updates from the previous response, as raw text",
								},
							},
							Required: []string{"content"},
						},
					}},
					Messages: fileMessages,
				},
			)

			if err != nil {
				fmt.Println("Error getting file from model: " + filePath + ": " + err.Error())

				// Send error to error channel and cancel the context
				select {
				case errChan <- err:
				default:
				}
				cancel()
			}

			var fileResStr string
			for _, choice := range fileResp.Choices {
				if choice.FinishReason == "function_call" && choice.Message.FunctionCall != nil && choice.Message.FunctionCall.Name == "writeFile" {
					fnCall := choice.Message.FunctionCall
					fileResStr = fnCall.Arguments
					break
				}
			}

			if fileResStr == "" {
				fmt.Println("No response from model for file " + filePath)
			} else {
				var fileRes parseFileResponse
				if err := json.Unmarshal([]byte(fileResStr), &fileRes); err != nil {
					// Send error to error channel and cancel the context
					select {
					case errChan <- err:
					default:
					}
					cancel()
				}

				fmt.Println("File response from model for file " + filePath + ":")
				fmt.Println(fileRes.Content)

				// Send result to channel
				results <- fileResult{
					filePath: filePath,
					content:  fileRes.Content,
				}
			}

			done <- true

		}(filePath)
	}

	go func() {
		// Listen for context cancellation
		select {
		case <-ctx.Done():
			done <- true
			return
		default:
		}

		fmt.Println("Getting exec.sh from model: ")

		execResp, err := client.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model: openai.GPT4,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleAssistant,
						Content: initialStrRes,
					},
					{
						Role: openai.ChatMessageRoleUser,
						Content: fmt.Sprintf(`
								Output a shell script that includes any commands from the previous message that should be executed after any relevant files are created and/or updated. 
								
								If no commands should be executed, output only 'no exec commands'. Otherwise, output the script as raw text and nothing else.
							`),
					},
				},
			},
		)

		if err != nil {
			fmt.Println("Error getting shell script from model: " + err.Error())

			// Send error to error channel and cancel the context
			select {
			case errChan <- err:
			default:
			}
			cancel()
		}

		if err == nil && len(execResp.Choices) > 0 {
			content := execResp.Choices[0].Message.Content

			fmt.Println("Exec res from model:")
			fmt.Println(content)

			// Check if content includes phrase 'No commands'
			if strings.Contains(strings.ToLower(content), "no exec commands") {
				fmt.Println("No commands to execute")
				done <- true
				return
			}

			// Send result to channel
			results <- fileResult{
				content: content,
				isExec:  true,
			}
		} else {
			fmt.Println("No response from model for shell script")
		}

		done <- true
	}()

	// Wait for all goroutines to finish or an error to occur
	for i := 0; i < len(firstParsePromptResp.Files)+1; i++ {
		select {
		case <-done:
		case err := <-errChan:
			return nil, err
		}
	}
	close(results) // Close the results channel after all routines are done

	// Process the results
	for result := range results {
		fmt.Println("File response from model for file " + result.filePath + ":")
		fmt.Println(result.content)

		content := strings.TrimPrefix(result.content, "```")
		content = strings.TrimSuffix(content, "```")

		if result.isExec {
			finalResp.Exec = content
		} else {
			finalResp.Files[result.filePath] = content
		}
	}

	// convert final response to JSON
	finalByteRes, err := json.Marshal(finalResp)

	if err != nil {
		return nil, err
	}

	return finalByteRes, nil
}
