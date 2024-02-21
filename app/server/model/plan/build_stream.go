package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/model"
	"plandex-server/types"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

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
	buffer           string
	contextPart      *db.Context
	currentState     string
}

func (fileState *activeBuildStreamFileState) listenStream(stream *openai.ChatCompletionStream) {
	filePath := fileState.filePath
	build := fileState.build
	currentOrgId := fileState.currentOrgId
	planId := fileState.plan.Id
	branch := fileState.branch
	currentState := fileState.currentState
	contextPart := fileState.contextPart
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
			fileState.onBuildFileError(fmt.Errorf("stream timeout due to inactivity for file '%s'", filePath))
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

				fileState.onBuildFileError(fmt.Errorf("stream error for file '%s': %v", filePath, err))
				return
			}

			if len(response.Choices) == 0 {
				fileState.onBuildFileError(fmt.Errorf("stream error: no choices"))
				return
			}

			choice := response.Choices[0]
			var content string
			delta := response.Choices[0].Delta

			if len(delta.ToolCalls) == 0 {
				log.Println("Stream chunk missing function call. Response:")
				spew.Dump(response)

				fileState.onBuildFileError(fmt.Errorf("stream chunk missing function call. Reason: %s, File: %s", choice.FinishReason, filePath))
				return
			}

			content = delta.ToolCalls[0].Function.Arguments
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

			fileState.buffer += content

			var streamed types.StreamedReplacements
			err = json.Unmarshal([]byte(fileState.buffer), &streamed)
			if err == nil && len(streamed.Replacements) > 0 {
				log.Printf("File %s: Parsed streamed replacements\n", filePath)
				spew.Dump(streamed)

				planFileResult, allSucceeded := getPlanResult(
					planResultParams{
						orgId:                currentOrgId,
						planId:               planId,
						planBuildId:          build.Id,
						convoMessageId:       build.ConvoMessageId,
						filePath:             filePath,
						currentState:         currentState,
						context:              contextPart,
						fileContent:          activeBuild.FileContent,
						streamedReplacements: streamed.Replacements,
					},
				)

				if !allSucceeded {
					log.Println("Failed replacements:")
					for _, replacement := range planFileResult.Replacements {
						if replacement.Failed {
							spew.Dump(replacement)
						}
					}

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
			}
		}
	}

}
