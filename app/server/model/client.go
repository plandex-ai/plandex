package model

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

const OPENAI_STREAM_CHUNK_TIMEOUT = time.Duration(30) * time.Second

func NewClient(apiKey string) *openai.Client {
	config := openai.DefaultConfig(apiKey)
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
