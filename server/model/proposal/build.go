package proposal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/types"
	"sort"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

const MAX_RETRIES = 5

type buildPlanParams struct {
	ProposalId      string
	FileContents    map[string]string
	NumTokensByFile map[string]int
	OnStream        types.OnStreamFunc
}

func buildPlan(params buildPlanParams) error {
	proposalId := params.ProposalId
	fileContents := params.FileContents
	numTokensByFile := params.NumTokensByFile
	onStream := params.OnStream

	// goEnv := os.Getenv("GOENV")
	// if goEnv == "test" {
	// 	streamFilesLoremIpsum(onStream)
	// 	return nil
	// }

	proposal := proposals.Get(proposalId)
	if proposal == nil {
		return errors.New("proposal not found")
	}

	if !proposal.IsFinished() {
		return errors.New("proposal not finished")
	}

	ctx, cancel := context.WithCancel(context.Background())

	plans.Set(proposalId, &types.Plan{
		ProposalId: proposalId,
		NumFiles:   len(proposal.PlanDescription.Files),
		Buffers:    map[string]string{},
		Files:      map[string]*shared.PlanFile{},
		FileErrs:   map[string]error{},
		ProposalStage: types.ProposalStage{
			CancelFn: &cancel,
		},
	})

	onFinishBuild := func() {
		fmt.Println("Build finished. Starting write phase.")

		onStream(shared.STREAM_WRITE_PHASE, nil)

		plan := plans.Get(proposalId)

		for _, planFile := range plan.Files {
			// fmt.Println()
			// spew.Dump(planFile)

			bytes, err := json.Marshal(planFile)
			if err != nil {
				onStream("", err)
				return
			}

			onStream(string(bytes), nil)
		}

		fmt.Println("Stream finished")
		onStream(shared.STREAM_FINISHED, nil)
	}

	onFinishBuildFile := func(filePath string, planFile *shared.PlanFile) {
		finished := false

		// fmt.Println("onFinishBuildFile: " + filePath)
		// spew.Dump(planFile)

		plans.Update(proposalId, func(plan *types.Plan) {
			plan.Files[filePath] = planFile

			if plan.DidFinish() {
				plan.Finish()
				finished = true
			}
		})

		fmt.Printf("Finished building file %s\n", filePath)

		if finished {
			onFinishBuild()
		}
	}

	onBuildFileError := func(filePath string, err error) {
		fmt.Printf("Error for file %s: %v\n", filePath, err)
		plans.Update(proposalId, func(p *types.Plan) {
			p.FileErrs[filePath] = err
			p.SetErr(err)
		})
		onStream("", err)
	}

	var buildFile func(filePath string, numRetry int, failedReplacements map[int]*shared.Replacement)
	buildFile = func(filePath string, numRetry int, failedReplacements map[int]*shared.Replacement) {
		fmt.Printf("Building file %s, numRetry: %d\n", filePath, numRetry)

		// get relevant file context (if any)
		var fileContext *shared.ModelContextPart
		for _, part := range proposal.Request.ModelContext {
			if part.FilePath == filePath {
				fileContext = part
				break
			}
		}

		var currentState string
		currentPlanFile, fileInCurrentPlan := proposal.Request.CurrentPlan.Files[filePath]

		if fileInCurrentPlan {
			currentState = currentPlanFile
		} else if fileContext != nil {
			currentState = fileContext.Body
		}

		if currentState == "" {
			fmt.Printf("File %s not found in model context or current plan. Creating new file.\n", filePath)

			planTokenCount := &shared.PlanTokenCount{
				Path:      filePath,
				NumTokens: numTokensByFile[filePath],
				Finished:  true,
			}

			// fmt.Printf("%s: %s", filePath, content)
			planTokenCountJson, err := json.Marshal(planTokenCount)
			if err != nil {
				onBuildFileError(filePath, fmt.Errorf("error marshalling plan chunk: %v", err))
				return
			}
			onStream(string(planTokenCountJson), nil)

			// new file
			planFile := &shared.PlanFile{
				Path:    filePath,
				Content: fileContents[filePath],
			}
			onFinishBuildFile(filePath, planFile)
			return
		}

		fmt.Println("Getting file from model: " + filePath)
		// fmt.Println("File context:", fileContext)

		msg := prompts.SysReplace

		if currentState != "" {
			fmt.Println("Adding current state to message")
			msg += "\n\n" + fmt.Sprintf("\nCurrent state of %s:\n```\n%s\n```", filePath, currentState)
			// fmt.Println("Message:")
			// fmt.Println(msg)
		}

		fileMessages := []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: msg,
			}, {
				Role:    openai.ChatMessageRoleUser,
				Content: proposal.Request.Prompt,
			}, {
				Role:    openai.ChatMessageRoleAssistant,
				Content: proposal.Content,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompts.GetReplacePrompt(filePath),
			},
		}

		if numRetry > 0 && failedReplacements != nil {
			bytes, err := json.Marshal(failedReplacements)
			if err != nil {
				onBuildFileError(filePath, fmt.Errorf("error marshalling failed replacements: %v", err))
				return
			}

			fileMessages = append(fileMessages,
				openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: string(bytes),
				},

				openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: prompts.GetCorrectReplacementPrompt(failedReplacements),
				})
		}

		fmt.Println("Calling model for file: " + filePath)

		// for _, msg := range fileMessages {
		// 	fmt.Printf("%s: %s\n", msg.Role, msg.Content)
		// }

		modelReq := openai.ChatCompletionRequest{
			Model:          model.BuilderModel,
			Functions:      []openai.FunctionDefinition{prompts.ReplaceFn},
			Messages:       fileMessages,
			Temperature:    0.0,
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
		}

		stream, err := model.Client.CreateChatCompletionStream(ctx, modelReq)
		if err != nil {
			fmt.Printf("Error creating plan file stream for path '%s': %v\n", filePath, err)

			if numRetry >= MAX_RETRIES {
				onBuildFileError(filePath, fmt.Errorf("failed to create plan file stream for path '%s' after %d retries: %v", filePath, numRetry, err))
			} else {
				fmt.Println("Retrying build plan for file: " + filePath)
				buildFile(filePath, numRetry+1, failedReplacements)
				if err != nil {
					onBuildFileError(filePath, fmt.Errorf("failed to retry build plan for file '%s': %v", filePath, err))
				}
			}
			return
		}

		go func() {
			defer stream.Close()

			// Create a timer that will trigger if no chunk is received within the specified duration
			timer := time.NewTimer(model.OPENAI_STREAM_CHUNK_TIMEOUT)
			defer timer.Stop()

			handleErrorRetry := func(maxRetryErr error, shouldSleep bool, failed map[int]*shared.Replacement) {
				fmt.Printf("Error for file %s: %v\n", filePath, maxRetryErr)

				if numRetry >= MAX_RETRIES {
					onBuildFileError(filePath, maxRetryErr)
				} else {
					if shouldSleep {
						time.Sleep(1 * time.Second * time.Duration(math.Pow(float64(numRetry+1), 2)))
					}
					buildFile(filePath, numRetry+1, failed)
					if err != nil {
						onBuildFileError(filePath, fmt.Errorf("failed to retry build plan for file '%s': %v", filePath, err))
					}
				}

			}

			for {
				select {
				case <-ctx.Done():
					// The main context was canceled (not the timer)
					return
				case <-timer.C:
					// Timer triggered because no new chunk was received in time
					handleErrorRetry(
						fmt.Errorf("stream timeout due to inactivity for file '%s' after %d retries", filePath, numRetry),
						true,
						failedReplacements,
					)
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
						fmt.Printf("File %s: Error receiving stream chunk: %v\n", filePath, err)

						handleErrorRetry(
							fmt.Errorf("stream error for file '%s' after %d retries: %v", filePath, numRetry, err),
							true,
							failedReplacements,
						)
						return
					}

					if len(response.Choices) == 0 {
						handleErrorRetry(fmt.Errorf("stream error: no choices"), true, failedReplacements)
						return
					}

					choice := response.Choices[0]

					if choice.FinishReason != "" {
						if choice.FinishReason != openai.FinishReasonFunctionCall {
							handleErrorRetry(
								fmt.Errorf("stream finished without a function call. Reason: %s, File: %s", choice.FinishReason, filePath),
								false,
								failedReplacements,
							)
							return
						}

						fmt.Printf("File %s: Stream finished with non-function call\n", filePath)
						fmt.Println("finish reason: " + choice.FinishReason)

						plan := plans.Get(proposalId)
						if plan.Files[filePath] == nil {
							fmt.Printf("Stream finished before replacements parsed. File: %s\n", filePath)
							fmt.Println("Buffer:")
							fmt.Println(plan.Buffers[filePath])

							handleErrorRetry(
								fmt.Errorf("stream finished before replacements parsed. File: %s", filePath),
								true,
								failedReplacements,
							)
							return
						}
					}

					var content string
					delta := response.Choices[0].Delta

					if delta.FunctionCall == nil {
						fmt.Println("No function call in delta. File:", filePath)
						spew.Dump(delta)
						continue
					} else {
						content = delta.FunctionCall.Arguments
					}

					planTokenCount := &shared.PlanTokenCount{
						Path:      filePath,
						NumTokens: 1,
						Finished:  false,
					}

					// fmt.Printf("%s: %s", filePath, content)
					planTokenCountJson, err := json.Marshal(planTokenCount)
					if err != nil {
						onBuildFileError(filePath, fmt.Errorf("error marshalling plan chunk: %v", err))
						return
					}
					onStream(string(planTokenCountJson), nil)

					var buffer string
					plans.Update(proposalId, func(p *types.Plan) {
						p.Buffers[filePath] += content
						buffer = p.Buffers[filePath]
					})

					var replacements shared.StreamedReplacements
					err = json.Unmarshal([]byte(buffer), &replacements)
					if err == nil && len(replacements.Replacements) > 0 {
						fmt.Printf("File %s: Parsed replacements\n", filePath)

						planFile, failed := applyReplacements(filePath, currentState, proposal.Content, replacements.Replacements)

						if failed != nil {
							fmt.Println("Failed replacements:")
							spew.Dump(failed)
							handleErrorRetry(
								fmt.Errorf("failed to apply replacements to file '%s' after %d retries", filePath, numRetry),
								false,
								failed,
							)
							return
						}

						planTokenCount := &shared.PlanTokenCount{
							Path:      filePath,
							NumTokens: 0,
							Finished:  true,
						}

						// fmt.Printf("%s: %s", filePath, content)
						planTokenCountJson, err := json.Marshal(planTokenCount)
						if err != nil {
							onBuildFileError(filePath, fmt.Errorf("error marshalling plan chunk: %v", err))
							return
						}
						onStream(string(planTokenCountJson), nil)

						onFinishBuildFile(filePath, planFile)
						return
					}
				}
			}
		}()
	}

	for _, filePath := range proposal.PlanDescription.Files {
		go buildFile(filePath, 0, nil)
	}

	return nil
}

