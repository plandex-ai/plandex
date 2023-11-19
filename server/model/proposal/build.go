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
	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

const MaxRetries = 3
const MaxReplacementRetries = 1

func BuildPlan(params *types.BuildParams, onStream types.OnStreamFunc) error {
	// proposalId := params.ProposalId
	// fileContents := params.FileContents
	// numTokensByFile := params.NumTokensByFile
	// onStream := params.OnStream

	// goEnv := os.Getenv("GOENV")
	// if goEnv == "test" {
	// 	streamFilesLoremIpsum(onStream)
	// 	return nil
	// }

	proposalId := params.ProposalId

	proposal := proposals.Get(proposalId)
	if proposal == nil {
		return errors.New("proposal not found")
	}

	if !proposal.IsFinished() {
		return errors.New("proposal not finished")
	}

	buildUUID, err := uuid.NewRandom()
	if err != nil {
		fmt.Printf("Failed to generate build id: %v\n", err)
		return err
	}
	buildId := buildUUID.String()

	replyInfo := shared.NewReplyInfo()
	replyInfo.AddToken(proposal.Content, true)
	_, fileContents, numTokensByFile, _ := replyInfo.FinishAndRead()

	ctx, cancel := context.WithCancel(context.Background())

	builds.Set(proposalId, &types.Build{
		BuildId:    buildId,
		ProposalId: proposalId,
		NumFiles:   len(proposal.PlanDescription.Files),
		Buffers:    map[string]string{},
		Results:    map[string]*shared.PlanResult{},
		Errs:       map[string]error{},
		ProposalStage: types.ProposalStage{
			CancelFn: &cancel,
		},
	})

	onFinishBuild := func() {
		fmt.Println("Build finished. Starting write phase.")

		onStream(shared.STREAM_WRITE_PHASE, nil)

		plan := builds.Get(proposalId)
		ts := shared.StringTs()

		for _, planRes := range plan.Results {
			planRes.Ts = ts
			bytes, err := json.Marshal(planRes)
			if err != nil {
				onStream("", err)
				return
			}

			onStream(string(bytes), nil)
		}

		fmt.Println("Stream finished")
		onStream(shared.STREAM_FINISHED, nil)
	}

	onFinishBuildFile := func(filePath string, planRes *shared.PlanResult) {
		finished := false

		// fmt.Println("onFinishBuildFile: " + filePath)
		// spew.Dump(planRes)

		builds.Update(proposalId, func(plan *types.Build) {
			plan.Results[filePath] = planRes

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
		builds.Update(proposalId, func(p *types.Build) {
			p.Errs[filePath] = err
			p.SetErr(err)
		})
		onStream("", err)
	}

	var buildFile func(filePath string, numRetry int, numReplacementRetry int, res *shared.PlanResult)
	buildFile = func(filePath string, numRetry int, numReplacementsRetry int, res *shared.PlanResult) {
		fmt.Printf("Building file %s, numRetry: %d\n", filePath, numRetry)

		// get relevant file context (if any)
		var contextPart *shared.ModelContextPart
		for _, part := range proposal.Request.ModelContext {
			if part.FilePath == filePath {
				contextPart = part
				break
			}
		}

		var currentState string
		currentPlanFile, fileInCurrentPlan := proposal.Request.CurrentPlan.Files[filePath]

		if fileInCurrentPlan {
			currentState = currentPlanFile

			fmt.Printf("File %s found in current plan. Using current state.\n", filePath)
			fmt.Println("Current state:")
			fmt.Println(currentState)
		} else if contextPart != nil {
			currentState = contextPart.Body
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
			planRes := &shared.PlanResult{
				ProposalId: proposalId,
				Path:       filePath,
				Content:    fileContents[filePath],
			}
			onFinishBuildFile(filePath, planRes)
			return
		}

		fmt.Println("Getting file from model: " + filePath)
		// fmt.Println("File context:", fileContext)

		replacePrompt := prompts.GetReplacePrompt(filePath)
		currentStatePrompt := prompts.GetBuildCurrentStatePrompt(filePath, currentState)
		sysPrompt := prompts.GetBuildSysPrompt(filePath, currentStatePrompt)

		fileMessages := []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: sysPrompt,
			}, {
				Role:    openai.ChatMessageRoleUser,
				Content: proposal.Request.Prompt,
			}, {
				Role:    openai.ChatMessageRoleAssistant,
				Content: proposal.Content,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: replacePrompt,
			},
		}

		if numReplacementsRetry > 0 && res != nil {
			bytes, err := json.Marshal(res.Replacements)
			if err != nil {
				onBuildFileError(filePath, fmt.Errorf("error marshalling replacements: %v", err))
				return
			}

			correctReplacementPrompt, err := prompts.GetCorrectReplacementPrompt(res.Replacements, currentState)
			if err != nil {
				onBuildFileError(filePath, fmt.Errorf("error getting correct replacement prompt: %v", err))
				return
			}

			fileMessages = append(fileMessages,
				openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: string(bytes),
				},

				openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: correctReplacementPrompt,
				})
		}

		fmt.Println("Calling model for file: " + filePath)

		for _, msg := range fileMessages {
			fmt.Printf("%s: %s\n", msg.Role, msg.Content)
		}

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

			if numRetry >= MaxRetries {
				onBuildFileError(filePath, fmt.Errorf("failed to create plan file stream for path '%s' after %d retries: %v", filePath, numRetry, err))
			} else {
				fmt.Println("Retrying build plan for file: " + filePath)
				buildFile(filePath, numRetry+1, numReplacementsRetry, res)
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

			handleErrorRetry := func(maxRetryErr error, shouldSleep bool, isReplacementsRetry bool, res *shared.PlanResult) {
				fmt.Printf("Error for file %s: %v\n", filePath, maxRetryErr)

				if (isReplacementsRetry && numReplacementsRetry >= MaxReplacementRetries) ||
					(!isReplacementsRetry && numRetry >= MaxRetries) {
					onBuildFileError(filePath, maxRetryErr)
				} else {
					if shouldSleep {
						time.Sleep(1 * time.Second * time.Duration(math.Pow(float64(numRetry+1), 2)))
					}
					if isReplacementsRetry {
						buildFile(filePath, numRetry+1, numReplacementsRetry+1, res)
					} else {
						buildFile(filePath, numRetry+1, numReplacementsRetry, res)
					}
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
						false,
						res,
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
							false,
							res,
						)
						return
					}

					if len(response.Choices) == 0 {
						handleErrorRetry(fmt.Errorf("stream error: no choices"), true, false, res)
						return
					}

					choice := response.Choices[0]

					if choice.FinishReason != "" {
						if choice.FinishReason != openai.FinishReasonFunctionCall {
							handleErrorRetry(
								fmt.Errorf("stream finished without a function call. Reason: %s, File: %s", choice.FinishReason, filePath),
								false,
								false,
								res,
							)
							return
						}

						fmt.Printf("File %s: Stream finished with non-function call\n", filePath)
						fmt.Println("finish reason: " + choice.FinishReason)

						plan := builds.Get(proposalId)
						if plan.Results[filePath] == nil {
							fmt.Printf("Stream finished before replacements parsed. File: %s\n", filePath)
							fmt.Println("Buffer:")
							fmt.Println(plan.Buffers[filePath])

							handleErrorRetry(
								fmt.Errorf("stream finished before replacements parsed. File: %s", filePath),
								true,
								false,
								res,
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
					builds.Update(proposalId, func(p *types.Build) {
						p.Buffers[filePath] += content
						buffer = p.Buffers[filePath]
					})

					var replacements shared.StreamedReplacements
					err = json.Unmarshal([]byte(buffer), &replacements)
					if err == nil && len(replacements.Replacements) > 0 {
						fmt.Printf("File %s: Parsed replacements\n", filePath)

						planResult, allSucceeded := getPlanResult(proposalId, filePath, currentState, contextPart, replacements.Replacements)

						if !allSucceeded {
							fmt.Println("Failed replacements:")
							for _, replacement := range planResult.Replacements {
								if replacement.Failed {
									spew.Dump(replacement)
								}
							}

							handleErrorRetry(
								fmt.Errorf("failed replacements for file '%s' after %d retries", filePath, numReplacementsRetry),
								false,
								true,
								planResult,
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

						onFinishBuildFile(filePath, planResult)
						return
					}
				}
			}
		}()
	}

	for _, filePath := range proposal.PlanDescription.Files {
		go buildFile(filePath, 0, 0, nil)
	}

	return nil
}

func getPlanResult(proposalId, filePath, currentState string, contextPart *shared.ModelContextPart, replacements []*shared.Replacement) (*shared.PlanResult, bool) {
	updated := currentState

	sort.Slice(replacements, func(i, j int) bool {
		iIdx := strings.Index(updated, replacements[i].Old)
		jIdx := strings.Index(updated, replacements[j].Old)
		return iIdx < jIdx
	})

	allSucceeded := true

	lastInsertedIdx := 0
	for _, replacement := range replacements {
		pre := updated[:lastInsertedIdx]
		sub := updated[lastInsertedIdx:]
		originalIdx := strings.Index(sub, replacement.Old)

		if originalIdx == -1 {
			allSucceeded = false
			replacement.Failed = true
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

	var contextSha string
	if contextPart != nil {
		contextSha = contextPart.Sha
	}

	return &shared.PlanResult{
		ProposalId:   proposalId,
		Path:         filePath,
		Replacements: replacements,
		ContextSha:   contextSha,
		AnyFailed:    !allSucceeded,
	}, allSucceeded
}
