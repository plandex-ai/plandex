package model

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"plandex-server/types"
	shared "plandex-shared"
	"time"
)

type OnStreamFn func(chunk string, buffer string) (shouldStop bool)

func CreateChatCompletionWithInternalStream(
	clients map[string]ClientInfo,
	authVars map[string]string,
	modelConfig *shared.ModelRoleConfig,
	ctx context.Context,
	req types.ExtendedChatCompletionRequest,
	onStream OnStreamFn,
	reqStarted time.Time,
) (*types.ModelResponse, error) {
	providerComposite := modelConfig.GetProviderComposite(authVars)
	_, ok := clients[providerComposite]
	if !ok {
		return nil, fmt.Errorf("client not found for provider composite: %s", providerComposite)
	}

	baseModelConfig := modelConfig.GetBaseModelConfig(authVars)

	resolveReq(&req, modelConfig)

	// choose the fastest provider by latency/throughput on openrouter
	if baseModelConfig.Provider == shared.ModelProviderOpenRouter {
		req.Model += ":nitro"
	}

	// Force streaming mode since we're using the streaming API
	req.Stream = true

	return withStreamingRetries(ctx, func(numTotalRetry int, modelErr *shared.ModelError) (resp *types.ModelResponse, fallbackRes shared.FallbackResult, err error) {
		fallbackRes = modelConfig.GetFallbackForModelError(numTotalRetry, modelErr)
		resolvedModelConfig := fallbackRes.ModelRoleConfig

		if resolvedModelConfig == nil {
			return nil, fallbackRes, fmt.Errorf("model config is nil")
		}

		providerComposite := resolvedModelConfig.GetProviderComposite(authVars)
		opClient, ok := clients[providerComposite]

		if !ok {
			return nil, fallbackRes, fmt.Errorf("client not found for provider composite: %s", providerComposite)
		}

		modelConfig = resolvedModelConfig
		resp, err = processChatCompletionStream(resolvedModelConfig, opClient, authVars, ctx, req, onStream, reqStarted)
		if err != nil {
			return nil, fallbackRes, err
		}
		return resp, fallbackRes, nil
	}, func(resp *types.ModelResponse, err error) {
		if resp != nil {
			resp.Stopped = true
			resp.Error = err.Error()
		}
	})
}

func processChatCompletionStream(
	modelConfig *shared.ModelRoleConfig,
	client ClientInfo,
	authVars map[string]string,
	ctx context.Context,
	req types.ExtendedChatCompletionRequest,
	onStream OnStreamFn,
	reqStarted time.Time,
) (*types.ModelResponse, error) {
	streamCtx, cancel := context.WithCancel(ctx)

	stream, err := createChatCompletionStreamExtended(modelConfig, client, authVars, streamCtx, req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("error creating chat completion stream: %w", err)
	}

	defer stream.Close()
	defer cancel()

	accumulator := types.NewStreamCompletionAccumulator()
	// Create a timer that will trigger if no chunk is received within the specified duration
	timer := time.NewTimer(ACTIVE_STREAM_CHUNK_TIMEOUT)
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
			timer.Reset(ACTIVE_STREAM_CHUNK_TIMEOUT)

			// Process the response
			if response.Usage != nil {
				accumulator.SetUsage(response.Usage)
				return accumulator.Result(false, nil), nil
			}

			emptyChoices := false
			var content string

			if len(response.Choices) == 0 {
				// Previously we'd return an error if there were no choices, but some models do this and then keep streaming, so we'll just log it and continue
				log.Println("processChatCompletionStream - no choices in response")
				// err := fmt.Errorf("no choices in response")
				// return accumulator.Result(false, err), err
				emptyChoices = true
			}

			// We'll be more accepting of multiple choices and just take the first one
			// if len(response.Choices) > 1 {
			// 	err = fmt.Errorf("stream finished with more than one choice | The model failed to generate a valid response.")
			// 	return accumulator.Result(true, err), err
			// }

			if !emptyChoices {
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
						timer.Reset(USAGE_CHUNK_TIMEOUT)
						streamFinished = true
						continue
					}
				}

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
	operation func(numRetry int, modelErr *shared.ModelError) (resp *T, fallbackRes shared.FallbackResult, err error),
	onContextDone func(resp *T, err error),
) (*T, error) {
	var resp *T
	var numTotalRetry int
	var numFallbackRetry int
	var fallbackRes shared.FallbackResult
	var modelErr *shared.ModelError

	for {
		if ctx.Err() != nil {
			if resp != nil {
				// Return partial result with context error
				onContextDone(resp, ctx.Err())
				return resp, ctx.Err()
			}
			return nil, ctx.Err()
		}

		var err error

		var numRetry int
		if numFallbackRetry > 0 {
			numRetry = numFallbackRetry
		} else {
			numRetry = numTotalRetry
		}

		log.Printf("withStreamingRetries - will run operation")

		resp, fallbackRes, err = operation(numTotalRetry, modelErr)
		if err == nil {
			return resp, nil
		}

		log.Printf("withStreamingRetries - operation returned error: %v", err)

		isFallback := fallbackRes.IsFallback
		maxRetries := MAX_RETRIES_WITHOUT_FALLBACK
		if isFallback {
			maxRetries = MAX_ADDITIONAL_RETRIES_WITH_FALLBACK
		}

		compareRetries := numTotalRetry
		if isFallback {
			compareRetries = numFallbackRetry
		}

		log.Printf("Error in streaming operation: %v, isFallback: %t, numTotalRetry: %d, numFallbackRetry: %d, numRetry: %d, compareRetries: %d, maxRetries: %d\n", err, isFallback, numTotalRetry, numFallbackRetry, numRetry, compareRetries, maxRetries)

		classifyRes := classifyBasicError(err)
		modelErr = &classifyRes

		newFallback := false
		if !modelErr.Retriable {
			log.Printf("withStreamingRetries - operation returned non-retriable error: %v", err)
			if modelErr.Kind == shared.ErrContextTooLong && fallbackRes.ModelRoleConfig.LargeContextFallback == nil {
				log.Printf("withStreamingRetries - non-retriable context too long error and no large context fallback is defined, returning error")
				// if it's a context too long error and no large context fallback is defined, return the error
				return resp, err
			} else if modelErr.Kind != shared.ErrContextTooLong && fallbackRes.ModelRoleConfig.ErrorFallback == nil {
				log.Printf("withStreamingRetries - non-retriable error and no error fallback is defined, returning error")
				// if it's any other error and no error fallback is defined, return the error
				return resp, err
			}
			log.Printf("withStreamingRetries - operation returned non-retriable error, but has fallback - resetting numFallbackRetry to 0 and continuing to retry")
			numFallbackRetry = 0
			newFallback = true
			// otherwise, continue to retry logic
		}

		if compareRetries >= maxRetries {
			log.Printf("withStreamingRetries - compareRetries >= maxRetries - returning error")
			return resp, err
		}

		var retryDelay time.Duration
		if modelErr != nil && modelErr.RetryAfterSeconds > 0 {
			// if the model err has a retry after, then use that with a bit of padding
			retryDelay = time.Duration(int(float64(modelErr.RetryAfterSeconds)*1.1)) * time.Second
		} else {
			// otherwise, use some jitter
			retryDelay = time.Duration(1000+rand.Intn(200)) * time.Millisecond
		}

		log.Printf("withStreamingRetries - retrying stream in %v seconds", retryDelay)
		time.Sleep(retryDelay)

		numTotalRetry++
		if isFallback && !newFallback {
			numFallbackRetry++
		}
	}
}
