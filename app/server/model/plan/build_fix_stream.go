package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"plandex-server/model"
	"plandex-server/types"
	"strings"
	"time"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (fileState *activeBuildStreamFileState) listenStreamFixChanges(stream *openai.ChatCompletionStream) {
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

	for {
		select {
		case <-activePlan.Ctx.Done():
			// The main context was canceled (not the timer)
			return
		case <-timer.C:
			// Timer triggered because no new chunk was received in time
			fileState.fixRetryOrAbort(fmt.Errorf("listenStreamFixChanges - stream timeout due to inactivity for file '%s'", filePath))
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

				fileState.fixRetryOrAbort(fmt.Errorf("listenStreamFixChanges - stream error: %v", err))
				return
			}

			if len(response.Choices) == 0 {
				fileState.fixRetryOrAbort(fmt.Errorf("listenStreamFixChanges - stream error: no choices"))
				return
			}

			var content string
			delta := response.Choices[0].Delta

			if len(delta.ToolCalls) > 0 {
				content = delta.ToolCalls[0].Function.Arguments

				trimmed := strings.TrimSpace(content)
				if trimmed == "{%invalidjson%}" || trimmed == "``(no output)``````" {
					log.Println("listenStreamFixChanges - File", filePath+":", "%invalidjson%} token in streamed chunk")
					fileState.fixRetryOrAbort(fmt.Errorf("listenStreamFixChanges - invalid JSON in streamed chunk for file '%s'", filePath))
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

					fileState.fixRetryOrAbort(fmt.Errorf("listenStreamFixChanges - stream buffer tokens too high for file '%s'", filePath))
					return
				}
			}

			var streamed types.ChangesWithLineNums
			err = json.Unmarshal([]byte(fileState.activeBuild.FixBuffer), &streamed)

			if err == nil {
				log.Printf("listenStreamFixChanges - File %s: Parsed streamed replacements\n", filePath)
				// spew.Dump(streamed)
				fileState.onFixResult(streamed)
				return
			} else if len(delta.ToolCalls) == 0 {
				log.Println("listenStreamFixChanges - Stream chunk missing function call.")

				fileState.fixRetryOrAbort(fmt.Errorf("listenStreamFixChanges - stream chunk missing function call. File: %s", filePath))
				return
			}
		}
	}
}
