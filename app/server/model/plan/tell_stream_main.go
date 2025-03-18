package plan

import (
	"fmt"
	"log"
	"plandex-server/model"
	"time"

	shared "plandex-shared"

	"github.com/davecgh/go-spew/spew"
)

func (state *activeTellStreamState) listenStream(stream *model.ExtendedChatCompletionStream) {
	defer stream.Close()

	plan := state.plan
	planId := plan.Id
	branch := state.branch

	active := GetActivePlan(planId, branch)

	if active == nil {
		log.Printf("listenStream - Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return
	}

	state.chunkProcessor = &chunkProcessor{
		replyOperations:                 []*shared.Operation{},
		chunksReceived:                  0,
		maybeRedundantOpeningTagContent: "",
		fileOpen:                        false,
		contentBuffer:                   "",
		awaitingBlockOpeningTag:         false,
		awaitingBlockClosingTag:         false,
		awaitingBackticks:               false,
	}

	// Create a timer that will trigger if no chunk is received within the specified duration
	timer := time.NewTimer(model.OPENAI_STREAM_CHUNK_TIMEOUT)
	defer timer.Stop()
	streamFinished := false

mainLoop:
	for {
		select {
		case <-active.Ctx.Done():
			// The main modelContext was canceled (not the timer)
			log.Println("\nTell: stream canceled")
			state.execHookOnStop(false)
			return
		case <-timer.C:
			// Timer triggered because no new chunk was received in time
			log.Println("\nTell: stream timeout due to inactivity")
			if streamFinished {
				log.Println("Tell stream finished—timed out waiting for usage chunk")
				state.execHookOnStop(true)
				return
			} else {
				res := state.onError(onErrorParams{
					streamErr: fmt.Errorf("stream timeout due to inactivity | The model is not responding."),
					storeDesc: true,
					canRetry:  true,
				})
				if res.shouldReturn {
					return
				}
				if res.shouldContinueMainLoop {
					continue mainLoop
				}
			}

		default:
			response, err := stream.Recv()

			if err == nil {
				// Successfully received a chunk, reset the timer
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(model.OPENAI_STREAM_CHUNK_TIMEOUT)
			} else {
				if err.Error() == "context canceled" {
					log.Println("Tell: stream context canceled")
					state.execHookOnStop(false)
					return
				}

				log.Printf("Tell: error receiving stream chunk: %v\n", err)
				state.execHookOnStop(true)

				state.onError(onErrorParams{
					streamErr: fmt.Errorf("error receiving stream chunk: %v", err),
					storeDesc: true,
					canRetry:  true,
				})
				// here we want to return no matter what -- state.onError will decide whether to retry or not
				return
			}

			// log.Println("tell stream main: received stream response", spew.Sdump(response))

			if response.ID != "" && state.generationId == "" {
				state.generationId = response.ID
			}

			if state.firstTokenAt.IsZero() {
				state.firstTokenAt = time.Now()
			}

			if len(response.Choices) == 0 {
				if response.Usage != nil {
					state.handleUsageChunk(response.Usage)
					return
				}

				log.Println("listenStream - stream finished with no choices", spew.Sdump(response))

				// Previously we'd return an error if there were no choices, but some models do this and then keep streaming, so we'll just log it and continue, waiting for an EOF if there's a problem
				// res := state.onError(onErrorParams{
				// 	streamErr: fmt.Errorf("stream finished with no choices | The model failed to generate a valid response."),
				// 	storeDesc: true,
				// 	canRetry:  true,
				// })
				// if res.shouldReturn {
				// 	return
				// }
				// if res.shouldContinueMainLoop {
				// 	// continue instead of returning so that context cancellation is handled
				// 	continue mainLoop
				// }

				continue mainLoop
			}

			// We'll be more accepting of multiple choices and just take the first one
			// if len(response.Choices) > 1 {
			// 	res := state.onError(onErrorParams{
			// 		streamErr: fmt.Errorf("stream finished with more than one choice | The model failed to generate a valid response."),
			// 		storeDesc: true,
			// 		canRetry:  true,
			// 	})
			// 	if res.shouldReturn {
			// 		return
			// 	}
			// 	if res.shouldContinueMainLoop {
			// 		// continue instead of returning so that context cancellation is handled
			// 		continue mainLoop
			// 	}
			// }

			choice := response.Choices[0]

			processChunkRes := state.processChunk(choice)
			if processChunkRes.shouldReturn {
				return
			}

			if choice.FinishReason != "" {
				log.Println("Model stream finished")
				log.Println("Finish reason: ", choice.FinishReason)

				if choice.FinishReason == "error" {
					res := state.onError(onErrorParams{
						streamErr: fmt.Errorf("model stopped with error status | The model is not responding."),
						storeDesc: true,
						canRetry:  true,
					})
					if res.shouldReturn {
						return
					}
					if res.shouldContinueMainLoop {
						continue mainLoop
					}
				}

				streamFinishResult := state.handleStreamFinished()
				if streamFinishResult.shouldContinueMainLoop {
					continue mainLoop
				}
				if streamFinishResult.shouldReturn {
					return
				}

				// usage can either be included in the final chunk (openrouter) or in a separate chunk (openai)
				// if the usage chunk is included, handle it and then return out of listener
				// otherwise keep listening for the usage chunk
				if response.Usage != nil {
					state.handleUsageChunk(response.Usage)
					return
				}

				// Reset the timer for the usage chunk
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(model.OPENAI_USAGE_CHUNK_TIMEOUT)
				streamFinished = true
				continue
			} else if response.Usage != nil {
				state.handleUsageChunk(response.Usage)
				return
			}
			// let main loop continue
		}
	}
}
