package model

import (
	"context"
	"fmt"
	"io"
	"log"
	"plandex-server/types"
	shared "plandex-shared"
	"time"
)

type OnStreamFn func(chunk string, buffer string) (shouldStop bool)

func CreateChatCompletionWithInternalStream(
	clients map[string]ClientInfo,
	modelConfig *shared.ModelRoleConfig,
	ctx context.Context,
	req types.ExtendedChatCompletionRequest,
	onStream OnStreamFn,
	reqStarted time.Time,
) (*types.ModelResponse, error) {
	client, ok := clients[modelConfig.BaseModelConfig.ApiKeyEnvVar]
	if !ok {
		return nil, fmt.Errorf("client not found for api key env var: %s", modelConfig.BaseModelConfig.ApiKeyEnvVar)
	}

	resolveReq(&req, modelConfig)

	// choose the fastest provider by latency/throughput on openrouter
	if modelConfig.BaseModelConfig.Provider == shared.ModelProviderOpenRouter {
		req.Model += ":nitro"
	}

	// Force streaming mode since we're using the streaming API
	req.Stream = true

	return withStreamingRetries[types.ModelResponse](ctx, func() (*types.ModelResponse, error) {
		return processChatCompletionStream(modelConfig, client, modelConfig.BaseModelConfig.BaseUrl, ctx, req, onStream, reqStarted)
	})
}

func processChatCompletionStream(
	modelConfig *shared.ModelRoleConfig,
	client ClientInfo,
	baseUrl string,
	ctx context.Context,
	req types.ExtendedChatCompletionRequest,
	onStream OnStreamFn,
	reqStarted time.Time,
) (*types.ModelResponse, error) {
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := createChatCompletionStreamExtended(modelConfig, client, baseUrl, streamCtx, req)
	if err != nil {
		return nil, fmt.Errorf("error creating chat completion stream: %w", err)
	}
	defer stream.Close()

	accumulator := types.NewStreamCompletionAccumulator()
	// Create a timer that will trigger if no chunk is received within the specified duration
	timer := time.NewTimer(OPENAI_STREAM_CHUNK_TIMEOUT)
	defer timer.Stop()
	streamFinished := false

	receivedFirstChunk := false

	// Process stream until EOF or error
	for {
		select {
		case <-streamCtx.Done():
			log.Println("Stream canceled")
			return accumulator.Result(true, streamCtx.Err()), streamCtx.Err()
		case <-timer.C:
			log.Println("Stream timed out due to inactivity")
			if streamFinished {
				log.Println("Stream finished—timed out waiting for usage chunk")
				return accumulator.Result(false, nil), nil
			} else {
				log.Println("Stream timed out due to inactivity")
				return accumulator.Result(true, fmt.Errorf("stream timed out due to inactivity. The model is not responding.")), nil
			}
		default:
			response, err := stream.Recv()
			if err == io.EOF {
				if streamFinished {
					return accumulator.Result(false, nil), nil
				}

				err = fmt.Errorf("model stream ended unexpectedly: %w", err)
				return accumulator.Result(true, err), err
			}
			if err != nil {
				err = fmt.Errorf("error receiving stream chunk: %w", err)
				return accumulator.Result(true, err), err
			}

			if response.ID != "" {
				accumulator.SetGenerationId(response.ID)
			}

			if !receivedFirstChunk {
				receivedFirstChunk = true
				accumulator.SetFirstTokenAt(time.Now())
			}

			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(OPENAI_STREAM_CHUNK_TIMEOUT)

			// Process the response
			if response.Usage != nil {
				accumulator.SetUsage(response.Usage)
				return accumulator.Result(false, nil), nil
			}

			if len(response.Choices) == 0 {
				log.Println("No choices in response")
				err := fmt.Errorf("no choices in response")
				return accumulator.Result(false, err), err
			}

			if len(response.Choices) > 1 {
				err = fmt.Errorf("stream finished with more than one choice | The model failed to generate a valid response.")
				return accumulator.Result(true, err), err
			}

			choice := response.Choices[0]

			if choice.FinishReason != "" {
				if choice.FinishReason == "error" {
					err = fmt.Errorf("model stopped with error status | The model is not responding.")
					return accumulator.Result(true, err), err
				} else {
					// Reset the timer for the usage chunk
					if !timer.Stop() {
						<-timer.C
					}
					timer.Reset(OPENAI_USAGE_CHUNK_TIMEOUT)
					streamFinished = true
					continue
				}
			}

			var content string

			if req.Tools != nil {
				if choice.Delta.ToolCalls != nil {
					toolCall := choice.Delta.ToolCalls[0]
					content = toolCall.Function.Arguments
				}
			} else {
				if choice.Delta.Content != "" {
					content = choice.Delta.Content
				}
			}

			accumulator.AddContent(content)
			// pass the chunk and the accumulated content to the callback
			if onStream != nil {
				shouldReturn := onStream(content, accumulator.Content())
				if shouldReturn {
					return accumulator.Result(false, nil), nil
				}
			}
		}
	}
}

func withStreamingRetries[T any](
	ctx context.Context,
	operation func() (*types.ModelResponse, error),
) (*types.ModelResponse, error) {
	var result *types.ModelResponse
	var numRetry int

	for {
		if ctx.Err() != nil {
			if result != nil {
				// Return partial result with context error
				result.Stopped = true
				result.Error = ctx.Err().Error()
				return result, ctx.Err()
			}
			return nil, ctx.Err()
		}

		resp, err := operation()
		if err == nil {
			return resp, nil
		}

		// Store the partial result for potential return
		result = resp

		log.Printf("Error in streaming operation: %v, retry: %d\n", err, numRetry)

		if isNonRetriableErr(err) {
			return result, err
		}

		if numRetry >= OPENAI_MAX_RETRIES {
			log.Println("Max retries reached - no retry")
			return result, err
		}

		// Handle retry timing
		if duration := parseRetryAfter(err.Error()); duration != nil {
			log.Printf("Retry duration found: %v\n", *duration)
			waitDuration := time.Duration(float64(*duration) * OPENAI_BACKOFF_MULTIPLIER)

			if waitDuration > OPENAI_ABORT_WAIT_DURATION {
				return result, err
			} else if waitDuration > OPENAI_MAX_WAIT_DURATION {
				waitDuration = OPENAI_MAX_WAIT_DURATION
			}

			time.Sleep(waitDuration)
		} else {
			waitBackoff(numRetry)
		}

		numRetry++
	}
}
