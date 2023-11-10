package proposal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/types"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func buildPlan(proposalId string, fileContents map[string]string, numTokensByFile map[string]int, onStream types.OnStreamFunc) error {
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
			// get relevant file context (if any)
			var fileContext *shared.ModelContextPart
			for _, part := range proposal.Request.ModelContext {
				if part.FilePath == filePath {
					fileContext = part
					break
				}
			}

			_, fileInCurrentPlan := proposal.Request.CurrentPlan.Files[filePath]

			if fileContext == nil && !fileInCurrentPlan {
				// new file
				streamed := shared.StreamedFile{
					Content: fileContents[filePath],
				}
				bytes, err := json.Marshal(streamed)

				if err != nil {
					onError(fmt.Errorf("error marshalling streamed file: %v", err))
					return
				}

				chunk := &shared.PlanChunk{
					Path:      filePath,
					Content:   string(bytes),
					NumTokens: numTokensByFile[filePath],
				}

				// fmt.Printf("%s: %s", filePath, content)
				chunkJson, err := json.Marshal(chunk)
				if err != nil {
					onError(fmt.Errorf("error marshalling plan chunk: %v", err))
					return
				}
				onStream(string(chunkJson), nil)

				finished := false
				plans.Update(proposalId, func(plan *types.Plan) {
					plan.FilesFinished[filePath] = true
					plan.Files[filePath] = string(bytes)

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

				return
			}

			fmt.Println("Getting file from model: " + filePath)
			fmt.Println("File context:", fileContext)

			fmtStr := "\nCurrent state of %s:\n```\n%s\n```"
			fmtArgs := []interface{}{filePath}

			currentState := proposal.Request.CurrentPlan.Files[filePath]
			if currentState != "" {
				fmtArgs = append(fmtArgs, currentState)
			} else if fileContext != nil {
				fmtArgs = append(fmtArgs, fileContext.Body)
			}

			fileMessages := []openai.ChatCompletionMessage{}

			var msg string
			if fileContext != nil {
				msg = prompts.SysReplace
			} else {
				msg = prompts.SysWriteFile
			}

			if fileContext != nil || currentState != "" {
				fmt.Println("Adding current state to message")
				msg += "\n\n" + fmt.Sprintf(fmtStr, fmtArgs...)

				fmt.Println("Message:")
				fmt.Println(msg)
			}

			fileMessages = append(fileMessages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: msg,
			}, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: proposal.Request.Prompt,
			}, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: proposal.Content,
			})

			if fileContext != nil {
				fileMessages = append(fileMessages,
					openai.ChatCompletionMessage{
						Role: openai.ChatMessageRoleUser,
						Content: fmt.Sprintf(`
					Based on your instructions, apply the changes from the plan to %s. You must call either the 'writeReplacements' function or the 'writeEntireFile' function, depending on which will apply the changes with fewer tokens.
					`, filePath),
					})
			} else {
				fileMessages = append(fileMessages,
					openai.ChatCompletionMessage{
						Role: openai.ChatMessageRoleUser,
						Content: fmt.Sprintf(`
						Based on your instructions, apply the changes from the plan to %s. You must call the 'writeEntireFile' function with the ENTIRE FILE, including your suggested changes. Don't call any other function.
						`, filePath),
					})
			}

			fmt.Println("Calling model for file: " + filePath)

			// for _, msg := range fileMessages {
			// 	fmt.Printf("%s: %s\n", msg.Role, msg.Content)
			// }

			functions := []openai.FunctionDefinition{prompts.WriteFileFn}

			if fileContext != nil {
				functions = append(functions, prompts.ReplaceFn)
			}

			modelReq := openai.ChatCompletionRequest{
				Model:       model.BuilderModel,
				Functions:   functions,
				Messages:    fileMessages,
				Temperature: 0.0,
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
							onError(fmt.Errorf("stream error: %v", err))
							return
						}

						if len(response.Choices) == 0 {
							onError(fmt.Errorf("stream error: no choices"))
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
								onError(fmt.Errorf("stream finished without a function call. Reason: %s", choice.FinishReason))
								return
							}
						}

						var content string
						delta := response.Choices[0].Delta

						if delta.FunctionCall == nil {
							spew.Dump(delta)
							continue
						} else {
							content = delta.FunctionCall.Arguments
						}

						plans.Update(proposalId, func(p *types.Plan) {
							p.Files[filePath] += content
						})

						chunk := &shared.PlanChunk{
							Path:      filePath,
							Content:   content,
							NumTokens: 1,
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
