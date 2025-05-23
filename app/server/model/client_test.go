package model

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"plandex-server/types"
	"strings"
	"testing"
	"time"

	shared "plandex-shared"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mockJulesAPIKeyValue = "test-jules-api-key"
)

func TestCreateChatCompletionStream_JulesProvider_Success(t *testing.T) {
	// --- Mock Server Setup ---
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Verify Method and Path
		assert.Equal(t, http.MethodPost, r.Method, "Expected POST request")
		assert.Equal(t, "/chat/completions", r.URL.Path, "Expected path /chat/completions")

		// 2. Verify Authorization Header
		expectedAuthHeader := "Bearer " + mockJulesAPIKeyValue
		assert.Equal(t, expectedAuthHeader, r.Header.Get("Authorization"), "Incorrect Authorization header")

		// 3. Verify Request Body
		var reqBody types.ExtendedChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err, "Failed to decode request body")
		defer r.Body.Close()

		assert.Equal(t, "jules-v1", string(reqBody.Model), "Request body has incorrect model name") // Cast to string
		require.Len(t, reqBody.Messages, 1, "Expected 1 message in request body")
		assert.Equal(t, "Hello Jules!", reqBody.Messages[0].Content[0].Text, "Incorrect message content in request body")

		// 4. Respond with SSE Stream
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("Streaming unsupported!")
		}

		// Chunk 1
		chunk1 := types.ExtendedChatCompletionStreamResponse{
			ID:      "chatcmpl-mock-jules-1",
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   "jules-v1",
			Choices: []types.ExtendedChatCompletionStreamChoice{
				{
					Index: 0,
					Delta: types.ExtendedChatCompletionStreamChoiceDelta{Role: "assistant", Content: "Response "},
				},
			},
		}
		jsonChunk1, _ := json.Marshal(chunk1)
		fmt.Fprintf(w, "data: %s\n\n", string(jsonChunk1))
		flusher.Flush()
		time.Sleep(10 * time.Millisecond) // Simulate network delay

		// Chunk 2
		chunk2 := types.ExtendedChatCompletionStreamResponse{
			ID:      "chatcmpl-mock-jules-2",
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   "jules-v1",
			Choices: []types.ExtendedChatCompletionStreamChoice{
				{
					Index: 0,
					Delta: types.ExtendedChatCompletionStreamChoiceDelta{Content: "from Jules!"},
				},
			},
		}
		jsonChunk2, _ := json.Marshal(chunk2)
		fmt.Fprintf(w, "data: %s\n\n", string(jsonChunk2))
		flusher.Flush()
		time.Sleep(10 * time.Millisecond)

		// Done event
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer mockServer.Close()

	// --- Go Test Logic ---
	t.Setenv("JULES_API_KEY", mockJulesAPIKeyValue)

	// Prepare ClientInfo
	// In a real scenario, InitClients would be called. Here we manually construct.
	// The key for the clients map is the ApiKeyEnvVar ("JULES_API_KEY")
	clientsMap := map[string]ClientInfo{
		"JULES_API_KEY": {
			ApiKey:   mockJulesAPIKeyValue,
			Endpoint: mockServer.URL, // This is the base URL for the provider, used by createChatCompletionStreamExtended
		},
	}

	// Prepare ModelRoleConfig
	// Start with a copy of JulesV1BaseConfig and override BaseUrl
	julesModelBaseConfig := shared.JulesV1BaseConfig 
	julesModelBaseConfig.BaseUrl = mockServer.URL // IMPORTANT: Override to use mock server

	modelConfig := &shared.ModelRoleConfig{
		Role:            shared.ModelRolePlanner, // Example role
		BaseModelConfig: julesModelBaseConfig,
		Temperature:     0.7,
		TopP:            1.0,
	}
	
	// Prepare ExtendedChatCompletionRequest
	apiRequest := types.ExtendedChatCompletionRequest{
		Model: "jules-v1", // This should match what JulesV1BaseConfig uses or what the server expects
		Messages: []types.ExtendedChatMessage{
			{Role: "user", Content: []types.ExtendedChatMessagePart{{Type: "text", Text: "Hello Jules!"}}},
		},
		Stream: true,
	}

	// Call CreateChatCompletionStream
	// Note: CreateChatCompletionStream eventually calls createChatCompletionStreamExtended.
	// The `baseUrl` argument to `createChatCompletionStreamExtended` comes from `resolvedModelConfig.BaseModelConfig.BaseUrl`.
	stream, err := CreateChatCompletionStream(clientsMap, modelConfig, context.Background(), apiRequest)
	require.NoError(t, err, "CreateChatCompletionStream returned an error")
	require.NotNil(t, stream, "CreateChatCompletionStream returned a nil stream")
	defer stream.Close()

	// Consume Stream
	var fullResponse strings.Builder
	var receivedChunks []types.ExtendedChatCompletionStreamResponse

	for {
		response, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break // [DONE] event
		}
		require.NoError(t, recvErr, "stream.Recv() returned an error")
		
		receivedChunks = append(receivedChunks, *response)
		if len(response.Choices) > 0 {
			// Corrected: Delta.Content is a string for stream responses
			fullResponse.WriteString(response.Choices[0].Delta.Content)
		}
	}

	// Assertions
	assert.Equal(t, "Response from Jules!", fullResponse.String(), "Full streamed response does not match")
	require.Len(t, receivedChunks, 2, "Expected 2 data chunks before [DONE]")
	// Corrected: Delta.Content is a string
	assert.Equal(t, "Response ", receivedChunks[0].Choices[0].Delta.Content)
	assert.Equal(t, "from Jules!", receivedChunks[1].Choices[0].Delta.Content)
}

