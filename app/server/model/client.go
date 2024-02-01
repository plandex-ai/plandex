package model

import (
	"time"

	"github.com/sashabaranov/go-openai"
)

const OPENAI_STREAM_CHUNK_TIMEOUT = time.Duration(30) * time.Second

func NewClient(apiKey string) *openai.Client {
	return openai.NewClient(apiKey)
}