func applyReplacements(filePath string, original string, planSuggestion string, replacements []*shared.Replacement) (*shared.PlanFile, map[int]*shared.Replacement) {
	updated := original

	sort.Slice(replacements, func(i, j int) bool {
		iIdx := strings.Index(updated, replacements[i].Old)
		jIdx := strings.Index(updated, replacements[j].Old)
		return iIdx < jIdx
	})

	failedReplacements := map[int]*shared.Replacement{}

	lastInsertedIdx := 0
	for i, replacement := range replacements {
		pre := updated[:lastInsertedIdx]
		sub := updated[lastInsertedIdx:]
		originalIdx := strings.Index(sub, replacement.Old)

		if originalIdx == -1 {
			failedReplacements[i] = replacement
		} else {
			replaced := strings.Replace(sub, replacement.Old, replacement.New, 1)

			// log.Println("Replacement: " + replacement.Old + " -> " + replacement.New)
			// log.Println("Pre: " + pre)
			// log.Println("Sub: " + sub)
			// log.Println("Idx: " + fmt.Sprintf("%d", idx))
			// log.Println("Updated: " + updated)

			updated = pre + replaced

			lastInsertedIdx = lastInsertedIdx + originalIdx + len(replacement.New)
		}

	}

	if len(failedReplacements) > 0 {
		return nil, failedReplacements
	}

	return &shared.PlanFile{
		Path:    filePath,
		Content: updated,
	}, nil
}
