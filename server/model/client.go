package model

import (
	"os"
	"time"

	"github.com/sashabaranov/go-openai"
)

const OPENAI_STREAM_CHUNK_TIMEOUT = time.Duration(30) * time.Second

var Client *openai.Client

func init() {
	Client = openai.NewClient(os.Getenv("OPENAI_API_KEY"))
}
