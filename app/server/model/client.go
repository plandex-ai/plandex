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
	"strings"
	"sync"
	"time"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

// note that we are *only* using streaming requests now
// non-streaming request handling has been removed completely
// streams offer more predictable cancellation partial results

const (
	ACTIVE_STREAM_CHUNK_TIMEOUT          = time.Duration(60) * time.Second
	USAGE_CHUNK_TIMEOUT                  = time.Duration(10) * time.Second
	MAX_ADDITIONAL_RETRIES_WITH_FALLBACK = 1
	MAX_RETRIES_WITHOUT_FALLBACK         = 2
	MAX_RETRY_DELAY_SECONDS              = 10
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
	_, ok := clients[modelConfig.BaseModelConfig.ApiKeyEnvVar]
	if !ok {
		fmt.Printf("client not found for api key env var: %s", modelConfig.BaseModelConfig.ApiKeyEnvVar)
		if modelConfig.MissingKeyFallback != nil {
			fmt.Println("using missing key fallback")
			return CreateChatCompletionStream(clients, modelConfig.MissingKeyFallback, ctx, req)
		}
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

	return withStreamingRetries(ctx, func(numTotalRetry int, modelErr *shared.ModelError, stripCacheControl bool) (*ExtendedChatCompletionStream, shared.FallbackResult, error) {
		fallbackRes := modelConfig.GetFallbackForModelError(numTotalRetry, modelErr)
		resolvedModelConfig := fallbackRes.ModelRoleConfig

		if resolvedModelConfig == nil {
			return nil, fallbackRes, fmt.Errorf("model config is nil")
		}

		opClient, ok := clients[resolvedModelConfig.BaseModelConfig.ApiKeyEnvVar]

		if !ok {
			if resolvedModelConfig.MissingKeyFallback != nil {
				fmt.Println("using missing key fallback")
				resolvedModelConfig = resolvedModelConfig.MissingKeyFallback
				opClient, ok = clients[resolvedModelConfig.BaseModelConfig.ApiKeyEnvVar]
				if !ok {
					return nil, fallbackRes, fmt.Errorf("client not found for api key env var: %s", resolvedModelConfig.BaseModelConfig.ApiKeyEnvVar)
				}
			} else {
				return nil, fallbackRes, fmt.Errorf("client not found for api key env var: %s", resolvedModelConfig.BaseModelConfig.ApiKeyEnvVar)
			}
		}

		if stripCacheControl {
			for i := range req.Messages {
				for j := range req.Messages[i].Content {
					if req.Messages[i].Content[j].CacheControl != nil {
						req.Messages[i].Content[j].CacheControl = nil
					}
				}
			}
		}

		modelConfig = resolvedModelConfig
		resp, err := createChatCompletionStreamExtended(resolvedModelConfig, opClient, resolvedModelConfig.BaseModelConfig.BaseUrl, ctx, req)
		return resp, fallbackRes, err
	}, func(resp *ExtendedChatCompletionStream, err error) {})
}

func createChatCompletionStreamExtended(
	modelConfig *shared.ModelRoleConfig,
	client ClientInfo,
	baseUrl string,
	ctx context.Context,
	extendedReq types.ExtendedChatCompletionRequest,
) (*ExtendedChatCompletionStream, error) {
	// var openaiReq *types.ExtendedOpenAIChatCompletionRequest // This variable was declared but not used after refactoring.
	// The logic for using extendedReq.ToOpenAI() is handled by actualReqData below.
	if modelConfig.BaseModelConfig.Provider == shared.ModelProviderOpenAI && !modelConfig.BaseModelConfig.UsesOpenAIResponsesAPI {
		// openaiReq = extendedReq.ToOpenAI() // This assignment is handled by actualReqData
		log.Println("Creating chat completion stream with direct OpenAI provider request")
	}

	var httpReq *http.Request
	var err error // Declare err here to be used by all branches for json marshaling and request creation

	if modelConfig.BaseModelConfig.Provider == shared.ModelProviderJules {
		log.Println("Creating chat completion stream with Jules provider request")
		jsonBody, err_marshal := json.Marshal(extendedReq)
		if err_marshal != nil {
			return nil, fmt.Errorf("error marshaling Jules request: %w", err_marshal)
		}

		url := modelConfig.BaseModelConfig.BaseUrl + "/chat/completions"
		httpReq, err = http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("error creating Jules request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")
		httpReq.Header.Set("Cache-Control", "no-cache")
		httpReq.Header.Set("Connection", "keep-alive")
		httpReq.Header.Set("Authorization", "Bearer "+client.ApiKey)
		// No OrgId or OpenRouter specific headers for Jules
	} else {
		// Existing logic for OpenAI, OpenRouter, and other potential custom providers
		var jsonBody []byte
		var actualReqData interface{} // Use interface to hold either extendedReq or openaiReq

		if modelConfig.BaseModelConfig.Provider == shared.ModelProviderOpenAI && !modelConfig.BaseModelConfig.UsesOpenAIResponsesAPI {
			actualReqData = extendedReq.ToOpenAI() // Convert to OpenAI specific request
			log.Println("Creating chat completion stream with direct OpenAI provider request (original body)")
		} else {
			actualReqData = extendedReq
			log.Printf("Creating chat completion stream for provider %s (extendedReq body)", modelConfig.BaseModelConfig.Provider)
		}

		jsonBody, err = json.Marshal(actualReqData)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request for provider %s: %w", modelConfig.BaseModelConfig.Provider, err)
		}

		var url string
		if modelConfig.BaseModelConfig.UsesOpenAIResponsesAPI {
			url = baseUrl + "/responses"
		} else {
			url = baseUrl + "/chat/completions"
		}

		httpReq, err = http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("error creating request for provider %s: %w", modelConfig.BaseModelConfig.Provider, err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")
		httpReq.Header.Set("Cache-Control", "no-cache")
		httpReq.Header.Set("Connection", "keep-alive")
		httpReq.Header.Set("Authorization", "Bearer "+client.ApiKey)

		if modelConfig.BaseModelConfig.Provider == shared.ModelProviderOpenAI && client.OrgId != "" {
			httpReq.Header.Set("OpenAI-Organization", client.OrgId)
		}
		if modelConfig.BaseModelConfig.Provider == shared.ModelProviderOpenRouter {
			addOpenRouterHeaders(httpReq)
		}
	}

	// Common logic for sending request and processing response
	// log.Println("request jsonBody", string(jsonBody)) // jsonBody might be out of scope here if defined inside blocks

	httpResp, err := httpClient.Do(httpReq) //nolint:bodyclose // body is closed in stream.Close()
	if err != nil {
		return nil, fmt.Errorf("error making request to provider %s: %w", modelConfig.BaseModelConfig.Provider, err)
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusBadRequest {
		defer httpResp.Body.Close()
		body, errRead := io.ReadAll(httpResp.Body)
		if errRead != nil { // Corrected the error variable name here
			return nil, fmt.Errorf("error reading error response from provider %s: %w", modelConfig.BaseModelConfig.Provider, errRead)
		}
		return nil, &HTTPError{
			StatusCode: httpResp.StatusCode,
			Body:       string(body),
			Header:     httpResp.Header.Clone(), // retain Retry-After etc.
		}
	}

	// Log response headers
	// log.Println("Response headers:")
	// for key, values := range httpResp.Header {
	// 	log.Printf("%s: %v\n", key, values)
	// }

	reader := &StreamReader[types.ExtendedChatCompletionStreamResponse]{
		reader:             bufio.NewReader(httpResp.Body),
		response:           httpResp,
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

		// log.Println("\n\n--- stream data:\n", data, "\n\n")

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

	// this isn't working yet
	// if modelConfig.BaseModelConfig.UsesOpenAIResponsesAPI {
	// 	log.Println("Using OpenAI Responses API")
	// 	input := make([]types.ExtendedChatMessage, 0)

	// 	for _, msg := range req.Messages {
	// 		// isDevRole := false
	// 		isAssistantRole := false
	// 		if msg.Role == openai.ChatMessageRoleSystem {
	// 			log.Println("Changing role of system message to 'developer'")
	// 			msg.Role = openai.ChatMessageRoleDeveloper
	// 			// isDevRole = true
	// 		} else if msg.Role == openai.ChatMessageRoleAssistant {
	// 			isAssistantRole = true
	// 		}

	// 		for j, part := range msg.Content {
	// 			if part.Type == "text" {
	// 				if isAssistantRole {
	// 					part.Type = "output_text"
	// 				} else {
	// 					part.Type = "input_text"
	// 				}
	// 			}

	// 			msg.Content[j] = part
	// 		}

	// 		input = append(input, msg)
	// 	}

	// 	// stop is not supported for the responses API
	// 	if req.Stop != nil {
	// 		req.Stop = nil
	// 	}

	// 	if req.StreamOptions != nil {
	// 		req.StreamOptions = nil
	// 	}

	// 	req.Input = input
	// 	req.Messages = nil

	// 	log.Println("req.Input", len(req.Input))
	// 	log.Println("req.Messages", len(req.Messages))
	// }

	if modelConfig.BaseModelConfig.RoleParamsDisabled {
		log.Println("Role params disabled - setting temperature and top p to 1")
		req.Temperature = 1
		req.TopP = 1
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
