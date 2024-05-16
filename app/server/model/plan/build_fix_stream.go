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
	"sort"
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

			var streamed types.StreamedChangesWithLineNums
			err = json.Unmarshal([]byte(fileState.activeBuild.FixBuffer), &streamed)

			if err == nil {
				log.Printf("listenStreamFixChanges - File %s: Parsed streamed replacements\n", filePath)
				// spew.Dump(streamed)

				sorted := []*shared.StreamedChangeWithLineNums{}

				// Sort the streamed changes by start line
				for _, change := range streamed.Changes {
					if change.HasChange {
						sorted = append(sorted, change)
					}
				}

				// Sort the streamed changes by start line
				sort.Slice(sorted, func(i, j int) bool {
					var iStartLine int
					var jStartLine int

					// Convert the line number part to an integer
					iStartLine, _, err := sorted[i].GetLines()

					if err != nil {
						log.Printf("listenStream - Error getting start line for change %v: %v\n", sorted[i], err)
						fileState.lineNumsRetryOrError(fmt.Errorf("listenStream - error getting start line for change %v: %v", sorted[i], err))
						return false
					}

					jStartLine, _, err = sorted[j].GetLines()

					if err != nil {
						log.Printf("listenStream - Error getting start line for change %v: %v\n", sorted[j], err)
						fileState.lineNumsRetryOrError(fmt.Errorf("listenStream - error getting start line for change %v: %v", sorted[j], err))
						return false
					}

					return iStartLine < jStartLine
				})

				fileState.streamedChangesWithLineNums = sorted

				var overlapStrategy OverlapStrategy = OverlapStrategyError
				if fileState.fixFileNumRetry > 1 {
					overlapStrategy = OverlapStrategySkip
				}

				planFileResult, updated, allSucceeded, err := GetPlanResult(
					activePlan.Ctx,
					PlanResultParams{
						OrgId:                       currentOrgId,
						PlanId:                      planId,
						PlanBuildId:                 build.Id,
						ConvoMessageId:              build.ConvoMessageId,
						FilePath:                    filePath,
						PreBuildState:               fileState.updated,
						StreamedChangesWithLineNums: streamed.Changes,
						OverlapStrategy:             overlapStrategy,

						IsFix:       true,
						IsSyntaxFix: fileState.isFixingSyntax,
						IsOtherFix:  fileState.isFixingOther,

						FixEpoch: fileState.syntaxNumEpoch,

						CheckSyntax: true,
					},
				)

				if err != nil {
					log.Println("listenStreamFixChanges - Error getting plan result:", err)
					fileState.fixRetryOrAbort(fmt.Errorf("listenStreamFixChanges - error getting plan result for file '%s': %v", filePath, err))
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
				// spew.Dump(planFileResult)

				// reset fix state
				fileState.isFixingSyntax = false
				fileState.isFixingOther = false

				// if we are below the number of FixSyntaxRetries, and the syntax is invalid, short-circuit here and retry
				// otherwise if the syntax is invalid but we're out of retries, continue to onFinishBuildFile, which will handle epoch-based retries (i.e. running additional fixes on top of this failed fix) if applicable
				if planFileResult.WillCheckSyntax && !planFileResult.SyntaxValid {
					if fileState.syntaxNumRetry < FixSyntaxRetries {
						fileState.isFixingSyntax = true
						fileState.syntaxNumRetry++
						go fileState.fixFileLineNums()
						return
					}
				}

				fileState.onFinishBuildFile(planFileResult, updated)
				return
			} else if len(delta.ToolCalls) == 0 {
				log.Println("listenStreamFixChanges - Stream chunk missing function call.")

				fileState.fixRetryOrAbort(fmt.Errorf("listenStreamFixChanges - stream chunk missing function call. File: %s", filePath))
				return
			}
		}
	}
}

func (fileState *activeBuildStreamFileState) fixRetryOrAbort(err error) {
	if fileState.fixFileNumRetry < MaxBuildStreamErrorRetries {
		fileState.fixFileNumRetry++
		fileState.activeBuild.FixBuffer = ""
		fileState.activeBuild.FixBufferTokens = 0
		log.Printf("Retrying fix file '%s' due to error: %v\n", fileState.filePath, err)

		// Exponential backoff
		time.Sleep(time.Duration((fileState.verifyFileNumRetry*fileState.verifyFileNumRetry)/2)*200*time.Millisecond + time.Duration(rand.Intn(500))*time.Millisecond)

		fileState.fixFileLineNums()
	} else {
		log.Printf("Aborting fix file '%s' due to error: %v\n", fileState.filePath, err)

		fileState.onFinishBuildFile(nil, "")
	}
}
