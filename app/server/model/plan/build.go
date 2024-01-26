package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"plandex-server/db"
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

func QueueBuild(currentOrgId, currentUserId, planId, branch string, activeBuild *types.ActiveBuild) {
	activePlan := Active.Get(planId)
	filePath := activeBuild.Path

	Active.Update(planId, func(active *types.ActivePlan) {
		active.BuildQueuesByPath[filePath] = append(active.BuildQueuesByPath[filePath], activeBuild)
	})
	log.Printf("Queued build for file %s\n", filePath)

	if activePlan.IsBuildingByPath[filePath] {
		log.Printf("Already building file %s\n", filePath)
		return
	} else {
		log.Printf("Will process build queue for file %s\n", filePath)
		go execPlanBuild(currentOrgId, currentUserId, branch, activePlan, activeBuild)
	}
}

func execPlanBuild(currentOrgId, currentUserId, branch string, activePlan *types.ActivePlan, activeBuild *types.ActiveBuild) {
	Active.Update(activePlan.Id, func(ap *types.ActivePlan) {
		ap.IsBuildingByPath[activeBuild.Path] = true
	})

	planId := activePlan.Id
	filePath := activeBuild.Path

	buildInfo := &shared.BuildInfo{
		Path:      filePath,
		NumTokens: 0,
		Finished:  false,
	}
	activePlan.Stream(shared.StreamMessage{
		Type:      shared.StreamMessageBuildInfo,
		BuildInfo: buildInfo,
	})

	replyInfo := types.NewReplyParser()
	replyInfo.AddChunk(activePlan.CurrentReplyContent, true)
	_, fileContents, _, _ := replyInfo.Read()

	errCh := make(chan error)

	var build *db.PlanBuild
	var currentPlan *shared.CurrentPlanState

	go func() {
		build = &db.PlanBuild{
			OrgId:          currentOrgId,
			PlanId:         planId,
			ConvoMessageId: activeBuild.AssistantMessageId,
			FilePath:       filePath,
		}
		err := db.StorePlanBuild(build)

		if err != nil {
			errCh <- fmt.Errorf("error storing plan build: %v", err)
			return
		}

		errCh <- nil
	}()

	go func() {
		err := db.SetPlanStatus(planId, shared.PlanStatusBuilding, "")
		if err != nil {
			errCh <- fmt.Errorf("error setting plan status to building: %v", err)
			return
		}
		errCh <- nil
	}()

	go func() {
		res, err := db.GetCurrentPlanState(db.CurrentPlanStateParams{
			OrgId:    currentOrgId,
			PlanId:   planId,
			Contexts: activePlan.Contexts,
		})
		if err != nil {
			errCh <- fmt.Errorf("error getting current plan state: %v", err)
			return
		}
		currentPlan = res
		errCh <- nil
	}()

	for i := 0; i < 3; i++ {
		err := <-errCh
		if err != nil {
			log.Printf("Error building plan %s: %v\n", planId, err)
			Active.Update(activePlan.Id, func(ap *types.ActivePlan) {
				ap.IsBuildingByPath[activeBuild.Path] = false
			})
			activePlan.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    err.Error(),
			}
			return
		}
	}

	onFinishBuild := func() {
		log.Println("Build finished")

		if Active.Get(planId).RepliesFinished {
			activePlan.Stream(shared.StreamMessage{
				Type: shared.StreamMessageFinished,
			})
		}
	}

	onFinishBuildFile := func(filePath string, planRes *db.PlanFileResult) {
		finished := false

		// log.Println("onFinishBuildFile: " + filePath)
		// spew.Dump(planRes)

		repoLockId, err := db.LockRepo(currentOrgId, currentUserId, planId, branch, db.LockScopeWrite)
		if err != nil {
			log.Printf("Error locking repo: %v\n", err)
			activePlan.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Error locking repo: " + err.Error(),
			}
			return
		}

		err = func() error {
			defer func() {
				err := db.UnlockRepo(repoLockId)
				if err != nil {
					log.Printf("Error unlocking repo: %v\n", err)
				}
			}()

			err = db.StorePlanResult(planRes)
			if err != nil {
				log.Printf("Error storing plan result: %v\n", err)
				activePlan.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Error storing plan result: " + err.Error(),
				}
				return err
			}
			return nil
		}()

		if err != nil {
			return
		}

		activeBuild.Success = true

		Active.Update(planId, func(ap *types.ActivePlan) {
			ap.BuiltFiles[filePath] = true
			ap.IsBuildingByPath[filePath] = false

			if ap.BuildFinished() {
				finished = true
			}
		})

		if !activePlan.PathFinished(filePath) {
			log.Printf("Processing next build for file %s\n", filePath)
			var nextBuild *types.ActiveBuild
			for _, build := range activePlan.BuildQueuesByPath[filePath] {
				if !build.BuildFinished() {
					nextBuild = build
					break
				}
			}
			if nextBuild != nil {
				go execPlanBuild(currentOrgId, currentUserId, branch, activePlan, nextBuild)
			}
			return
		}

		log.Printf("Finished building file %s\n", filePath)

		if finished {
			onFinishBuild()
		}
	}

	onBuildFileError := func(filePath string, err error) {
		log.Printf("Error for file %s: %v\n", filePath, err)

		activeBuild.Error = err

		activePlan.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    err.Error(),
		}

		if err != nil {
			log.Printf("Error storing plan error result: %v\n", err)
		}

		build.Error = err.Error()

		err = db.SetBuildError(build)
		if err != nil {
			log.Printf("Error setting build error: %v\n", err)
		}

		Active.Update(activePlan.Id, func(ap *types.ActivePlan) {
			ap.IsBuildingByPath[activeBuild.Path] = false
		})
	}

	var buildFile func(filePath string, numRetry int, numReplacementRetry int, res *db.PlanFileResult)
	buildFile = func(filePath string, numRetry int, numReplacementsRetry int, res *db.PlanFileResult) {
		log.Printf("Building file %s, numRetry: %d\n", filePath, numRetry)

		// get relevant file context (if any)
		contextPart := activePlan.ContextsByPath[filePath]

		var currentState string
		currentPlanFile, fileInCurrentPlan := currentPlan.CurrentPlanFiles.Files[filePath]

		if fileInCurrentPlan {
			currentState = currentPlanFile

			log.Printf("File %s found in current plan. Using current state.\n", filePath)
			log.Println("Current state:")
			log.Println(currentState)
		} else if contextPart != nil {
			currentState = contextPart.Body
		}

		if currentState == "" {
			log.Printf("File %s not found in model context or current plan. Creating new file.\n", filePath)

			buildInfo := &shared.BuildInfo{
				Path:      filePath,
				NumTokens: 0,
				Finished:  true,
			}

			activePlan.Stream(shared.StreamMessage{
				Type:      shared.StreamMessageBuildInfo,
				BuildInfo: buildInfo,
			})

			// new file
			planRes := &db.PlanFileResult{
				OrgId:          currentOrgId,
				PlanId:         planId,
				PlanBuildId:    build.Id,
				ConvoMessageId: build.ConvoMessageId,
				Path:           filePath,
				Content:        fileContents[filePath],
			}
			onFinishBuildFile(filePath, planRes)
			return
		}

		log.Println("Getting file from model: " + filePath)
		// log.Println("File context:", fileContext)

		replacePrompt := prompts.GetReplacePrompt(filePath)
		currentStatePrompt := prompts.GetBuildCurrentStatePrompt(filePath, currentState)
		sysPrompt := prompts.GetBuildSysPrompt(filePath, currentStatePrompt)

		fileMessages := []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: sysPrompt,
			}, {
				Role:    openai.ChatMessageRoleUser,
				Content: activePlan.Prompt,
			}, {
				Role:    openai.ChatMessageRoleAssistant,
				Content: activePlan.CurrentReplyContent,
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

		log.Println("Calling model for file: " + filePath)

		// for _, msg := range fileMessages {
		// 	log.Printf("%s: %s\n", msg.Role, msg.Content)
		// }

		modelReq := openai.ChatCompletionRequest{
			Model:          model.BuilderModel,
			Functions:      []openai.FunctionDefinition{prompts.ReplaceFn},
			Messages:       fileMessages,
			Temperature:    0.2,
			TopP:           0.1,
			ResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
		}

		stream, err := model.Client.CreateChatCompletionStream(activePlan.Ctx, modelReq)
		if err != nil {
			log.Printf("Error creating plan file stream for path '%s': %v\n", filePath, err)

			if numRetry >= MaxRetries {
				onBuildFileError(filePath, fmt.Errorf("failed to create plan file stream for path '%s' after %d retries: %v", filePath, numRetry, err))
			} else {
				log.Println("Retrying build plan for file: " + filePath)
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

			handleErrorRetry := func(maxRetryErr error, shouldSleep bool, isReplacementsRetry bool, res *db.PlanFileResult) {
				log.Printf("Error for file %s: %v\n", filePath, maxRetryErr)

				if isReplacementsRetry && numReplacementsRetry >= MaxReplacementRetries {
					// in this case, we just want to ignore the error and continue
				} else if !isReplacementsRetry && numRetry >= MaxRetries {
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
				case <-activePlan.Ctx.Done():
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
					} else {
						log.Printf("File %s: Error receiving stream chunk: %v\n", filePath, err)

						if err == context.Canceled {
							log.Printf("File %s: Stream canceled\n", filePath)
							return
						}

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

						log.Printf("File %s: Stream finished with non-function call\n", filePath)
						log.Println("finish reason: " + choice.FinishReason)

						active := Active.Get(planId)
						if !active.BuiltFiles[filePath] {
							log.Printf("Stream finished before replacements parsed. File: %s\n", filePath)
							log.Println("Buffer:")
							log.Println(activeBuild.Buffer)

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
						log.Println("No function call in delta. File:", filePath)
						spew.Dump(delta)
						continue
					} else {
						content = delta.FunctionCall.Arguments
					}

					buildInfo := &shared.BuildInfo{
						Path:      filePath,
						NumTokens: 1,
						Finished:  false,
					}

					// log.Printf("%s: %s", filePath, content)
					activePlan.Stream(shared.StreamMessage{
						Type:      shared.StreamMessageBuildInfo,
						BuildInfo: buildInfo,
					})

					activeBuild.Buffer += content

					var streamed types.StreamedReplacements
					err = json.Unmarshal([]byte(activeBuild.Buffer), &streamed)
					if err == nil && len(streamed.Replacements) > 0 {
						log.Printf("File %s: Parsed replacements\n", filePath)

						planFileResult, allSucceeded := getPlanResult(
							planResultParams{
								orgId:          currentOrgId,
								planId:         planId,
								planBuildId:    build.Id,
								convoMessageId: build.ConvoMessageId,
								filePath:       filePath,
								currentState:   currentState,
								context:        contextPart,
								replacements:   streamed.Replacements,
							},
						)

						// proposalId, filePath, currentState, contextPart, replacements.Replacements)

						if !allSucceeded {
							log.Println("Failed replacements:")
							for _, replacement := range planFileResult.Replacements {
								if replacement.Failed {
									spew.Dump(replacement)
								}
							}

							if numReplacementsRetry < MaxReplacementRetries {
								handleErrorRetry(
									nil, // no error -- if we reach MAX_REPLACEMENT_RETRIES, we just ignore the error and continue
									false,
									true,
									planFileResult,
								)
								return
							}
						}

						buildInfo := &shared.BuildInfo{
							Path:      filePath,
							NumTokens: 0,
							Finished:  true,
						}
						activePlan.Stream(shared.StreamMessage{
							Type:      shared.StreamMessageBuildInfo,
							BuildInfo: buildInfo,
						})

						onFinishBuildFile(filePath, planFileResult)
						return
					}
				}
			}
		}()
	}

	buildFile(filePath, 0, 0, nil)
}

