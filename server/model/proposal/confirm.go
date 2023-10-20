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

const systemPrompt = `
[YOUR INSTRUCTIONS]

You apply changes from a plan to a given code file. You can either us the 'writeEntireFile' function to write the entire file, or the 'writeReplacements' function to write a list of replacements to apply to the file. Decide which is a more efficient way to apply the changes, and call the appropriate function.

If the current state of the file within the plan is included, apply your changes to the current state of the file.

A. If you are using the 'writeEntireFile' function, call it with the full content of the file as raw text, including any updates from the previous response. Ouput the entire updated file.
	
B. If you are using the 'writeReplacements' function, call it with a list of replacements to apply to the file. Each replacement is an object with two properties: 'old' and 'new'. 'old' is the old text to replace, and 'new' is the new text to replace it with. You can include as many replacements as you want. You must include at least one replacement.
- The 'new' text must include the full text of the replacement without any placeholders or references to the original file.
- The 'old' text must be a substring of the current file.
- The 'old' text must not overlap with any other 'old' text in the list of replacements.

Replacement examples below. Note: >>> and <<< indicate the start and end of an example response.

1.)
If the current file is:
` + "```" + `
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
` + "```" + `

And the previous response was:

>>>
You can change the main.go file to print the current time instead of "Hello, world!".:

- main.go:
` + "```" + `
func main() {
	fmt.Println(time.Now())
}
` + "```" + `

You'll also need to import the time package:

- main.go:
` + "```" + `
import (
	"fmt"
	"time"
)
` + "```" + `
<<<

Then you would call the 'writeReplacements' function like this:

writeReplacements({
	replacements: [
		{
			old: "import \"fmt\"",
			new: "import (\n\t\"fmt\"\n\t\"time\"\n)"
		},
		{
			old: "fmt.Println(\"Hello, world!\")",
			new: "fmt.Println(time.Now())"
		}
	}
})

2.)
If the current file is:
` + "```" + `
package helpers

func Add(a, b int) int {
	return a + b
}
` + "```" + `

And the previous response was:

>>>
Add another function to the helpers.go file that subtracts two numbers:

- helpers.go:
` + "```" + `
func Subtract(a, b int) int {
	return a - b
}
` + "```" + `
<<<

Then you would call the 'writeReplacements' function like this:

writeReplacements({
	replacements: [
		{
			old: "\n}",
			new: "\n}\n\nfunc Subtract(a, b int) int {\n\treturn a - b\n}"
		}
	]
})

3.)
If the current file is:
` + "```" + `
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
` + "```" + `

And the previous response was:

>>>
You can change the main.go file to print "I love you!" in addition to "Hello, world!".:

- main.go:
` + "```" + `
func main() {
	fmt.Println("Hello, world!")
	fmt.Println("I love you!")
}
` + "```" + `						
<<<

Then you would call the 'writeReplacements' function like this:

writeReplacements({
	replacements: [
		{
			old: "fmt.Println(\"Hello, world!\")",
			new: "fmt.Println(\"Hello, world!\")\n\tfmt.Println(\"I love you!\")"
		}
	]
})

[END INSTRUCTIONS]`

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

			fmtStr := "\nCurrent state of %s:\n```\n%s\n```"
			fmtArgs := []interface{}{filePath}

			currentState := proposal.Request.CurrentPlan.Files[filePath]
			if currentState != "" {
				fmtArgs = append(fmtArgs, currentState)
			} else if fileContext != nil {
				fmtArgs = append(fmtArgs, fileContext.Body)
			}

			fileMessages := []openai.ChatCompletionMessage{}
			if fileContext != nil || currentState != "" {
				fileMessages = append(fileMessages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt + "\n\n" + fmt.Sprintf(fmtStr, fmtArgs...),
				})
			}

			fileMessages = append(fileMessages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: proposal.Content,
			},
				openai.ChatCompletionMessage{
					Role: openai.ChatMessageRoleUser,
					Content: fmt.Sprintf(`
					Based on your instructions, apply the changes from the plan to %s. You must call either the 'writeReplacements' function or the 'writeEntireFile' function, depending on which is a more efficient way to apply the changes. Don't call any other function.
					`, filePath),
				})

			fmt.Println("Calling model for file: " + filePath)
			for _, msg := range fileMessages {
				fmt.Printf("%s: %s\n", msg.Role, msg.Content)
			}

			modelReq := openai.ChatCompletionRequest{
				Model: openai.GPT4,
				Functions: []openai.FunctionDefinition{{
					Name: "writeEntireFile",
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
				}, {
					Name: "writeReplacements",
					Parameters: &jsonschema.Definition{
						Type: jsonschema.Object,
						Properties: map[string]jsonschema.Definition{
							"replacements": {
								Type:        jsonschema.Array,
								Description: "A list of replacements to apply to the file",
								Items: &jsonschema.Definition{
									Type: jsonschema.Object,
									Properties: map[string]jsonschema.Definition{
										"old": {
											Type:        jsonschema.String,
											Description: "The old text to replace",
										},
										"new": {
											Type:        jsonschema.String,
											Description: "The new text to replace it with",
										},
									},
									Required: []string{"old", "new"},
								},
							},
						},
						Required: []string{"replacements"},
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
								onError(fmt.Errorf("Stream finished without a function call. Reason: %s", choice.FinishReason))
								return
							}
						}

						var content string
						delta := response.Choices[0].Delta

						if delta.FunctionCall == nil {
							fmt.Printf("\nStream received data not for a function call")
							continue
						} else {
							content = delta.FunctionCall.Arguments
						}

						plans.Update(proposalId, func(p *types.Plan) {
							p.Files[filePath] += content
						})

						chunk := &shared.PlanChunk{
							Path:    filePath,
							Content: content,
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
