package types

import (
	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

type CacheControlType string

const (
	CacheControlTypeEphemeral CacheControlType = "ephemeral"
)

type CacheControlSpec struct {
	Type CacheControlType `json:"type"`
}

type ExtendedChatMessagePart struct {
	Type         openai.ChatMessagePartType  `json:"type"`
	Text         string                      `json:"text,omitempty"`
	ImageURL     *openai.ChatMessageImageURL `json:"image_url,omitempty"`
	CacheControl *CacheControlSpec           `json:"cache_control,omitempty"`
}

type ExtendedChatMessage struct {
	Role    string                    `json:"role"`
	Content []ExtendedChatMessagePart `json:"content"`
}

func (msg *ExtendedChatMessage) ToOpenAI() *openai.ChatCompletionMessage {
	// If there's only one part and it's text, use simple Content field
	if len(msg.Content) == 1 && msg.Content[0].Type == "text" {
		return &openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content[0].Text,
		}
	}

	// Otherwise, use MultiContent for multiple parts or non-text content
	parts := make([]openai.ChatMessagePart, len(msg.Content))
	for i, part := range msg.Content {
		parts[i] = openai.ChatMessagePart{
			Type:     part.Type,
			Text:     part.Text,
			ImageURL: part.ImageURL,
		}
	}

	return &openai.ChatCompletionMessage{
		Role:         msg.Role,
		MultiContent: parts,
	}
}

type OpenAIPrediction struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type OpenRouterProviderConfig struct {
	Order          []string `json:"order"`
	AllowFallbacks bool     `json:"allow_fallbacks"`
}

type ExtendedChatCompletionRequest struct {
	// copied from openai.ChatCompletionRequest
	Model    string                `json:"model"`
	Messages []ExtendedChatMessage `json:"messages"`
	// MaxTokens The maximum number of tokens that can be generated in the chat completion.
	// This value can be used to control costs for text generated via API.
	// This value is now deprecated in favor of max_completion_tokens, and is not compatible with o1 series models.
	// refs: https://platform.openai.com/docs/api-reference/chat/create#chat-create-max_tokens
	MaxTokens int `json:"max_tokens,omitempty"`
	// MaxCompletionTokens An upper bound for the number of tokens that can be generated for a completion,
	// including visible output tokens and reasoning tokens https://platform.openai.com/docs/guides/reasoning
	MaxCompletionTokens int                                  `json:"max_completion_tokens,omitempty"`
	Temperature         float32                              `json:"temperature,omitempty"`
	TopP                float32                              `json:"top_p,omitempty"`
	N                   int                                  `json:"n,omitempty"`
	Stream              bool                                 `json:"stream,omitempty"`
	Stop                []string                             `json:"stop,omitempty"`
	PresencePenalty     float32                              `json:"presence_penalty,omitempty"`
	ResponseFormat      *openai.ChatCompletionResponseFormat `json:"response_format,omitempty"`
	Seed                *int                                 `json:"seed,omitempty"`
	FrequencyPenalty    float32                              `json:"frequency_penalty,omitempty"`
	// LogitBias is must be a token id string (specified by their token ID in the tokenizer), not a word string.
	// incorrect: `"logit_bias":{"You": 6}`, correct: `"logit_bias":{"1639": 6}`
	// refs: https://platform.openai.com/docs/api-reference/chat/create#chat/create-logit_bias
	LogitBias map[string]int `json:"logit_bias,omitempty"`
	// LogProbs indicates whether to return log probabilities of the output tokens or not.
	// If true, returns the log probabilities of each output token returned in the content of message.
	// This option is currently not available on the gpt-4-vision-preview model.
	LogProbs bool `json:"logprobs,omitempty"`
	// TopLogProbs is an integer between 0 and 5 specifying the number of most likely tokens to return at each
	// token position, each with an associated log probability.
	// logprobs must be set to true if this parameter is used.
	TopLogProbs int    `json:"top_logprobs,omitempty"`
	User        string `json:"user,omitempty"`
	// Deprecated: use Tools instead.
	Functions []openai.FunctionDefinition `json:"functions,omitempty"`
	// Deprecated: use ToolChoice instead.
	FunctionCall any           `json:"function_call,omitempty"`
	Tools        []openai.Tool `json:"tools,omitempty"`
	// This can be either a string or an ToolChoice object.
	ToolChoice any `json:"tool_choice,omitempty"`
	// Options for streaming response. Only set this when you set stream: true.
	StreamOptions *openai.StreamOptions `json:"stream_options,omitempty"`
	// Disable the default behavior of parallel tool calls by setting it: false.
	ParallelToolCalls any `json:"parallel_tool_calls,omitempty"`
	// Store can be set to true to store the output of this completion request for use in distillations and evals.
	// https://platform.openai.com/docs/api-reference/chat/create#chat-create-store
	Store bool `json:"store,omitempty"`
	// Metadata to store with the completion.
	Metadata map[string]string `json:"metadata,omitempty"`

	Prediction      *OpenAIPrediction         `json:"prediction,omitempty"`
	Provider        *OpenRouterProviderConfig `json:"provider,omitempty"`
	ReasoningEffort *shared.ReasoningEffort   `json:"reasoning_effort,omitempty"`
}

func (req *ExtendedChatCompletionRequest) ToOpenAI() *openai.ChatCompletionRequest {
	openaiMessages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		openaiMessages[i] = *msg.ToOpenAI()
	}

	return &openai.ChatCompletionRequest{
		Model:               req.Model,
		Messages:            openaiMessages,
		MaxTokens:           req.MaxTokens,
		MaxCompletionTokens: req.MaxCompletionTokens,
		Temperature:         req.Temperature,
		TopP:                req.TopP,
		N:                   req.N,
		Stream:              req.Stream,
		Stop:                req.Stop,
		PresencePenalty:     req.PresencePenalty,
		ResponseFormat:      req.ResponseFormat,
		Seed:                req.Seed,
		FrequencyPenalty:    req.FrequencyPenalty,
		LogitBias:           req.LogitBias,
		LogProbs:            req.LogProbs,
		TopLogProbs:         req.TopLogProbs,
		User:                req.User,
		Functions:           req.Functions,
		FunctionCall:        req.FunctionCall,
		Tools:               req.Tools,
		ToolChoice:          req.ToolChoice,
		StreamOptions:       req.StreamOptions,
		ParallelToolCalls:   req.ParallelToolCalls,
		Store:               req.Store,
		Metadata:            req.Metadata,
	}
}
