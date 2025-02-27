package model

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"plandex-server/types"
	"regexp"
	"strings"
	"sync"
	"time"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

const (
	OPENAI_STREAM_CHUNK_TIMEOUT = time.Duration(30) * time.Second
	OPENAI_USAGE_CHUNK_TIMEOUT  = time.Duration(5) * time.Second
	OPENAI_MAX_RETRIES          = 3
	OPENAI_MAX_WAIT_DURATION    = 60 * time.Second
	OPENAI_ABORT_WAIT_DURATION  = 120 * time.Second
	OPENAI_BACKOFF_MULTIPLIER   = 3.0
)

var httpClient = &http.Client{}

type ClientInfo struct {
	Client   *openai.Client
	ApiKey   string
	OrgId    string
	Endpoint string
}

func InitClients(apiKeys map[string]string, endpointsByApiKeyEnvVar map[string]string, openAIEndpoint, orgId string) map[string]ClientInfo {
	clients := make(map[string]ClientInfo)
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

func newClient(apiKey, endpoint, orgId string) ClientInfo {
	config := openai.DefaultConfig(apiKey)
	if endpoint != "" {
		config.BaseURL = endpoint
	}
	if orgId != "" {
		config.OrgID = orgId
	}

	return ClientInfo{
		Client:   openai.NewClientWithConfig(config),
		ApiKey:   apiKey,
		OrgId:    orgId,
		Endpoint: endpoint,
	}
}

// ExtendedChatCompletionStream can wrap either a native OpenAI stream or our custom implementation
type ExtendedChatCompletionStream struct {
	openaiStream *openai.ChatCompletionStream
	customReader *StreamReader[types.ExtendedChatCompletionStreamResponse]
	ctx          context.Context
}

// StreamReader handles the SSE stream reading
type StreamReader[T any] struct {
	reader             *bufio.Reader
	response           *http.Response
	emptyMessagesLimit int
	errAccumulator     *ErrorAccumulator
	unmarshaler        *JSONUnmarshaler
}

// ErrorAccumulator keeps track of errors during streaming
type ErrorAccumulator struct {
	errors []error
	mu     sync.Mutex
}

// JSONUnmarshaler handles JSON unmarshaling for stream responses
type JSONUnmarshaler struct{}

func CreateChatCompletionStream(
	clients map[string]ClientInfo,
	modelConfig *shared.ModelRoleConfig,
	ctx context.Context,
	req types.ExtendedChatCompletionRequest,
) (*ExtendedChatCompletionStream, error) {
	client, ok := clients[modelConfig.BaseModelConfig.ApiKeyEnvVar]
	if !ok {
		return nil, fmt.Errorf("client not found for api key env var: %s", modelConfig.BaseModelConfig.ApiKeyEnvVar)
	}

	resolveReq(&req, modelConfig)

	// choose the fastest provider by latency/throughput on openrouter
	if modelConfig.BaseModelConfig.Provider == shared.ModelProviderOpenRouter {
		req.Model += ":nitro"
	}

	if modelConfig.BaseModelConfig.IncludeReasoning {
		req.IncludeReasoning = true
	}

	return withRetries(ctx, func() (*ExtendedChatCompletionStream, error) {
		return createChatCompletionStreamExtended(modelConfig, client, modelConfig.BaseModelConfig.BaseUrl, ctx, req)
	})
}

func CreateChatCompletion(
	clients map[string]ClientInfo,
	modelConfig *shared.ModelRoleConfig,
	ctx context.Context,
	req types.ExtendedChatCompletionRequest,
) (openai.ChatCompletionResponse, error) {
	client, ok := clients[modelConfig.BaseModelConfig.ApiKeyEnvVar]
	if !ok {
		return openai.ChatCompletionResponse{}, fmt.Errorf("client not found for api key env var: %s", modelConfig.BaseModelConfig.ApiKeyEnvVar)
	}

	resolveReq(&req, modelConfig)

	// choose the fastest provider by latency/throughput on openrouter
	if modelConfig.BaseModelConfig.Provider == shared.ModelProviderOpenRouter {
		req.Model += ":nitro"
	}

	return withRetries(ctx, func() (openai.ChatCompletionResponse, error) {
		return createChatCompletionExtended(modelConfig, client, modelConfig.BaseModelConfig.BaseUrl, ctx, req)
	})
}

func createChatCompletionExtended(
	modelConfig *shared.ModelRoleConfig,
	client ClientInfo,
	baseUrl string,
	ctx context.Context,
	extendedReq types.ExtendedChatCompletionRequest,
) (openai.ChatCompletionResponse, error) {
	var openaiReq *types.ExtendedOpenAIChatCompletionRequest
	if modelConfig.BaseModelConfig.Provider == shared.ModelProviderOpenAI {
		log.Println("Creating chat completion with direct OpenAI provider request")
		openaiReq = extendedReq.ToOpenAI()
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseUrl+"/chat/completions", nil)
	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+client.ApiKey)
	if client.OrgId != "" {
		req.Header.Set("OpenAI-Organization", client.OrgId)
	}

	addOpenRouterHeaders(req)

	// Add body
	var jsonBody []byte
	if openaiReq != nil {
		jsonBody, err = json.Marshal(openaiReq)
	} else {
		jsonBody, err = json.Marshal(extendedReq)
	}
	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}
	req.Body = io.NopCloser(bytes.NewReader(jsonBody))

	// Make request
	resp, err := httpClient.Do(req)
	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}
	defer resp.Body.Close()

	// log the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}

	var response openai.ChatCompletionResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}

	return response, nil
}

