package model

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

const OPENAI_STREAM_CHUNK_TIMEOUT = time.Duration(30) * time.Second
const OPENAI_USAGE_CHUNK_TIMEOUT = time.Duration(5) * time.Second

func InitClients(apiKeys map[string]string, endpointsByApiKeyEnvVar map[string]string, openAIEndpoint, orgId string) map[string]*openai.Client {
	clients := make(map[string]*openai.Client)
	for key, apiKey := range apiKeys {
		var clientEndpoint string
		var clientOrgId string
		if key == "OPENAI_API_KEY" {
			clientEndpoint = openAIEndpoint
			clientOrgId = orgId
		} else {
			clientEndpoint = endpointsByApiKeyEnvVar[key]
		}
		clients[key] = newClient(apiKey, clientEndpoint, clientOrgId)
	}
	return clients
}

func newClient(apiKey, endpoint, orgId string) *openai.Client {
	config := openai.DefaultConfig(apiKey)
	if endpoint != "" {
		config.BaseURL = endpoint
	}
	if orgId != "" {
		config.OrgID = orgId
	}

	return openai.NewClientWithConfig(config)
}

func CreateChatCompletionStreamWithRetries(
	client *openai.Client,
	ctx context.Context,
	req openai.ChatCompletionRequest,
) (*openai.ChatCompletionStream, error) {
	return createChatCompletionStream(client, ctx, req, 0)
}

func createChatCompletionStream(
	client *openai.Client,
	ctx context.Context,
	req openai.ChatCompletionRequest,
	numRetry int,
) (*openai.ChatCompletionStream, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	stream, err := client.CreateChatCompletionStream(ctx, req)

	if err != nil {
		log.Printf("Error creating chat completion stream: %v, retry: %d\n", err, numRetry)

		if isNonRetriableErr(err) {
			return nil, err
		}

		// for retriable errors, retry with exponential backoff
		if numRetry < 5 {
			// check if the error message contains a retry duration
			if duration := parseRetryAfter(err.Error()); duration != nil {
				// wait for the duration times 3 to give some buffer
				waitDuration := time.Duration(float64(*duration) * 3)

				// ensure wait duration is 60 seconds or less - for really long retries just error out
				if waitDuration > 120*time.Second {
					return nil, err
				} else if waitDuration > 60*time.Second {
					waitDuration = 60 * time.Second
				}
				time.Sleep(waitDuration)
				return createChatCompletionStream(client, ctx, req, numRetry+1)
			}

			waitBackoff(numRetry)
			return createChatCompletionStream(client, ctx, req, numRetry+1)
		}

		log.Println("Max retries reached - no retry")
		return nil, err
	}

	return stream, nil
}

func CreateChatCompletionWithRetries(
	client *openai.Client,
	ctx context.Context,
	req openai.ChatCompletionRequest,
) (openai.ChatCompletionResponse, error) {
	return createChatCompletion(client, ctx, req, 0)
}

func createChatCompletion(
	client *openai.Client,
	ctx context.Context,
	req openai.ChatCompletionRequest,
	numRetry int,
) (openai.ChatCompletionResponse, error) {

	if ctx.Err() != nil {
		return openai.ChatCompletionResponse{}, ctx.Err()
	}

	resp, err := client.CreateChatCompletion(ctx, req)

	if err != nil {
		log.Printf("Error creating chat completion: %v, retry: %d\n", err, numRetry)

		if isNonRetriableErr(err) {
			return openai.ChatCompletionResponse{}, err
		}

		// for retriable errors, retry with exponential backoff
		if numRetry < 5 {
			// check if the error message contains a retry duration
			if duration := parseRetryAfter(err.Error()); duration != nil {
				log.Printf("Retry duration found: %v\n", *duration)

				// wait for the duration times 3 to give some buffer
				waitDuration := time.Duration(float64(*duration) * 3)

				// ensure wait duration is 60 seconds or less - for really long retries just error out
				if waitDuration > 120*time.Second {
					return openai.ChatCompletionResponse{}, err
				} else if waitDuration > 60*time.Second {
					waitDuration = 60 * time.Second
				}

				time.Sleep(waitDuration)
				return createChatCompletion(client, ctx, req, numRetry+1)
			}

			waitBackoff(numRetry)
			return createChatCompletion(client, ctx, req, numRetry+1)
		}

		log.Println("Max retries reached - no retry")
		return openai.ChatCompletionResponse{}, err
	}

	return resp, nil
}

func isNonRetriableErr(err error) bool {
	errStr := err.Error()

	// we don't want to retry on the errors below
	if strings.Contains(errStr, "context deadline exceeded") || strings.Contains(errStr, "context canceled") {
		log.Println("Context deadline exceeded or canceled - no retry")
		return true
	}

	if strings.Contains(errStr, "status code: 400") &&
		strings.Contains(errStr, "reduce the length of the messages") {
		log.Println("Token limit exceeded - no retry")
		return true
	}

	if strings.Contains(errStr, "status code: 401") {
		log.Println("Invalid auth or api key - no retry")
		return true
	}

	if strings.Contains(errStr, "status code: 429") && strings.Contains(errStr, "exceeded your current quota") {
		log.Println("Current quota exceeded - no retry")
		return true
	}

	return false
}

func waitBackoff(numRetry int) {
	d := time.Duration(1<<uint(numRetry)) * time.Second
	log.Printf("Retrying in %v\n", d)
	time.Sleep(d)
}

// parseRetryAfter takes an error message and returns the retry duration or nil if no duration is found.
func parseRetryAfter(errorMessage string) *time.Duration {
	// Regex pattern to find the duration in seconds or milliseconds
	pattern := regexp.MustCompile(`try again in (\d+(\.\d+)?(ms|s))`)
	match := pattern.FindStringSubmatch(errorMessage)
	if len(match) > 1 {
		durationStr := match[1] // the duration string including the unit
		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			fmt.Println("Error parsing duration:", err)
			return nil
		}
		return &duration
	}
	return nil
}
