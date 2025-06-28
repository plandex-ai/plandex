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
	MAX_RETRIES_WITHOUT_FALLBACK         = 3
	MAX_RETRY_DELAY_SECONDS              = 10
)

var httpClient = &http.Client{}

type ClientInfo struct {
	Client         *openai.Client
	ProviderConfig shared.ModelProviderConfigSchema
	ApiKey         string
	OpenAIOrgId    string
}

func InitClients(authVars map[string]string, settings *shared.PlanSettings) map[string]ClientInfo {
	clients := make(map[string]ClientInfo)
	providers := shared.GetProvidersForAuthVars(authVars, settings)

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
	settings *shared.PlanSettings,
	ctx context.Context,
	req types.ExtendedChatCompletionRequest,
) (*ExtendedChatCompletionStream, error) {
	providerComposite := modelConfig.GetProviderComposite(authVars, settings)
	_, ok := clients[providerComposite]
	if !ok {
		return nil, fmt.Errorf("client not found for provider composite: %s", providerComposite)
	}

	baseModelConfig := modelConfig.GetBaseModelConfig(authVars, settings)

	// ensure the model name is set correctly on fallbacks
	req.Model = baseModelConfig.ModelName

	resolveReq(&req, modelConfig, baseModelConfig, settings)

	// choose the fastest provider by latency/throughput on openrouter
	if baseModelConfig.Provider == shared.ModelProviderOpenRouter {
		if !strings.HasSuffix(string(req.Model), ":nitro") && !strings.HasSuffix(string(req.Model), ":free") && !strings.HasSuffix(string(req.Model), ":floor") {
			req.Model += ":nitro"
		}
	}

	if baseModelConfig.ReasoningBudget > 0 {
		req.ReasoningConfig = &types.ReasoningConfig{
			MaxTokens: baseModelConfig.ReasoningBudget,
			Exclude:   !baseModelConfig.IncludeReasoning || baseModelConfig.HideReasoning,
		}
	} else if baseModelConfig.ReasoningEffortEnabled {
		req.ReasoningConfig = &types.ReasoningConfig{
			Effort:  shared.ReasoningEffort(baseModelConfig.ReasoningEffort),
			Exclude: !baseModelConfig.IncludeReasoning || baseModelConfig.HideReasoning,
		}
	} else if baseModelConfig.IncludeReasoning {
		req.ReasoningConfig = &types.ReasoningConfig{
			Exclude: baseModelConfig.HideReasoning,
		}
	}

	return withStreamingRetries(ctx, func(numTotalRetry int, didProviderFallback bool, modelErr *shared.ModelError) (*ExtendedChatCompletionStream, shared.FallbackResult, error) {
		fallbackRes := modelConfig.GetFallbackForModelError(numTotalRetry, didProviderFallback, modelErr, authVars, settings)
		resolvedModelConfig := fallbackRes.ModelRoleConfig

		if resolvedModelConfig == nil {
			return nil, fallbackRes, fmt.Errorf("model config is nil")
		}

		providerComposite := resolvedModelConfig.GetProviderComposite(authVars, settings)

		baseModelConfig := resolvedModelConfig.GetBaseModelConfig(authVars, settings)

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

		resp, err := createChatCompletionStreamExtended(resolvedModelConfig, opClient, authVars, settings, ctx, req)
		return resp, fallbackRes, err
	}, func(resp *ExtendedChatCompletionStream, err error) {})
}

