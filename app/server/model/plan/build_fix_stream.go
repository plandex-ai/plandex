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

func (fileState *activeBuildStreamFileState) listenStreamFixChanges(stream *openai.ChatCompletionStream) {
	auth := fileState.auth
	filePath := fileState.filePath
	planId := fileState.plan.Id
	branch := fileState.branch

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		log.Printf("listenStreamFixChanges - Active plan not found for plan ID %s on branch %s\n", planId, branch)
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
				OutputTokens:  fileState.activeBuild.FixBufferTokens,
				ModelName:     fileState.settings.ModelPack.GetAutoFix().BaseModelConfig.ModelName,
				ModelProvider: fileState.settings.ModelPack.GetAutoFix().BaseModelConfig.Provider,
				ModelPackName: fileState.settings.ModelPack.Name,
				ModelRole:     shared.ModelRoleAutoFix,
				Purpose:       "Generated file update for auto-fix",
			},
		})

		if apiErr != nil {
			log.Printf("Error executing did send model request hook after cancel or error: %v\n", apiErr)

			if sendStreamErr {
				activePlan := GetActivePlan(planId, branch)

				if activePlan == nil {
					log.Printf("listenStreamFixChanges - Active plan not found for plan ID %s on branch %s\n", planId, branch)
					return
				}

				activePlan.StreamDoneCh <- apiErr
			}
		}
	}

	for {
		select {
		case <-activePlan.Ctx.Done():
			// The main context was canceled (not the timer)
			execHookOnStop(false)
			return
		case <-timer.C:
			log.Println("\nFix: stream timed out due to inactivity")

			execHookOnStop(true)

			if streamFinished {
				log.Println("\nFix stream finished-timed out waiting for usage chunk")
				return
			}

			fileState.fixRetryOrAbort(fmt.Errorf("listenStreamFixChanges - stream timeout due to inactivity for file '%s' | This usually means the model is not responding.", filePath))
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
				log.Printf("listenStreamFixChanges - File %s: Error receiving stream chunk: %v\n", filePath, err)

				if err == context.Canceled {
					log.Printf("listenStreamFixChanges - File %s: Stream canceled\n", filePath)
					execHookOnStop(false)
					return
				}

				execHookOnStop(true)

				if !streamFinished {
					fileState.fixRetryOrAbort(fmt.Errorf("listenStreamFixChanges - stream error: %v", err))
				}
				return
			}

			if len(response.Choices) == 0 {
				if response.Usage != nil {
					log.Println("Fix stream usage:")
					spew.Dump(response.Usage)

					_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
						Auth: auth,
						Plan: fileState.plan,
						DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
							InputTokens:   response.Usage.PromptTokens,
							OutputTokens:  response.Usage.CompletionTokens,
							ModelName:     fileState.settings.ModelPack.GetAutoFix().BaseModelConfig.ModelName,
							ModelProvider: fileState.settings.ModelPack.GetAutoFix().BaseModelConfig.Provider,
							ModelPackName: fileState.settings.ModelPack.Name,
							ModelRole:     shared.ModelRoleAutoFix,
							Purpose:       "Generated file update for auto-fix",
						},
					})

					if apiErr != nil {
						log.Printf("Fix stream: error executing did send model request hook: %v\n", apiErr)

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
				fileState.fixRetryOrAbort(fmt.Errorf("listenStreamFixChanges - stream error: no choices | This usually means the model failed to generate a valid response."))
				return
			}

			// if stream finished and it's not a usage chunk, keep listening for usage chunk
			if streamFinished {
				log.Printf("listenStreamFixChanges - File %s: Stream finished, no usage chunk-will keep listening\n", filePath)
				continue
			}

			var content string
			choice := response.Choices[0]
			delta := choice.Delta

			if len(delta.ToolCalls) > 0 {
				content = delta.ToolCalls[0].Function.Arguments

				trimmed := strings.TrimSpace(content)
				if trimmed == "{%invalidjson%}" || trimmed == "``(no output)``````" {
					log.Println("listenStreamFixChanges - File", filePath+":", "%invalidjson%} token in streamed chunk")
					execHookOnStop(true)
					fileState.fixRetryOrAbort(fmt.Errorf("listenStreamFixChanges - invalid JSON in streamed chunk for file '%s' | This usually means the model failed to generate valid JSON.", filePath))
					return
				}

				buildInfo := &shared.BuildInfo{
					Path:      filePath,
					NumTokens: 1,
					Finished:  false,
				}

				// log.Printf("listenStreamFixChanges - %s: %s", filePath, content)
				activePlan.Stream(shared.StreamMessage{
					Type:      shared.StreamMessageBuildInfo,
					BuildInfo: buildInfo,
				})

				fileState.activeBuild.FixBuffer += content
				fileState.activeBuild.FixBufferTokens++

				// After a reasonable threshhold, if buffer has significantly more tokens than original file + proposed changes, something is wrong
				cutoff := int(math.Max(float64(fileState.activeBuild.CurrentFileTokens+fileState.activeBuild.FileContentTokens), 500) * 20)
				if fileState.activeBuild.FixBufferTokens > 500 && fileState.activeBuild.FixBufferTokens > cutoff {
					log.Printf("File %s: Stream buffer tokens too high\n", filePath)
					log.Printf("Current file tokens: %d\n", fileState.activeBuild.CurrentFileTokens)
					log.Printf("File content tokens: %d\n", fileState.activeBuild.FileContentTokens)
					log.Printf("Cutoff: %d\n", cutoff)
					log.Printf("Buffer tokens: %d\n", fileState.activeBuild.FixBufferTokens)
					log.Println("Buffer:")
					log.Println(fileState.activeBuild.FixBuffer)

					execHookOnStop(true)
					fileState.fixRetryOrAbort(fmt.Errorf("listenStreamFixChanges - stream buffer tokens too high for file '%s' | This usually means the model failed to generate valid JSON.", filePath))
					return
				}
			}

			var streamed types.ChangesWithLineNums
			err = json.Unmarshal([]byte(fileState.activeBuild.FixBuffer), &streamed)

			if err == nil {
				log.Printf("listenStreamFixChanges - File %s: Parsed streamed replacements\n", filePath)
				// spew.Dump(streamed)
				streamFinished = true // Stream finished successfully

				log.Printf("listenStreamFixChanges - File %s: Stream finished\n", filePath)
				fileState.onFixResult(streamed)

				// Reset the timer for the usage chunk
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(model.OPENAI_USAGE_CHUNK_TIMEOUT)

				continue // continue for usage chunk
			} else if len(delta.ToolCalls) == 0 {
				log.Printf("listenStreamFixChanges - File %s: Stream chunk missing function call. Reason: %s\n", filePath, choice.FinishReason)

				execHookOnStop(true)
				fileState.fixRetryOrAbort(fmt.Errorf("listenStreamFixChanges - stream chunk missing function call. File: %s | This usually means the model failed to generate a valid response.", filePath))
				return
			}
		}
	}
}
