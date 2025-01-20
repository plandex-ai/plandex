package plan

import (
	"fmt"
	"log"
	"plandex-server/model"
	"time"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) listenStream(stream *openai.ChatCompletionStream) {
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
				state.onError(fmt.Errorf("stream timeout due to inactivity | This usually means the model is not responding."), true, "", "")
				continue mainLoop
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
				return
			}

			if len(response.Choices) == 0 {
				if response.Usage != nil {
					state.handleUsageChunk(response.Usage)
					return
				}

				state.onError(fmt.Errorf("stream finished with no choices | This usually means the model failed to generate a valid response."), true, "", "")
				continue mainLoop
			}

			if len(response.Choices) > 1 {
				state.onError(fmt.Errorf("stream finished with more than one choice | This usually means the model failed to generate a valid response."), true, "", "")
				continue mainLoop
			}

			choice := response.Choices[0]

			processChunkRes := state.processChunk(choice)
			if processChunkRes.shouldReturn {
				return
			}

			if choice.FinishReason != "" {
				log.Println("Model stream finished")
				log.Println("Finish reason: ", choice.FinishReason)

				if choice.FinishReason == "error" {
					state.onError(fmt.Errorf("model stopped with error status | This usually means the model is not responding."), true, "", "")
					continue mainLoop
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
			}
			// let main loop continue
		}
	}
}