type planResultParams struct {
	orgId          string
	planId         string
	planBuildId    string
	convoMessageId string
	filePath       string
	currentState   string
	context        *db.Context
	replacements   []*shared.Replacement
}

func getPlanResult(params planResultParams) (*db.PlanFileResult, bool) {
	orgId := params.orgId
	planId := params.planId
	planBuildId := params.planBuildId
	filePath := params.filePath
	currentState := params.currentState
	contextPart := params.context
	replacements := params.replacements
	updated := params.currentState

	sort.Slice(replacements, func(i, j int) bool {
		iIdx := strings.Index(updated, replacements[i].Old)
		jIdx := strings.Index(updated, replacements[j].Old)
		return iIdx < jIdx
	})

	_, allSucceeded := shared.ApplyReplacements(currentState, replacements, true)

	var contextSha string
	if contextPart != nil {
		contextSha = contextPart.Sha
	}

	for _, replacement := range replacements {
		id := uuid.New().String()
		replacement.Id = id
	}

	return &db.PlanFileResult{
		OrgId:          orgId,
		PlanId:         planId,
		PlanBuildId:    planBuildId,
		ConvoMessageId: params.convoMessageId,
		Content:        "",
		Path:           filePath,
		Replacements:   replacements,
		ContextSha:     contextSha,
		AnyFailed:      !allSucceeded,
	}, allSucceeded
}
