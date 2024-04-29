package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"plandex-server/model"
	"plandex-server/types"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (fileState *activeBuildStreamFileState) listenStreamFixChanges(stream *openai.ChatCompletionStream) {
	filePath := fileState.filePath
	build := fileState.build
	currentOrgId := fileState.currentOrgId
	planId := fileState.plan.Id
	branch := fileState.branch
	activeBuild := fileState.activeBuild

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		log.Printf("listenStreamFixChanges - Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return
	}

	defer stream.Close()

	// Create a timer that will trigger if no chunk is received within the specified duration
	timer := time.NewTimer(model.OPENAI_STREAM_CHUNK_TIMEOUT)
	defer timer.Stop()

	for {
		select {
		case <-activePlan.Ctx.Done():
			// The main context was canceled (not the timer)
			return
		case <-timer.C:
			// Timer triggered because no new chunk was received in time
			fileState.fixRetryOrError(fmt.Errorf("listenStreamFixChanges - stream timeout due to inactivity for file '%s'", filePath))
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
					return
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
				log.Println("Build fix stream - Plan file result:")
				fileState.onFinishBuildFile(nil)
				return
			}

			if len(response.Choices) == 0 {
				buildInfo := &shared.BuildInfo{
					Path:      filePath,
					NumTokens: 0,
					Finished:  true,
				}
				activePlan.Stream(shared.StreamMessage{
					Type:      shared.StreamMessageBuildInfo,
					BuildInfo: buildInfo,
				})
				log.Println("Build fix stream - Plan file result:")
				fileState.onFinishBuildFile(nil)
				return
			}

			var content string
			delta := response.Choices[0].Delta

			if len(delta.ToolCalls) > 0 {
				content = delta.ToolCalls[0].Function.Arguments

				trimmed := strings.TrimSpace(content)
				if trimmed == "{%invalidjson%}" || trimmed == "``(no output)``````" {
					log.Println("listenStreamFixChanges - File", filePath+":", "%invalidjson%} token in streamed chunk")
					fileState.fixRetryOrError(fmt.Errorf("listenStreamFixChanges - invalid JSON in streamed chunk for file '%s'", filePath))

					buildInfo := &shared.BuildInfo{
						Path:      filePath,
						NumTokens: 0,
						Finished:  true,
					}
					activePlan.Stream(shared.StreamMessage{
						Type:      shared.StreamMessageBuildInfo,
						BuildInfo: buildInfo,
					})
					log.Println("Build fix stream - Plan file result:")
					fileState.onFinishBuildFile(nil)

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
				cutoff := int(math.Max(float64(fileState.activeBuild.CurrentFileTokens+fileState.activeBuild.FileContentTokens), 500) * 2)
				if fileState.activeBuild.FixBufferTokens > 500 && fileState.activeBuild.FixBufferTokens > cutoff {
					log.Printf("File %s: Stream buffer tokens too high\n", filePath)
					log.Printf("Current file tokens: %d\n", fileState.activeBuild.CurrentFileTokens)
					log.Printf("File content tokens: %d\n", fileState.activeBuild.FileContentTokens)
					log.Printf("Cutoff: %d\n", cutoff)
					log.Printf("Buffer tokens: %d\n", fileState.activeBuild.FixBufferTokens)
					log.Println("Buffer:")
					log.Println(fileState.activeBuild.FixBuffer)

					buildInfo := &shared.BuildInfo{
						Path:      filePath,
						NumTokens: 0,
						Finished:  true,
					}
					activePlan.Stream(shared.StreamMessage{
						Type:      shared.StreamMessageBuildInfo,
						BuildInfo: buildInfo,
					})
					log.Println("Build fix stream - Plan file result:")
					fileState.onFinishBuildFile(nil)
					return
				}
			}

			var streamed types.StreamedChangesWithLineNums
			err = json.Unmarshal([]byte(fileState.activeBuild.FixBuffer), &streamed)

			if err == nil {
				log.Printf("listenStreamFixChanges - File %s: Parsed streamed replacements\n", filePath)
				// spew.Dump(streamed)

				fileState.streamedChangesWithLineNums = streamed.Changes

				var overlapStrategy OverlapStrategy = OverlapStrategyError
				if fileState.fixFileNumRetry > 1 {
					overlapStrategy = OverlapStrategySkip
				}

				planFileResult, _, allSucceeded, err := getPlanResult(
					planResultParams{
						orgId:                       currentOrgId,
						planId:                      planId,
						planBuildId:                 build.Id,
						convoMessageId:              build.ConvoMessageId,
						filePath:                    filePath,
						preBuildState:               fileState.updated,
						fileContent:                 activeBuild.FileContent,
						streamedChangesWithLineNums: streamed.Changes,
						overlapStrategy:             overlapStrategy,
					},
				)

				if err != nil {
					log.Println("listenStreamFixChanges - Error getting plan result:", err)
					buildInfo := &shared.BuildInfo{
						Path:      filePath,
						NumTokens: 0,
						Finished:  true,
					}
					activePlan.Stream(shared.StreamMessage{
						Type:      shared.StreamMessageBuildInfo,
						BuildInfo: buildInfo,
					})
					log.Println("Build fix stream - Plan file result:")
					fileState.onFinishBuildFile(nil)
					return
				}

				if !allSucceeded {
					log.Println("listenStreamFixChanges - Failed replacements:")
					for _, replacement := range planFileResult.Replacements {
						if replacement.Failed {
							spew.Dump(replacement)
						}
					}

					// no retry here as this should never happen
					fileState.onBuildFileError(fmt.Errorf("listenStreamFixChanges - replacements failed for file '%s'", filePath))
					return

				}

				log.Println("listenStreamFixChanges - Plan file result:")
				spew.Dump(planFileResult)

				fileState.onFinishBuildFile(planFileResult)
				return
			} else if len(delta.ToolCalls) == 0 {
				log.Println("listenStreamFixChanges - Stream chunk missing function call.")

				buildInfo := &shared.BuildInfo{
					Path:      filePath,
					NumTokens: 0,
					Finished:  true,
				}
				activePlan.Stream(shared.StreamMessage{
					Type:      shared.StreamMessageBuildInfo,
					BuildInfo: buildInfo,
				})
				log.Println("Build fix stream - Plan file result:")
				fileState.onFinishBuildFile(nil)
				return
			}
		}
	}
}

func (fileState *activeBuildStreamFileState) fixRetryOrError(err error) {
	if fileState.fixFileNumRetry < MaxBuildStreamErrorRetries {
		fileState.fixFileNumRetry++
		fileState.activeBuild.FixBuffer = ""
		fileState.activeBuild.FixBufferTokens = 0
		log.Printf("Retrying fix file '%s' due to error: %v\n", fileState.filePath, err)

		// Exponential backoff
		time.Sleep(time.Duration(fileState.fixFileNumRetry*fileState.fixFileNumRetry)*time.Second + time.Duration(rand.Intn(1001))*time.Millisecond)

		fileState.fixFileLineNums()
	} else {
		fileState.onBuildFileError(err)
	}
}
