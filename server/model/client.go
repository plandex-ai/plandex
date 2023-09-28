package model

import (
	"os"

	"github.com/sashabaranov/go-openai"
)

var client *openai.Client

func init() {
	client = openai.NewClient(os.Getenv("OPENAI_API_KEY"))
}
