package proposal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"plandex-server/model"
	"plandex-server/types"
	"time"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func confirmProposal(proposalId string, onStream types.OnStreamFunc) error {
	goEnv := os.Getenv("GOENV")
	if goEnv == "test" {
		streamFilesLoremIpsum(onStream)
		return nil
	}

	proposal := proposals.Get(proposalId)
	if proposal == nil {
		return errors.New("proposal not found")
	}

	if !proposal.IsFinished() {
		return errors.New("proposal not finished")
	}

	ctx, cancel := context.WithCancel(context.Background())

	plans.Set(proposalId, &types.Plan{
		ProposalId:    proposalId,
		NumFiles:      len(proposal.PlanDescription.Files),
		Files:         map[string]string{},
		FileErrs:      map[string]error{},
		FilesFinished: map[string]bool{},
		ProposalStage: types.ProposalStage{
			CancelFn: &cancel,
		},
	})

	for _, filePath := range proposal.PlanDescription.Files {
		onError := func(err error) {
			fmt.Printf("Error for file %s: %v\n", filePath, err)
			plans.Update(proposalId, func(p *types.Plan) {
				p.FileErrs[filePath] = err
				p.SetErr(err)
			})
			onStream("", err)
		}

		go func(filePath string) {
			fmt.Println("Getting file from model: " + filePath)

			// get relevant file context (if any)
			var fileContext *shared.ModelContextPart
			for _, part := range proposal.Request.ModelContext {
				if part.FilePath == filePath {
					fileContext = &part
					break
				}
			}

			fmtStr := ""
			fmtArgs := []interface{}{}

			if fileContext != nil {
				fmtStr += "Original file %s:\n```\n%s\n```"
				fmtArgs = []interface{}{filePath, fileContext.Body}
			}

			currentState := proposal.Request.CurrentPlan.Files[filePath]
			if currentState != "" {
				fmtStr += "\nCurrent state of file %s in the plan:\n```\n%s\n```"
				fmtArgs = append(fmtArgs, filePath, currentState)
			}

			fileMessages := []openai.ChatCompletionMessage{}
			if fileContext != nil || currentState != "" {
				fileMessages = append(fileMessages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleSystem,
					Content: fmt.Sprintf(fmtStr, fmtArgs...),
				})
			}

			fileMessages = append(fileMessages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: proposal.Content,
			},
				openai.ChatCompletionMessage{
					Role: openai.ChatMessageRoleUser,
					Content: fmt.Sprintf(`
						Based on your previous response, call the 'writeFile' function with the full content of the file %s as raw text, including any updates. If the current state of the file within the plan is included above, apply your changes to the *current file*, not the original file. If there is no current file, apply your changes to the original file. You must include the entire file and not leave anything out, even if it is already present the source file. Do not include any placeholders or references to the original file. Output the updated entire file. Only call the 'writeFile' function in your reponse. Don't call any other function.
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
						onError(fmt.Errorf("stream timeout due to inactivity"))
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

						if err != nil {
							onError(fmt.Errorf("Stream error: %v", err))
							return
						}

						if len(response.Choices) == 0 {
							onError(fmt.Errorf("Stream error: no choices"))
							return
						}

						choice := response.Choices[0]

						if choice.FinishReason != "" {
							if choice.FinishReason == openai.FinishReasonFunctionCall {
								finished := false
								plans.Update(proposalId, func(plan *types.Plan) {
									plan.FilesFinished[filePath] = true

									if plan.DidFinish() {
										plan.Finish()
										finished = true
									}
								})

								if finished {
									fmt.Println("Stream finished")
									onStream(shared.STREAM_FINISHED, nil)
									return
								}

							} else {
								onError(fmt.Errorf("Stream finished without 'writeFile' function call. Reason: %s", choice.FinishReason))
								return
							}

							return
						}

						var content string
						delta := response.Choices[0].Delta

						if delta.FunctionCall == nil {
							fmt.Printf("\nStream received data not for 'writeFile' function call")
							continue
						} else {
							content = delta.FunctionCall.Arguments
						}

						plans.Update(proposalId, func(p *types.Plan) {
							p.Files[filePath] += content
						})

						chunk := &shared.PlanChunk{
							FilePath: filePath,
							Content:  content,
						}

						// fmt.Printf("%s: %s", filePath, content)
						chunkJson, err := json.Marshal(chunk)
						if err != nil {
							onError(fmt.Errorf("error marshalling plan chunk: %v", err))
							return
						}
						onStream(string(chunkJson), nil)
					}
				}
			}()
		}(filePath)
	}

	return nil
}
