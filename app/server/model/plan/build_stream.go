package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/model"
	"plandex-server/types"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

const MaxBuildStreamErrorRetries = 3 // uses naive exponential backoff so be careful about setting this too high

type activeBuildStreamState struct {
	client        *openai.Client
	auth          *types.ServerAuth
	currentOrgId  string
	currentUserId string
	plan          *db.Plan
	branch        string
	settings      *shared.PlanSettings
	modelContext  []*db.Context
}

type activeBuildStreamFileState struct {
	*activeBuildStreamState
	filePath         string
	convoMessageId   string
	build            *db.PlanBuild
	currentPlanState *shared.CurrentPlanState
	activeBuild      *types.ActiveBuild
	currentState     string
	numRetry         int
}

func (fileState *activeBuildStreamFileState) listenStream(stream *openai.ChatCompletionStream) {
	filePath := fileState.filePath
	build := fileState.build
	currentOrgId := fileState.currentOrgId
	planId := fileState.plan.Id
	branch := fileState.branch
	currentState := fileState.currentState
	activeBuild := fileState.activeBuild

	activePlan := GetActivePlan(planId, branch)

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
			fileState.retryOrError(fmt.Errorf("stream timeout due to inactivity for file '%s'", filePath))
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
					log.Println("current buffer:")
					log.Println(fileState.activeBuild.Buffer)
					return
				}

				fileState.retryOrError(fmt.Errorf("stream error for file '%s': %v", filePath, err))
				return
			}

			if len(response.Choices) == 0 {
				fileState.retryOrError(fmt.Errorf("stream error: no choices"))
				return
			}

			choice := response.Choices[0]
			var content string
			delta := response.Choices[0].Delta

			if len(delta.ToolCalls) > 0 {
				content = delta.ToolCalls[0].Function.Arguments

				trimmed := strings.TrimSpace(content)
				if trimmed == "{%invalidjson%}" || trimmed == "``(no output)``````" {
					log.Println("File", filePath+":", "%invalidjson%} token in streamed chunk")
					fileState.retryOrError(fmt.Errorf("invalid JSON in streamed chunk for file '%s'", filePath))

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

				fileState.activeBuild.Buffer += content
				fileState.activeBuild.BufferTokens++

				// After a reasonable threshhold, if buffer has significantly more tokens than original file + proposed changes, something is wrong
				if fileState.activeBuild.BufferTokens > 500 && fileState.activeBuild.BufferTokens > int(float64(fileState.activeBuild.CurrentFileTokens+fileState.activeBuild.FileContentTokens)*1.5) {
					log.Printf("File %s: Stream buffer tokens too high\n", filePath)
					log.Printf("Current file tokens: %d\n", fileState.activeBuild.CurrentFileTokens)
					log.Printf("File content tokens: %d\n", fileState.activeBuild.FileContentTokens)
					log.Printf("Cutoff: %d\n", int(float64(fileState.activeBuild.CurrentFileTokens+fileState.activeBuild.FileContentTokens)*1.5))
					log.Printf("Buffer tokens: %d\n", fileState.activeBuild.BufferTokens)
					log.Println("Buffer:")
					log.Println(fileState.activeBuild.Buffer)

					fileState.retryOrError(fmt.Errorf("stream buffer tokens too high for file '%s'", filePath))
					return
				}
			}

			var streamed types.StreamedChanges
			err = json.Unmarshal([]byte(fileState.activeBuild.Buffer), &streamed)

			if err == nil {
				log.Printf("File %s: Parsed streamed replacements\n", filePath)
				// spew.Dump(streamed)

				planFileResult, allSucceeded := getPlanResult(
					planResultParams{
						orgId:           currentOrgId,
						planId:          planId,
						planBuildId:     build.Id,
						convoMessageId:  build.ConvoMessageId,
						filePath:        filePath,
						currentState:    currentState,
						fileContent:     activeBuild.FileContent,
						streamedChanges: streamed.Changes,
					},
				)

				if !allSucceeded {
					log.Println("Failed replacements:")
					for _, replacement := range planFileResult.Replacements {
						if replacement.Failed {
							spew.Dump(replacement)
						}
					}

					// no retry here as this should never happen
					fileState.onBuildFileError(fmt.Errorf("replacements failed for file '%s'", filePath))
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

				fileState.onFinishBuildFile(planFileResult)
				return
			} else if len(delta.ToolCalls) == 0 {
				log.Println("Stream chunk missing function call. Response:")
				log.Println(spew.Sdump(response))
				log.Println(spew.Sdump(fileState))

				fileState.retryOrError(fmt.Errorf("stream chunk missing function call. Reason: %s, File: %s", choice.FinishReason, filePath))
				return
			}
		}
	}

}

func (fileState *activeBuildStreamFileState) retryOrError(err error) {
	if fileState.numRetry < MaxBuildStreamErrorRetries {
		fileState.numRetry++
		fileState.activeBuild.Buffer = ""
		fileState.activeBuild.BufferTokens = 0
		log.Printf("Retrying build file '%s' due to error: %v\n", fileState.filePath, err)

		// Exponential backoff
		time.Sleep(time.Duration(fileState.numRetry*fileState.numRetry) * time.Second)

		fileState.buildFile()
	} else {
		fileState.onBuildFileError(err)
	}
}
