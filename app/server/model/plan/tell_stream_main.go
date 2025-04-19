package plan

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"plandex-server/model"
	"plandex-server/notify"
	"runtime/debug"
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

	defer func() {
		if r := recover(); r != nil {
			log.Printf("listenStream: Panic: %v\n%s\n", r, string(debug.Stack()))

			go notify.NotifyErr(notify.SeverityError, fmt.Errorf("listenStream: Panic: %v\n%s", r, string(debug.Stack())))

			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Panic in listenStream",
			}
		}
	}()

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

	modelProvider := state.modelConfig.BaseModelConfig.Provider
	modelName := state.modelConfig.BaseModelConfig.ModelName

	log.Println("modelProvider", modelProvider)
	log.Println("modelName", modelName)

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
				state.execHookOnStop(false)
				return
			} else {
				res := state.onError(onErrorParams{
					streamErr: fmt.Errorf("stream timeout due to inactivity: The underlying LLM is not responding"),
					storeDesc: true,
					canRetry:  active.CurrentReplyContent == "", // if there was no output yet, we can retry
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

				var msg string
				if active.CurrentReplyContent == "" {
					msg = fmt.Sprintf("The AI model (%s/%s) didn't respond: %v", modelProvider, modelName, err)
				} else {
					msg = fmt.Sprintf("The AI model (%s/%s) stopped responding: %v", modelProvider, modelName, err)
				}
				state.onError(onErrorParams{
					streamErr: errors.New(msg),
					storeDesc: true,
					canRetry:  active.CurrentReplyContent == "", // if there was no output yet, we can retry
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

			if response.Error != nil {
				log.Println("listenStream - stream finished with error", spew.Sdump(response.Error))
				res := state.onError(onErrorParams{
					streamErr: fmt.Errorf("The AI model (%s/%s) stopped streaming with error code %d: %s", modelProvider, modelName, response.Error.Code, response.Error.Message),
					storeDesc: true,
					canRetry:  active.CurrentReplyContent == "" && !(response.Error.Code >= 400 && response.Error.Code < 500),
				})
				if res.shouldReturn {
					return
				}
				if res.shouldContinueMainLoop {
					continue mainLoop
				}
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
					log.Println("Model stream finished with error")

					res := state.onError(onErrorParams{
						streamErr: fmt.Errorf("The AI model (%s/%s) stopped streaming with an error status", modelProvider, modelName),
						storeDesc: true,
						canRetry:  active.CurrentReplyContent == "",
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
