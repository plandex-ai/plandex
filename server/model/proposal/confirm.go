package proposal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"plandex-server/model"
	"plandex-server/types"
	"strings"
	"sync"
	"time"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type firstParseResponse struct {
	Summary   string   `json:"summary"`
	CommitMsg string   `json:"commitMsg"`
	Files     []string `json:"files"`
}

type parseFileResponse struct {
	Content string `json:"content"`
}

func ConfirmProposal(proposalId string, onStream types.OnStreamPlanFunc) (*context.CancelFunc, error) {
	mu.Lock()
	proposal, ok := proposalsMap[proposalId]
	mu.Unlock()
	if !ok {
		return nil, errors.New("proposal not found")
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	var proposalContent string
	mu.Lock()
	proposal.Cancel = &cancel
	proposalsMap[proposalId] = proposal
	proposalContent = proposal.Content
	mu.Unlock()

	setError := func(err error) {
		mu.Lock()
		proposal := proposalsMap[proposalId]
		proposal.ProposalError = err
		proposalsMap[proposalId] = proposal
		mu.Unlock()
	}

	firstParseResp, err := model.Client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Functions: []openai.FunctionDefinition{{
				Name: "parse",
				Parameters: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"summary": {
							Type:        jsonschema.String,
							Description: "High-level summary of changed proposed in previous response without code or commands.",
						},
						"commitMsg": {
							Type:        jsonschema.String,
							Description: "A brief commit message for the changes proposed in AI's plan from previous response.",
						},
						"files": {
							Type:        jsonschema.Array,
							Description: "An array of file paths in AI's plan from previous response that should be created or updated.",
							Items: &jsonschema.Definition{
								Type: jsonschema.String,
							},
						},
					},
					Required: []string{"summary", "commitMsg", "files"},
				},
			}},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an AI parser. You turn an AI's plan for a programming task into a high-level summary, a set of files and an exec script. You call the 'parse' function with these arguments: summary, commitMsg, files. Only call the 'parse' function in your response. Don't call any other function.",
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: proposalContent,
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("Error during first parse completion: %v\n", err)
		setError(err)
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
		err = fmt.Errorf("no parse function call found in response")
		setError(err)
		return nil, err
	}

	firstParseByteRes := []byte(firstParseStrRes)
	var firstParsePromptResp firstParseResponse
	err = json.Unmarshal(firstParseByteRes, &firstParsePromptResp)
	if err != nil {
		fmt.Printf("Error unmarshalling first parse response: %v\n", err)
		fmt.Printf("JSON causing the error: %s\n", firstParseByteRes)
		setError(err)
		return nil, err
	}

	for _, filePath := range firstParsePromptResp.Files {
		wg.Add(1)

		onError := func(err error) {
			setError(err)
			onStream(&shared.PlanChunk{FilePath: filePath}, true, err)
		}

		go func(filePath string) {
			defer wg.Done()

			fmt.Println("Getting file from model: " + filePath)

			// get relevant file context (if any)
			var fileContext *shared.ModelContextPart
			for _, part := range *proposal.ModelContext {
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
				Content: proposalContent,
			},
				openai.ChatCompletionMessage{
					Role: openai.ChatMessageRoleUser,
					Content: fmt.Sprintf(`
						Based on your previous response, call the 'writeFile' function with the full content of the file %s as raw text, including any updates. You must include the entire file and not leave anything out, even if it is already present in the original file. Do not include any placeholders or references to the original file. Output the updated entire file. Only call the 'writeFile' function in your reponse. Don't call any other function.
							`, filePath),
				})

			modelReq := openai.ChatCompletionRequest{
				Model: openai.GPT4,
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
			}

			stream, err := model.Client.CreateChatCompletionStream(ctx, modelReq)
			if err != nil {
				fmt.Printf("Error creating plan file stream for path %s: %v\n", filePath, err)
				onError(err)
				return
			}

			go func() {
				defer stream.Close()

				// Create a timer that will trigger if no chunk is received within the specified duration
				timer := time.NewTimer(model.OPENAI_STREAM_CHUNK_TIMEOUT)
				defer timer.Stop()

				for {
					select {
					case <-ctx.Done():
						// The main context was canceled (not the timer)
						return
					case <-timer.C:
						// Timer triggered because no new chunk was received in time
						fmt.Println("\nStream timeout due to inactivity")
						err = fmt.Errorf("stream timeout due to inactivity")
						onError(err)
						return
					default:
						response, err := stream.Recv()

						if err == nil {
							// Successfully received a chunk, reset the timer
							if !timer.Stop() {
								<-timer.C
							}
							timer.Reset(model.OPENAI_STREAM_CHUNK_TIMEOUT)
						}

						if errors.Is(err, io.EOF) {
							fmt.Println("\nStream finished")
							return
						}

						if err != nil {
							fmt.Printf("\nStream error: %v\n", err)
							onError(err)
							return
						}

						if len(response.Choices) == 0 {
							fmt.Println("\nStream finished")
							return
						}

						// TODO handle different finish reasons
						if response.Choices[0].FinishReason != "" {
							fmt.Println("\nStream finished")
							return
						}

						var content string
						delta := response.Choices[0].Delta
						if delta.FunctionCall != nil && delta.FunctionCall.Name == "writeFile" {
							content = delta.FunctionCall.Arguments
						} else {

						}

						chunk := &shared.PlanChunk{
							FilePath: filePath,
							Content:  content,
							IsExec:   false,
						}

						fmt.Printf("%s: %s", filePath, content)
						onStream(chunk, false, nil)
					}
				}
			}()
		}(filePath)
	}

	wg.Add(1)
	onExecErr := func(err error) {
		setError(err)
		onStream(&shared.PlanChunk{IsExec: true}, true, err)
	}
	go func() {
		defer wg.Done()
		fmt.Println("Getting exec.sh from model: ")

		// Define the model request for exec.sh
		modelReq := openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Functions: []openai.FunctionDefinition{{
				Name: "writeExec",
				Parameters: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"content": {
							Type:        jsonschema.String,
							Description: "The shell script from the previous message, including any updates",
						},
					},
					Required: []string{"content"},
				},
			}},
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are an AI parser. You turn an AI's plan for a programming task into executable shell commands. Call the 'writeExec' function with the entire shell script as raw text, based on your previous response. Only call the 'writeExec' function in your response. Don't call any other function.",
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: proposalContent,
				},
				{
					Role: openai.ChatMessageRoleUser,
					Content: fmt.Sprintf(`
                    Based on your previous response, call the 'writeExec' function with the shell script that includes any commands that should be executed after any relevant files are created and/or updated. If no commands should be executed, pass only 'no exec commands' to the 'writeExec' function. Otherwise, pass the full script as raw text to the function and output nothing else.
                `),
				},
			},
		}

		stream, err := model.Client.CreateChatCompletionStream(ctx, modelReq)
		if err != nil {
			fmt.Println("Error creating shell script stream: " + err.Error())
			onExecErr(err)
			return
		}

		defer stream.Close()

		// Create a timer that will trigger if no chunk is received within the specified duration
		timer := time.NewTimer(model.OPENAI_STREAM_CHUNK_TIMEOUT)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				// The main context was canceled (not the timer)
				return
			case <-timer.C:
				// Timer triggered because no new chunk was received in time
				fmt.Println("\nStream timeout due to inactivity")
				err = fmt.Errorf("stream timeout due to inactivity")
				onExecErr(err)
				return
			default:
				response, err := stream.Recv()

				if err == nil {
					// Successfully received a chunk, reset the timer
					if !timer.Stop() {
						<-timer.C
					}
					timer.Reset(model.OPENAI_STREAM_CHUNK_TIMEOUT)
				}

				if errors.Is(err, io.EOF) {
					fmt.Println("\nStream finished for exec.sh")
					return
				}

				if err != nil {
					fmt.Printf("\nStream error for exec.sh: %v\n", err)
					onExecErr(err)
					return
				}

				// TODO handle different finish reasons
				if response.Choices[0].FinishReason != "" {
					fmt.Println("\nStream finished")
					return
				}

				if len(response.Choices) == 0 {
					fmt.Println("\nStream finished for exec.sh")
					return
				}

				var content string
				delta := response.Choices[0].Delta
				if delta.FunctionCall != nil && delta.FunctionCall.Name == "writeExec" {
					content = delta.FunctionCall.Arguments
				}

				fmt.Printf("exec.sh content: %s", content)

				// Check if content includes phrase 'No commands'
				if strings.Contains(strings.ToLower(content), "no exec commands") {
					fmt.Println("No commands to execute for exec.sh")
				} else {
					fmt.Println("Commands to execute for exec.sh")
					onStream(&shared.PlanChunk{
						IsExec:  true,
						Content: content,
					}, false, nil)
				}
			}
		}
	}()

	// Wait for all streams to finish and then inform the caller
	go func() {
		wg.Wait()
		mu.Lock()
		delete(proposalsMap, proposalId)
		mu.Unlock()
		onStream(nil, true, nil)
	}()

	return &cancel, nil

}