func createChatCompletionStreamExtended(
	modelConfig *shared.ModelRoleConfig,
	client ClientInfo,
	baseUrl string,
	ctx context.Context,
	extendedReq types.ExtendedChatCompletionRequest,
) (*ExtendedChatCompletionStream, error) {
	var openaiReq *types.ExtendedOpenAIChatCompletionRequest
	if modelConfig.BaseModelConfig.Provider == shared.ModelProviderOpenAI {
		openaiReq = extendedReq.ToOpenAI()
		log.Println("Creating chat completion stream with direct OpenAI provider request")
	}

	// Marshal the request body to JSON
	var jsonBody []byte
	var err error
	if openaiReq != nil {
		jsonBody, err = json.Marshal(openaiReq)
	} else {
		jsonBody, err = json.Marshal(extendedReq)
	}
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// log.Println("request jsonBody", string(jsonBody))

	// Create new request
	req, err := http.NewRequestWithContext(ctx, "POST", baseUrl+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set required headers for streaming
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Authorization", "Bearer "+client.ApiKey)
	if client.OrgId != "" {
		req.Header.Set("OpenAI-Organization", client.OrgId)
	}

	addOpenRouterHeaders(req)

	// Send the request
	resp, err := httpClient.Do(req) //nolint:bodyclose // body is closed in stream.Close()
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading error response: %w", err)
		}
		return nil, fmt.Errorf("streaming request failed: status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Log response headers
	// log.Println("Response headers:")
	// for key, values := range resp.Header {
	// 	log.Printf("%s: %v\n", key, values)
	// }

	reader := &StreamReader[types.ExtendedChatCompletionStreamResponse]{
		reader:             bufio.NewReader(resp.Body),
		response:           resp,
		emptyMessagesLimit: 30,
		errAccumulator:     NewErrorAccumulator(),
		unmarshaler:        &JSONUnmarshaler{},
	}

	return &ExtendedChatCompletionStream{
		customReader: reader,
		ctx:          ctx,
	}, nil
}

func NewErrorAccumulator() *ErrorAccumulator {
	return &ErrorAccumulator{
		errors: make([]error, 0),
	}
}

func (ea *ErrorAccumulator) Add(err error) {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	ea.errors = append(ea.errors, err)
}

func (ea *ErrorAccumulator) GetErrors() []error {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	return ea.errors
}

func (ju *JSONUnmarshaler) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// Recv reads from the stream
func (stream *StreamReader[T]) Recv() (*T, error) {
	for {
		line, err := stream.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		// Trim any whitespace
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for data prefix
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		// Extract the data
		data := strings.TrimPrefix(line, "data: ")

		// Check for stream completion
		if data == "[DONE]" {
			return nil, io.EOF
		}

		// Parse the response
		var response T
		err = stream.unmarshaler.Unmarshal([]byte(data), &response)
		if err != nil {
			stream.errAccumulator.Add(err)
			continue
		}

		return &response, nil
	}
}

func (stream *StreamReader[T]) Close() error {
	if stream.response != nil {
		return stream.response.Body.Close()
	}
	return nil
}

// Recv returns the next message in the stream
func (stream *ExtendedChatCompletionStream) Recv() (*types.ExtendedChatCompletionStreamResponse, error) {
	select {
	case <-stream.ctx.Done():
		return nil, stream.ctx.Err()
	default:
		if stream.openaiStream != nil {
			bytes, err := stream.openaiStream.RecvRaw()
			if err != nil {
				return nil, err
			}

			var response types.ExtendedChatCompletionStreamResponse
			err = json.Unmarshal(bytes, &response)
			if err != nil {
				return nil, err
			}
			return &response, nil
		}
		return stream.customReader.Recv()
	}
}

// Close the response body
func (stream *ExtendedChatCompletionStream) Close() error {
	if stream.openaiStream != nil {
		return stream.openaiStream.Close()
	}
	return stream.customReader.Close()
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

func resolveReq(req *types.ExtendedChatCompletionRequest, modelConfig *shared.ModelRoleConfig) {
	// if system prompt is disabled, change the role of the system message to user
	if modelConfig.BaseModelConfig.SystemPromptDisabled {
		log.Println("System prompt disabled - changing role of system message to user")
		for i, msg := range req.Messages {
			log.Println("Message role:", msg.Role)
			if msg.Role == openai.ChatMessageRoleSystem {
				log.Println("Changing role of system message to user")
				req.Messages[i].Role = openai.ChatMessageRoleUser
			}
		}

		for _, msg := range req.Messages {
			log.Println("Final message role:", msg.Role)
		}
	}

	if modelConfig.BaseModelConfig.RoleParamsDisabled {
		log.Println("Role params disabled - setting temperature and top p to 1")
		req.Temperature = 1
		req.TopP = 1
	}
}

// route directly to first-party providers on openrouter for the main models
// seems to be much faster this way currently
// func getOpenRouterProviderOrder(modelConfig *shared.ModelRoleConfig) []string {
// 	var providerOrder []string
// 	if modelConfig.BaseModelConfig.Provider == shared.ModelProviderOpenRouter {
// 		if len(modelConfig.BaseModelConfig.PreferredOpenRouterProviders) > 0 {
// 			for _, provider := range modelConfig.BaseModelConfig.PreferredOpenRouterProviders {
// 				providerOrder = append(providerOrder, string(provider))
// 			}
// 		}
// 	}
// 	return providerOrder
// }

func withRetries[T any](
	ctx context.Context,
	operation func() (T, error),
) (T, error) {
	var result T
	var numRetry int

	for {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		resp, err := operation()
		if err == nil {
			return resp, nil
		}

		log.Printf("Error in operation: %v, retry: %d\n", err, numRetry)

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

func addOpenRouterHeaders(req *http.Request) {
	req.Header.Set("HTTP-Referer", "https://plandex.ai")
	req.Header.Set("X-Title", "Plandex")
	req.Header.Set("X-OR-Prefer", "ttft,throughput")
	if os.Getenv("GOENV") == "production" {
		req.Header.Set("X-OR-Region", "us-east-1")
	}
}
