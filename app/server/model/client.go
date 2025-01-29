package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/plandex/plandex/shared"
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

type OpenAIPrediction struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type ExtendedChatCompletionRequest struct {
	*openai.ChatCompletionRequest
	Prediction OpenAIPrediction `json:"prediction,omitempty"`
}

func CreateChatCompletionStreamWithRetries(
	clients map[string]ClientInfo,
	modelConfig *shared.ModelRoleConfig,
	ctx context.Context,
	req openai.ChatCompletionRequest,
) (*openai.ChatCompletionStream, error) {
	client, ok := clients[modelConfig.BaseModelConfig.ApiKeyEnvVar]

	if !ok {
		return nil, fmt.Errorf("client not found for api key env var: %s", modelConfig.BaseModelConfig.ApiKeyEnvVar)
	}

	resolveReq(&req, modelConfig)

	for _, msg := range req.Messages {
		log.Println("Message role:", msg.Role)
	}

	return withRetries(ctx, func() (*openai.ChatCompletionStream, error) {
		return client.Client.CreateChatCompletionStream(ctx, req)
	})
}

func CreateChatCompletionWithRetries[T openai.ChatCompletionRequest | ExtendedChatCompletionRequest](
	clients map[string]ClientInfo,
	modelConfig *shared.ModelRoleConfig,
	ctx context.Context,
	req T,
) (openai.ChatCompletionResponse, error) {
	client, ok := clients[modelConfig.BaseModelConfig.ApiKeyEnvVar]

	if !ok {
		return openai.ChatCompletionResponse{}, fmt.Errorf("client not found for api key env var: %s", modelConfig.BaseModelConfig.ApiKeyEnvVar)
	}
	var baseReq *openai.ChatCompletionRequest
	var fn func() (openai.ChatCompletionResponse, error)

	if normalReq, ok := any(req).(openai.ChatCompletionRequest); ok {
		baseReq = &normalReq
		fn = func() (openai.ChatCompletionResponse, error) {
			return client.Client.CreateChatCompletion(ctx, normalReq)
		}
	} else if extendedReq, ok := any(req).(ExtendedChatCompletionRequest); ok {
		baseReq = extendedReq.ChatCompletionRequest
		fn = func() (openai.ChatCompletionResponse, error) {
			return createChatCompletionExtended(clients, modelConfig, ctx, extendedReq)
		}
	} else {
		log.Println("Invalid request type")
		log.Println("Request type:", reflect.TypeOf(req))
		return openai.ChatCompletionResponse{}, fmt.Errorf("invalid request type")
	}

	resolveReq(baseReq, modelConfig)

	return withRetries(ctx, fn)
}

func createChatCompletionExtended(
	clients map[string]ClientInfo,
	modelConfig *shared.ModelRoleConfig,
	ctx context.Context,
	extendedReq ExtendedChatCompletionRequest,
) (openai.ChatCompletionResponse, error) {
	client, ok := clients[modelConfig.BaseModelConfig.ApiKeyEnvVar]

	if !ok {
		return openai.ChatCompletionResponse{}, fmt.Errorf("client not found for api key env var: %s", modelConfig.BaseModelConfig.ApiKeyEnvVar)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", modelConfig.BaseModelConfig.BaseUrl+"/chat/completions", nil)
	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+client.ApiKey)
	if client.OrgId != "" {
		req.Header.Set("OpenAI-Organization", client.OrgId)
	}

	// Add body
	jsonBody, err := json.Marshal(extendedReq)
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
	log.Println("Response body:", string(body))

	var response openai.ChatCompletionResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}

	return response, nil
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

func resolveReq(req *openai.ChatCompletionRequest, modelConfig *shared.ModelRoleConfig) {
	log.Println("Resolving request")
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

	log.Println("Resolved request")
}

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
