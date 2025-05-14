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

	"github.com/davecgh/go-spew/spew"
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
	Client         *openai.Client
	ProviderConfig shared.ModelProviderConfigSchema
	ApiKey         string
	OpenAIOrgId    string
}

func InitClients(authVars map[string]string) map[string]ClientInfo {
	clients := make(map[string]ClientInfo)
	providers := shared.GetProvidersForAuthVars(authVars)

	for _, provider := range providers {
		clients[provider.ToComposite()] = newClient(provider, authVars)
	}

	return clients
}

func newClient(providerConfig shared.ModelProviderConfigSchema, authVars map[string]string) ClientInfo {
	var apiKey string
	if providerConfig.ApiKeyEnvVar != "" {
		apiKey = authVars[providerConfig.ApiKeyEnvVar]
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = providerConfig.BaseUrl

	var openAIOrgId string
	if providerConfig.Provider == shared.ModelProviderOpenAI && authVars["OPENAI_ORG_ID"] != "" {
		openAIOrgId = authVars["OPENAI_ORG_ID"]
		config.OrgID = openAIOrgId
	}

	return ClientInfo{
		Client:         openai.NewClientWithConfig(config),
		ApiKey:         apiKey,
		ProviderConfig: providerConfig,
		OpenAIOrgId:    openAIOrgId,
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
	authVars map[string]string,
	modelConfig *shared.ModelRoleConfig,
	ctx context.Context,
	req types.ExtendedChatCompletionRequest,
) (*ExtendedChatCompletionStream, error) {
	providerComposite := modelConfig.GetProviderComposite(authVars)
	_, ok := clients[providerComposite]
	if !ok {
		return nil, fmt.Errorf("client not found for provider composite: %s", providerComposite)
	}

	baseModelConfig := modelConfig.GetBaseModelConfig(authVars)

	// ensure the model name is set correctly on fallbacks
	req.Model = baseModelConfig.ModelName

	resolveReq(&req, modelConfig)

	// choose the fastest provider by latency/throughput on openrouter
	if baseModelConfig.Provider == shared.ModelProviderOpenRouter {
		req.Model += ":nitro"
	}

	if baseModelConfig.IncludeReasoning {
		req.IncludeReasoning = true
	}

	return withStreamingRetries(ctx, func(numTotalRetry int, modelErr *shared.ModelError) (*ExtendedChatCompletionStream, shared.FallbackResult, error) {
		fallbackRes := modelConfig.GetFallbackForModelError(numTotalRetry, modelErr)
		resolvedModelConfig := fallbackRes.ModelRoleConfig

		if resolvedModelConfig == nil {
			return nil, fallbackRes, fmt.Errorf("model config is nil")
		}

		providerComposite := resolvedModelConfig.GetProviderComposite(authVars)

		baseModelConfig := resolvedModelConfig.GetBaseModelConfig(authVars)

		opClient, ok := clients[providerComposite]

		if !ok {
			return nil, fallbackRes, fmt.Errorf("client not found for provider composite: %s", providerComposite)
		}

		if modelErr != nil && modelErr.Kind == shared.ErrCacheSupport {
			for i := range req.Messages {
				for j := range req.Messages[i].Content {
					if req.Messages[i].Content[j].CacheControl != nil {
						req.Messages[i].Content[j].CacheControl = nil
					}
				}
			}
		}

		modelConfig = resolvedModelConfig

		log.Println("createChatCompletionStreamExtended - modelConfig")
		spew.Dump(map[string]interface{}{
			"modelConfig.ModelId":      baseModelConfig.ModelId,
			"modelConfig.ModelTag":     baseModelConfig.ModelTag,
			"modelConfig.ModelName":    baseModelConfig.ModelName,
			"modelConfig.Provider":     baseModelConfig.Provider,
			"modelConfig.BaseUrl":      baseModelConfig.BaseUrl,
			"modelConfig.ApiKeyEnvVar": baseModelConfig.ApiKeyEnvVar,
		})

		resp, err := createChatCompletionStreamExtended(resolvedModelConfig, opClient, authVars, ctx, req)
		return resp, fallbackRes, err
	}, func(resp *ExtendedChatCompletionStream, err error) {})
}

func createChatCompletionStreamExtended(
	modelConfig *shared.ModelRoleConfig,
	client ClientInfo,
	authVars map[string]string,
	ctx context.Context,
	extendedReq types.ExtendedChatCompletionRequest,
) (*ExtendedChatCompletionStream, error) {
	baseModelConfig := modelConfig.GetBaseModelConfig(authVars)

	// ensure the model name is set correctly on fallbacks
	extendedReq.Model = baseModelConfig.ModelName

	var openaiReq *types.ExtendedOpenAIChatCompletionRequest
	if baseModelConfig.Provider == shared.ModelProviderOpenAI {
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
	baseUrl := baseModelConfig.BaseUrl
	url := baseUrl + "/chat/completions"

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set required headers for streaming
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// some providers send api key in the body, some in the header
	// some use other auth methods and so don't have a simple api key
	if client.ApiKey != "" {
		req.Header.Set("Authorization", "Bearer "+client.ApiKey)
	}
	if client.OpenAIOrgId != "" {
		req.Header.Set("OpenAI-Organization", client.OpenAIOrgId)
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
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
			Header:     resp.Header.Clone(), // retain Retry-After etc.
		}
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
	if modelConfig.GetSharedBaseConfig().SystemPromptDisabled {
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

	if modelConfig.GetSharedBaseConfig().RoleParamsDisabled {
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