func createChatCompletionStreamExtended(
	modelConfig *shared.ModelRoleConfig,
	client ClientInfo,
	authVars map[string]string,
	settings *shared.PlanSettings,
	ctx context.Context,
	extendedReq types.ExtendedChatCompletionRequest,
) (*ExtendedChatCompletionStream, error) {
	baseModelConfig := modelConfig.GetBaseModelConfig(authVars, settings)

	// ensure the model name is set correctly on fallbacks
	extendedReq.Model = baseModelConfig.ModelName

	var openaiReq *types.ExtendedOpenAIChatCompletionRequest
	if baseModelConfig.Provider == shared.ModelProviderOpenAI {
		openaiReq = extendedReq.ToOpenAI()
		log.Println("Creating chat completion stream with direct OpenAI provider request")
	}

	switch baseModelConfig.Provider {
	case shared.ModelProviderGoogleVertex:
		if authVars["VERTEXAI_PROJECT"] != "" {
			extendedReq.VertexProject = authVars["VERTEXAI_PROJECT"]
		}
		if authVars["VERTEXAI_LOCATION"] != "" {
			extendedReq.VertexLocation = authVars["VERTEXAI_LOCATION"]
		}
		if authVars["GOOGLE_APPLICATION_CREDENTIALS"] != "" {
			extendedReq.VertexCredentials = authVars["GOOGLE_APPLICATION_CREDENTIALS"]
		}
	case shared.ModelProviderAzureOpenAI:
		if authVars["AZURE_API_BASE"] != "" {
			extendedReq.LiteLLMApiBase = authVars["AZURE_API_BASE"]
		}
		if authVars["AZURE_API_VERSION"] != "" {
			extendedReq.AzureApiVersion = authVars["AZURE_API_VERSION"]
		}

		if authVars["AZURE_DEPLOYMENTS_MAP"] != "" {
			var azureDeploymentsMap map[string]string
			err := json.Unmarshal([]byte(authVars["AZURE_DEPLOYMENTS_MAP"]), &azureDeploymentsMap)
			if err != nil {
				return nil, fmt.Errorf("error unmarshalling AZURE_DEPLOYMENTS_MAP: %w", err)
			}
			modelName := string(extendedReq.Model)
			modelName = strings.ReplaceAll(modelName, "azure/", "")

			deploymentName, ok := azureDeploymentsMap[modelName]
			if ok {
				log.Println("azure - deploymentName", deploymentName)
				modelName = "azure/" + deploymentName
				extendedReq.Model = shared.ModelName(modelName)
			}
		}

		// azure uses 'reasoning_config' instead of 'reasoning' like direct openai api
		if extendedReq.ReasoningConfig != nil {
			extendedReq.AzureReasoningEffort = extendedReq.ReasoningConfig.Effort
			extendedReq.ReasoningConfig = nil
		}
	case shared.ModelProviderAmazonBedrock:
		if authVars["AWS_ACCESS_KEY_ID"] != "" {
			extendedReq.BedrockAccessKeyId = authVars["AWS_ACCESS_KEY_ID"]
		}
		if authVars["AWS_SECRET_ACCESS_KEY"] != "" {
			extendedReq.BedrockSecretAccessKey = authVars["AWS_SECRET_ACCESS_KEY"]
		}
		if authVars["AWS_SESSION_TOKEN"] != "" {
			extendedReq.BedrockSessionToken = authVars["AWS_SESSION_TOKEN"]
		}
		if authVars["AWS_REGION"] != "" {
			extendedReq.BedrockRegion = authVars["AWS_REGION"]
		}
		if authVars["AWS_INFERENCE_PROFILE_ARN"] != "" {
			extendedReq.BedrockInferenceProfileArn = authVars["AWS_INFERENCE_PROFILE_ARN"]
		}

	case shared.ModelProviderOllama:
		if os.Getenv("OLLAMA_BASE_URL") != "" {
			extendedReq.LiteLLMApiBase = os.Getenv("OLLAMA_BASE_URL")
		}
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

func resolveReq(req *types.ExtendedChatCompletionRequest, modelConfig *shared.ModelRoleConfig, baseModelConfig *shared.BaseModelConfig, settings *shared.PlanSettings) {
	// if system prompt is disabled, change the role of the system message to user
	if modelConfig.GetSharedBaseConfig(settings).SystemPromptDisabled {
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

	if modelConfig.GetSharedBaseConfig(settings).RoleParamsDisabled {
		log.Println("Role params disabled - setting temperature and top p to 0")
		req.Temperature = 0
		req.TopP = 0
	}

	if baseModelConfig.Provider == shared.ModelProviderOllama {
		// ollama doesn't support temperature or top p params
		log.Println("Ollama - clearing temperature and top p")
		req.Temperature = 0
		req.TopP = 0

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
