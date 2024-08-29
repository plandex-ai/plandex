package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/types"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (fileState *activeBuildStreamFileState) listenStreamVerifyOutput(stream *openai.ChatCompletionStream) {
	auth := fileState.auth
	filePath := fileState.filePath
	planId := fileState.plan.Id
	branch := fileState.branch

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		log.Printf("listenStreamVerifyOutput - Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return
	}

	defer stream.Close()

	// Create a timer that will trigger if no chunk is received within the specified duration
	timer := time.NewTimer(model.OPENAI_STREAM_CHUNK_TIMEOUT)
	defer timer.Stop()

	streamFinished := false

	execHookOnStop := func(sendStreamErr bool) {
		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			User:  auth.User,
			OrgId: auth.OrgId,
			Plan:  fileState.plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:   fileState.inputTokens,
				OutputTokens:  fileState.activeBuild.VerifyBufferTokens,
				ModelName:     fileState.settings.ModelPack.GetVerifier().BaseModelConfig.ModelName,
				ModelProvider: fileState.settings.ModelPack.GetVerifier().BaseModelConfig.Provider,
				ModelPackName: fileState.settings.ModelPack.Name,
				ModelRole:     shared.ModelRoleVerifier,
				Purpose:       "Verified file update",
			},
		})

		if apiErr != nil {
			log.Printf("Error executing did send model request hook after cancel or error: %v\n", apiErr)

			if sendStreamErr {
				activePlan := GetActivePlan(planId, branch)

				if activePlan == nil {
					log.Printf(" Active plan not found for plan ID %s on branch %s\n", planId, branch)
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
			log.Println("\nVerify: stream timeout due to inactivity")

			execHookOnStop(true)

			if streamFinished {
				log.Println("\nVerify stream finished-timed out waiting for usage chunk")
				return
			}

			fileState.verifyRetryOrAbort(fmt.Errorf("listenStreamVerifyOutput - stream timeout due to inactivity for file '%s'", filePath))
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
				log.Printf("listenStreamVerifyOutput - File %s: Error receiving stream chunk: %v\n", filePath, err)

				if err == context.Canceled {
					log.Printf("listenStreamVerifyOutput - File %s: Stream canceled\n", filePath)
					execHookOnStop(false)
					return
				}

				execHookOnStop(true)

				if !streamFinished {
					fileState.verifyRetryOrAbort(fmt.Errorf("listenStreamVerifyOutput - stream error: %v", err))
				}
				return
			}

			if len(response.Choices) == 0 {
				if response.Usage != nil {
					log.Println("Fix stream usage:")
					spew.Dump(response.Usage)

					_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
						User:  auth.User,
						OrgId: auth.OrgId,
						Plan:  fileState.plan,
						DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
							InputTokens:   response.Usage.PromptTokens,
							OutputTokens:  response.Usage.CompletionTokens,
							ModelName:     fileState.settings.ModelPack.GetVerifier().BaseModelConfig.ModelName,
							ModelProvider: fileState.settings.ModelPack.GetVerifier().BaseModelConfig.Provider,
							ModelPackName: fileState.settings.ModelPack.Name,
							ModelRole:     shared.ModelRoleVerifier,
							Purpose:       "Verified file update",
						},
					})

					if apiErr != nil {
						// stream has already finished so just log the error
						log.Printf("Verify stream: executing did send model request hook: %v\n", err)

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
				fileState.verifyRetryOrAbort(fmt.Errorf("listenStreamVerifyOutput - stream error: no choices"))
				return
			}

			// if stream finished and it's not a usage chunk, keep listening for usage chunk
			if streamFinished {
				log.Printf("listenStreamVerifyOutput - File %s: Stream finished, no usage chunk-will keep listening\n", filePath)
				continue
			}

			choice := response.Choices[0]
			var content string
			delta := choice.Delta

			if len(delta.ToolCalls) > 0 {
				content = delta.ToolCalls[0].Function.Arguments

				trimmed := strings.TrimSpace(content)
				if trimmed == "{%invalidjson%}" || trimmed == "``(no output)``````" {
					log.Println("File", filePath+":", "%invalidjson%} token in streamed chunk")
					execHookOnStop(true)
					fileState.verifyRetryOrAbort(fmt.Errorf("invalid JSON in streamed chunk for file '%s'", filePath))
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

				fileState.activeBuild.VerifyBuffer += content
				fileState.activeBuild.VerifyBufferTokens++
			}

			var streamed types.VerifyResult
			err = json.Unmarshal([]byte(fileState.activeBuild.VerifyBuffer), &streamed)

			if err == nil {
				log.Printf("listenStreamVerifyOutput - File %s: Parsed streamed verify result\n", filePath)
				// spew.Dump(streamed)
				streamFinished = true // Stream finished successfully

				log.Printf("listenStreamVerifyOutput - File %s: Streamed verify result\n", filePath)
				fileState.onVerifyResult(streamed)

				// Reset the timer for the usage chunk
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(model.OPENAI_USAGE_CHUNK_TIMEOUT)

				continue // continue for usage chunk
			} else if len(delta.ToolCalls) == 0 {

				log.Printf("listenStreamVerifyOuput - File %s: Stream chunk missing function call. Reason: %s\n", filePath, choice.FinishReason)

				execHookOnStop(true)
				fileState.verifyRetryOrAbort(fmt.Errorf("listenStreamVerifyOutput - stream chunk missing function call. Reason: %s, File: %s", choice.FinishReason, filePath))
			}
		}
	}

}
