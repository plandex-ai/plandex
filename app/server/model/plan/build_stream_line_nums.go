package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/types"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (fileState *activeBuildStreamFileState) listenStreamChangesWithLineNums(stream *openai.ChatCompletionStream) {
	auth := fileState.auth
	filePath := fileState.filePath
	planId := fileState.plan.Id
	branch := fileState.branch

	log.Printf("listenStream - Listening for streamed changes with line numbers for file %s\n", filePath)

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		log.Printf("listenStream - Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return
	}

	defer stream.Close()

	// Create a timer that will trigger if no chunk is received within the specified duration
	timer := time.NewTimer(model.OPENAI_STREAM_CHUNK_TIMEOUT)
	defer timer.Stop()

	streamFinished := false

	execHookOnStop := func(sendStreamErr bool) {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: fileState.plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:   fileState.inputTokens,
				OutputTokens:  fileState.activeBuild.WithLineNumsBufferTokens,
				ModelName:     fileState.settings.ModelPack.Builder.BaseModelConfig.ModelName,
				ModelProvider: fileState.settings.ModelPack.Builder.BaseModelConfig.Provider,
				ModelPackName: fileState.settings.ModelPack.Name,
				ModelRole:     shared.ModelRoleBuilder,
				Purpose:       "Generated file update",
			},
		})

		if apiErr != nil {
			log.Printf("Error executing did send model request hook after cancel or error: %v\n", apiErr)

			if sendStreamErr {
				activePlan := GetActivePlan(planId, branch)

				if activePlan == nil {
					log.Printf("listenStream - Active plan not found for plan ID %s on branch %s\n", planId, branch)
					return
				}

				activePlan.StreamDoneCh <- apiErr
			}
		}
	}

	log.Printf("listenStream - File %s: Listening for stream chunks\n", filePath)

	for {
		select {
		case <-activePlan.Ctx.Done():
			// The main context was canceled (not the timer)
			execHookOnStop(false)
			return
		case <-timer.C:
			log.Println("\nBuild: stream timed out due to inactivity")
			execHookOnStop(true)
			if streamFinished {
				log.Println("\nBuild stream finished-timed out waiting for usage chunk")
				return
			}

			fileState.lineNumsRetryOrError(fmt.Errorf("listenStream - stream timeout due to inactivity for file '%s' | This usually means the model is not responding.", filePath))
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
				log.Printf("listenStream - File %s: Error receiving stream chunk: %v\n", filePath, err)

				if err == context.Canceled {
					log.Printf("listenStream - File %s: Stream canceled\n", filePath)
					// log.Println("current buffer:")
					// log.Println(fileState.activeBuild.WithLineNumsBuffer)
					execHookOnStop(false)
					return
				}

				execHookOnStop(true)

				if !streamFinished {
					fileState.lineNumsRetryOrError(fmt.Errorf("listenStream - stream error for file '%s': %v", filePath, err))
				}
				return
			}

			if len(response.Choices) == 0 {
				if response.Usage != nil {
					log.Printf("listenStream - File %s: Stream usage chunk\n", filePath)
					spew.Dump(response.Usage)

					_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
						Auth: auth,
						Plan: fileState.plan,
						DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
							InputTokens:   response.Usage.PromptTokens,
							OutputTokens:  response.Usage.CompletionTokens,
							ModelName:     fileState.settings.ModelPack.Builder.BaseModelConfig.ModelName,
							ModelProvider: fileState.settings.ModelPack.Builder.BaseModelConfig.Provider,
							ModelPackName: fileState.settings.ModelPack.Name,
							ModelRole:     shared.ModelRoleBuilder,
							Purpose:       "Generated file update",
						},
					})

					if apiErr != nil {
						log.Printf("Build stream: error executing did send model request hook: %v\n", apiErr)

						// ensure the active plan is still available
						activePlan := GetActivePlan(planId, branch)

						if activePlan == nil {
							log.Printf(" Active plan not found for plan ID %s on branch %s\n", planId, branch)
							return
						}

						activePlan.StreamDoneCh <- apiErr
					}
					return
				}

				execHookOnStop(true)
				fileState.lineNumsRetryOrError(fmt.Errorf("listenStream - stream error: no choices | This usually means the model failed to generate a valid response."))
				return
			}

			// if stream finished and it's not a usage chunk, keep listening for usage chunk
			if streamFinished {
				log.Printf("listenStream - File %s: Stream finished, no usage chunk-will keep listening\n", filePath)
				continue
			}

			choice := response.Choices[0]
			var content string
			delta := response.Choices[0].Delta

			if len(delta.ToolCalls) > 0 {
				content = delta.ToolCalls[0].Function.Arguments

				trimmed := strings.TrimSpace(content)
				if trimmed == "{%invalidjson%}" || trimmed == "``(no output)``````" {
					log.Println("File", filePath+":", "%invalidjson%} token in streamed chunk")
					execHookOnStop(true)
					fileState.lineNumsRetryOrError(fmt.Errorf("Invalid JSON in streamed chunk for file '%s' | This usually means the model failed to generate valid JSON.", filePath))
					return
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

				fileState.activeBuild.WithLineNumsBuffer += content
				fileState.activeBuild.WithLineNumsBufferTokens++

				// After a reasonable threshhold, if buffer has significantly more tokens than original file + proposed changes, something is wrong
				cutoff := int(math.Max(float64(fileState.activeBuild.CurrentFileTokens+fileState.activeBuild.FileContentTokens), 500) * 20)
				if fileState.activeBuild.WithLineNumsBufferTokens > 500 && fileState.activeBuild.WithLineNumsBufferTokens > cutoff {
					log.Printf("File %s: Stream buffer tokens too high\n", filePath)
					log.Printf("Current file tokens: %d\n", fileState.activeBuild.CurrentFileTokens)
					log.Printf("File content tokens: %d\n", fileState.activeBuild.FileContentTokens)
					log.Printf("Cutoff: %d\n", cutoff)
					log.Printf("Buffer tokens: %d\n", fileState.activeBuild.WithLineNumsBufferTokens)
					log.Println("Buffer:")
					log.Println(fileState.activeBuild.WithLineNumsBuffer)

					execHookOnStop(true)
					fileState.lineNumsRetryOrError(fmt.Errorf("listenStream - stream buffer tokens too high for file '%s' | This usually means the model failed to generate valid JSON.", filePath))
					return
				}
			}

			var streamed types.ChangesWithLineNums
			err = json.Unmarshal([]byte(fileState.activeBuild.WithLineNumsBuffer), &streamed)

			if err == nil {
				log.Printf("listenStream - File %s: Parsed streamed replacements\n", filePath)
				// spew.Dump(streamed)

				streamFinished = true // Stream finished successfully

				log.Printf("listenStream - File %s: calling onBuildResult\n", filePath)
				fileState.onBuildResult(streamed)

				// Reset the timer for the usage chunk
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(model.OPENAI_USAGE_CHUNK_TIMEOUT)

				log.Printf("listenStream - File %s: continue for usage chunk\n", filePath)

				continue // continue for usage chunk
			} else if len(delta.ToolCalls) == 0 {
				log.Println("listenStream - Stream chunk missing function call.")
				// log.Println(spew.Sdump(response))
				// log.Println(spew.Sdump(fileState))

				// log.Println("Current buffer:")
				// log.Println(fileState.activeBuild.WithLineNumsBuffer)

				execHookOnStop(true)
				fileState.lineNumsRetryOrError(fmt.Errorf("listenStream - stream chunk missing function call. Reason: %s, File: %s | This usually means the model failed to generate a valid response.", choice.FinishReason, filePath))
				return
			}
		}
	}
}