func TestCreateChatCompletionStream_JulesProvider_HttpError(t *testing.T) {
	// --- Mock Server Setup ---
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Method and Path
		assert.Equal(t, http.MethodPost, r.Method, "Expected POST request")
		assert.Equal(t, "/chat/completions", r.URL.Path, "Expected path /chat/completions")

		// Respond with an error
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"error": {"message": "Jules mock internal server error"}}`)
	}))
	defer mockServer.Close()

	// --- Go Test Logic ---
	t.Setenv("JULES_API_KEY", mockJulesAPIKeyValue)

	clientsMap := map[string]ClientInfo{
		"JULES_API_KEY": {
			ApiKey:   mockJulesAPIKeyValue,
			Endpoint: mockServer.URL, 
		},
	}

	julesModelBaseConfig := shared.JulesV1BaseConfig
	julesModelBaseConfig.BaseUrl = mockServer.URL 

	modelConfig := &shared.ModelRoleConfig{
		Role:            shared.ModelRolePlanner,
		BaseModelConfig: julesModelBaseConfig,
	}

	apiRequest := types.ExtendedChatCompletionRequest{
		Model: "jules-v1",
		Messages: []types.ExtendedChatMessage{
			{Role: "user", Content: []types.ExtendedChatMessagePart{{Type: "text", Text: "Trigger error"}}},
		},
		Stream: true,
	}

	// Call CreateChatCompletionStream
	stream, err := CreateChatCompletionStream(clientsMap, modelConfig, context.Background(), apiRequest)

	// Assertions
	require.Error(t, err, "CreateChatCompletionStream should have returned an error")
	assert.Nil(t, stream, "Stream should be nil on error")

	// Check if the error is of the expected type (HTTPError, if defined and used, or check message)
	// For this test, we'll check for a generic error message that implies an HTTP issue.
	// In a real system, you might have a custom error type like `*HTTPError`.
	// The actual error returned by `createChatCompletionStreamExtended` for HTTP errors is `*HTTPError`.
	httpErr, ok := err.(*HTTPError) // Assuming HTTPError is accessible or defined in the same package.
	                               // If not, adjust to check err.Error() content.
	if ok {
		assert.Equal(t, http.StatusInternalServerError, httpErr.StatusCode, "Error status code does not match")
		assert.Contains(t, httpErr.Body, "Jules mock internal server error", "Error body does not match")
	} else {
		// Fallback if HTTPError type assertion fails (e.g. not exported or different error wrapping)
		assert.Contains(t, err.Error(), "status code 500", "Error message should indicate an HTTP 500 error")
	}
}
